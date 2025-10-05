package main

import (
	"context"
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dbehnke/allstar-nexus/backend/api"
	"github.com/dbehnke/allstar-nexus/backend/auth"
	"github.com/dbehnke/allstar-nexus/backend/config"
	"github.com/dbehnke/allstar-nexus/backend/database"
	"github.com/dbehnke/allstar-nexus/backend/middleware"
	"github.com/dbehnke/allstar-nexus/backend/repository"
	"github.com/dbehnke/allstar-nexus/internal/ami"
	"github.com/dbehnke/allstar-nexus/internal/astdb"
	"github.com/dbehnke/allstar-nexus/internal/core"
	"github.com/dbehnke/allstar-nexus/internal/web"
	"go.uber.org/zap"
)

//go:embed all:frontend/out all:vue-dashboard/dist
var frontendFiles embed.FS
var buildVersion = ""
var buildTime = ""

func main() {
	cfg := config.Load()

	// Initialize logger (simple for now)
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Initialize astdb downloader and ensure file exists
	astdbDownloader := astdb.NewDownloader(cfg.AstDBURL, cfg.AstDBPath, cfg.AstDBUpdateHours, logger)
	if err := astdbDownloader.EnsureExists(); err != nil {
		logger.Warn("failed to download astdb, node lookup may not work", zap.Error(err))
	} else {
		// Start auto-updater in background
		astdbDownloader.StartAutoUpdater()
		if count, err := astdbDownloader.GetNodeCount(); err == nil {
			logger.Info("astdb loaded successfully", zap.Int("node_count", count))
		}
	}

	// Open DB
	db, err := database.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("database open error: %v", err)
	}
	defer db.CloseSafe()
	if err := db.Migrate(); err != nil {
		log.Fatalf("migrate error: %v", err)
	}

	// API setup
	apiLayer := api.New(db.DB, cfg.JWTSecret, cfg.TokenTTL)
	apiLayer.SetAstDBPath(cfg.AstDBPath)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", api.Health)
	mux.HandleFunc("/api/dashboard/summary", apiLayer.DashboardSummary)
	limiter := middleware.RateLimiter(cfg.AuthRateLimitRPM)
	mux.Handle("/api/auth/register", limiter(http.HandlerFunc(apiLayer.Register)))
	mux.Handle("/api/auth/login", limiter(http.HandlerFunc(apiLayer.Login)))

	// Repositories for middleware loaders
	userRepo := repository.NewUserRepo(db.DB)
	userLoader := func(email string) (*repository.SafeUser, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		u, err := userRepo.GetByEmail(ctx, email)
		if err != nil || u == nil {
			return nil, err
		}
		return &repository.SafeUser{ID: u.ID, Email: u.Email, Role: u.Role}, nil
	}

	authMW := middleware.Auth(cfg.JWTSecret, userLoader)
	adminMW := middleware.RequireRole("admin", "superadmin")

	mux.Handle("/api/me", authMW(http.HandlerFunc(apiLayer.Me)))
	mux.Handle("/api/admin/summary", authMW(adminMW(http.HandlerFunc(apiLayer.AdminSummary))))

	// Node lookup API - can be public or require auth based on config
	if cfg.AllowAnonDashboard {
		publicLimiter := middleware.RateLimiter(cfg.PublicStatsRateLimitRPM)
		mux.Handle("/api/node-lookup", publicLimiter(http.HandlerFunc(apiLayer.NodeLookup)))
	} else {
		mux.Handle("/api/node-lookup", authMW(http.HandlerFunc(apiLayer.NodeLookup)))
	}

	// RPT and Voter stats APIs - require authentication
	mux.Handle("/api/rpt-stats", authMW(http.HandlerFunc(apiLayer.RPTStats)))
	mux.Handle("/api/voter-stats", authMW(http.HandlerFunc(apiLayer.VoterStats)))

	if cfg.AllowAnonDashboard {
		publicLimiter := middleware.RateLimiter(cfg.PublicStatsRateLimitRPM)
		mux.Handle("/api/link-stats", publicLimiter(http.HandlerFunc(apiLayer.LinkStatsHandler)))
		mux.Handle("/api/link-stats/top", publicLimiter(http.HandlerFunc(apiLayer.TopLinkStatsHandler)))
	} else {
		mux.Handle("/api/link-stats", authMW(http.HandlerFunc(apiLayer.LinkStatsHandler)))
		mux.Handle("/api/link-stats/top", authMW(http.HandlerFunc(apiLayer.TopLinkStatsHandler)))
	}

	// Frontend selection: prefer built Vue dashboard if present, else fallback to Next.js export.
	var staticFS fs.FS
	if vueFS, err := fs.Sub(frontendFiles, "vue-dashboard/dist"); err == nil {
		if entries, e2 := fs.ReadDir(vueFS, "."); e2 == nil && len(entries) > 0 {
			staticFS = vueFS
			log.Printf("serving Vue dashboard (built dist) at /")
		}
	}
	if staticFS == nil { // fallback
		fallback, err := fs.Sub(frontendFiles, "frontend/out")
		if err != nil {
			log.Fatalf("embed fs error: %v", err)
		}
		staticFS = fallback
		log.Printf("serving legacy Next.js export at /")
	}
	mux.Handle("/", http.FileServer(http.FS(staticFS)))

	// AMI + WebSocket wiring (conditional). Always provide a /ws endpoint so the UI never hard-fails.
	var hub *web.Hub
	if cfg.AMIEnabled {
		hub = web.NewHub()
		sm := core.NewStateManager()
		// Propagate build metadata into StateManager so UI can display it
		if buildVersion != "" {
			sm.SetVersion(buildVersion)
		}
		if buildTime != "" {
			sm.SetBuildTime(buildTime)
			cfg.BuildTime = buildTime
		}
		// Seed persisted link stats (if any) so totals survive restarts
		lsRepo := repository.NewLinkStatsRepo(db.DB)
		seedCtx, seedCancel := context.WithTimeout(context.Background(), 2*time.Second)
		if stats, err := lsRepo.GetAll(seedCtx); err == nil && len(stats) > 0 {
			li := make([]core.LinkInfo, 0, len(stats))
			for _, s := range stats {
				cs := time.Now()
				if s.ConnectedSince != nil {
					cs = *s.ConnectedSince
				}
				li = append(li, core.LinkInfo{Node: s.Node, ConnectedSince: cs, LastTxStart: s.LastTxStart, LastTxEnd: s.LastTxEnd, TotalTxSeconds: s.TotalTxSeconds})
			}
			sm.SeedLinkStats(li)
		}
		seedCancel()
		go hub.BroadcastLoop(sm.Updates())
		go hub.TalkerLoop(sm.TalkerEvents())
		go hub.LinkUpdateLoop(sm.LinkUpdates())
		go hub.LinkRemovalLoop(sm.LinkRemovals())
		go hub.LinkTxBatchLoop(sm.LinkTxEvents(), 100*time.Millisecond)
		go hub.HeartbeatLoop(sm, 5*time.Second)
		conn := ami.NewConnector(cfg.AMIHost, cfg.AMIPort, cfg.AMIUser, cfg.AMIPassword, cfg.AMIEvents, cfg.AMIRetryInterval, cfg.AMIRetryMax)
		// Pass AMI connector to API layer for RPT and Voter stats endpoints
		apiLayer.SetAMIConnector(conn)
		ctxAMI, cancelAMI := context.WithCancel(context.Background())
		if err := conn.Start(ctxAMI); err != nil {
			log.Printf("AMI start error: %v", err)
		}
		go sm.Run(conn.Raw())
		if !cfg.DisableLinkPoller && cfg.AMINodeID > 0 {
			// Use EnhancedPoller for XStat/SawStat polling (5 second intervals)
			enhancedPoller := core.NewEnhancedPoller(conn, sm, cfg.AMINodeID, 5*time.Second, logger)
			enhancedPoller.Start(ctxAMI)
			logger.Info("enhanced AMI poller started", zap.Int("node_id", cfg.AMINodeID), zap.Duration("interval", 5*time.Second))
		} else if !cfg.DisableLinkPoller {
			// Fallback to legacy LinkPoller if AMI_NODE_ID not configured
			poller := core.NewLinkPoller(conn, sm, 30*time.Second)
			poller.Start(ctxAMI)
			logger.Warn("using legacy link poller - set AMI_NODE_ID for enhanced features")
		}
		// Persist per-link TX stats on edges
		sm.SetPersistHook(func(list []core.LinkInfo) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			for _, li := range list {
				stat := repository.LinkStat{Node: li.Node, TotalTxSeconds: li.TotalTxSeconds, LastTxStart: li.LastTxStart, LastTxEnd: li.LastTxEnd, ConnectedSince: &li.ConnectedSince}
				_ = lsRepo.Upsert(ctx, stat)
			}
		})
		validator := func(r *http.Request) bool {
			token := r.URL.Query().Get("token")
			if token == "" {
				// allow anonymous if configured
				return cfg.AllowAnonDashboard
			}
			_, _, exp, err := auth.ParseJWT(token, cfg.JWTSecret)
			if err != nil || time.Now().After(exp) {
				return false
			}
			return true
		}
		mux.HandleFunc("/ws", hub.HandleWS(sm, validator))
		defer cancelAMI()
	} else {
		// Fallback: serve a static heartbeat-only websocket with empty state (allows anonymous dashboard to load).
		hub = web.NewHub()
		sm := core.NewStateManager()
		if buildVersion != "" {
			sm.SetVersion(buildVersion)
		}
		if buildTime != "" {
			sm.SetBuildTime(buildTime)
			cfg.BuildTime = buildTime
		}
		validator := func(r *http.Request) bool {
			token := r.URL.Query().Get("token")
			if token == "" {
				return cfg.AllowAnonDashboard
			}
			_, _, exp, err := auth.ParseJWT(token, cfg.JWTSecret)
			if err != nil || time.Now().After(exp) {
				return false
			}
			return true
		}
		mux.HandleFunc("/ws", hub.HandleWS(sm, validator))
		// Heartbeat provides periodic STATUS_UPDATE so client replaces 'Waiting for data'.
		go hub.HeartbeatLoop(sm, 5*time.Second)
	}

	addr := ":" + cfg.Port
	zapLogger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("failed to init zap: %v", err)
	}
	defer zapLogger.Sync()
	loggingMW := middleware.Logging(zapLogger)
	srv := &http.Server{Addr: addr, Handler: loggingMW(mux), ReadTimeout: 10 * time.Second, WriteTimeout: 15 * time.Second}

	// Start server in goroutine
	go func() {
		log.Printf("Allstar Nexus starting on %s (env=%s) build=%s", addr, cfg.Env, cfg.BuildTime)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Wait for termination signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	log.Printf("shutdown signal received, shutting down...")
	ctxShutdown, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctxShutdown); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
		if err := srv.Close(); err != nil {
			log.Printf("server close error: %v", err)
		}
	}
	log.Printf("server stopped cleanly")
}
