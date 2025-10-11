package main

import (
	"context"
	"embed"
	"flag"
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
	"github.com/dbehnke/allstar-nexus/backend/gamification"
	"github.com/dbehnke/allstar-nexus/backend/middleware"
	"github.com/dbehnke/allstar-nexus/backend/models"
	"github.com/dbehnke/allstar-nexus/backend/repository"
	"github.com/dbehnke/allstar-nexus/internal/ami"
	"github.com/dbehnke/allstar-nexus/internal/astdb"
	"github.com/dbehnke/allstar-nexus/internal/core"
	"github.com/dbehnke/allstar-nexus/internal/web"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

//go:embed all:frontend/dist
var frontendFiles embed.FS
var buildVersion = ""
var buildTime = ""

func main() {
	// Command-line flags
	configFile := flag.String("config", "", "Path to config file (default: search ./config.yaml, data/config.yaml, etc.)")
	flag.Parse()

	// Load configuration
	cfg := config.Load(*configFile)

	// Initialize logger (simple for now)
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Open DB
	db, err := database.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("database open error: %v", err)
	}
	defer db.CloseSafe()
	if err := db.Migrate(); err != nil {
		log.Fatalf("migrate error: %v", err)
	}

	// Initialize GORM database for all models
	gormDB, err := gorm.Open(sqlite.Open(cfg.DBPath), &gorm.Config{})
	if err != nil {
		log.Fatalf("GORM database open error: %v", err)
	}
	if err := gormDB.AutoMigrate(
		&models.User{},
		&models.TransmissionLog{},
		&models.NodeInfo{},
		&models.LinkStat{},
		&models.CallsignProfile{},
		&models.LevelConfig{},
		&models.XPActivityLog{},
	); err != nil {
		log.Fatalf("GORM auto-migrate error: %v", err)
	}
	logger.Info("GORM database initialized successfully")

	// Initialize repositories
	txLogRepo := repository.NewTransmissionLogRepository(gormDB)
	nodeInfoRepo := repository.NewNodeInfoRepository(gormDB)

	// Initialize astdb downloader with node info repository
	astdbDownloader := astdb.NewDownloader(cfg.AstDBURL, cfg.AstDBPath, cfg.AstDBUpdateHours, logger)
	astdbDownloader.SetNodeInfoRepository(nodeInfoRepo)

	if err := astdbDownloader.EnsureExists(); err != nil {
		logger.Warn("failed to download/import astdb, node lookup may not work", zap.Error(err))
	} else {
		// Start auto-updater in background
		astdbDownloader.StartAutoUpdater()

		// Log node count from database
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		if count, err := nodeInfoRepo.GetCount(ctx); err == nil {
			logger.Info("astdb loaded successfully", zap.Int64("node_count", count))
		}
		cancel()
	}

	// API setup (use GORM for all repos now)
	apiLayer := api.New(gormDB, cfg.JWTSecret, cfg.TokenTTL)
	apiLayer.SetAstDBPath(cfg.AstDBPath)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", api.Health)
	mux.HandleFunc("/api/dashboard/summary", apiLayer.DashboardSummary)
	limiter := middleware.RateLimiter(cfg.AuthRateLimitRPM)
	mux.Handle("/api/auth/register", limiter(http.HandlerFunc(apiLayer.Register)))
	mux.Handle("/api/auth/login", limiter(http.HandlerFunc(apiLayer.Login)))

	// Repositories for middleware loaders
	userRepo := repository.NewUserRepo(gormDB)
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

	// Node lookup and talker log APIs - can be public or require auth based on config
	if cfg.AllowAnonDashboard {
		publicLimiter := middleware.RateLimiter(cfg.PublicStatsRateLimitRPM)
		mux.Handle("/api/node-lookup", publicLimiter(http.HandlerFunc(apiLayer.NodeLookup)))
		mux.Handle("/api/talker-log", publicLimiter(http.HandlerFunc(apiLayer.TalkerLog)))
	} else {
		mux.Handle("/api/node-lookup", authMW(http.HandlerFunc(apiLayer.NodeLookup)))
		mux.Handle("/api/talker-log", authMW(http.HandlerFunc(apiLayer.TalkerLog)))
	}

	// RPT and Voter stats APIs - require authentication
	mux.Handle("/api/rpt-stats", authMW(http.HandlerFunc(apiLayer.RPTStats)))
	mux.Handle("/api/voter-stats", authMW(http.HandlerFunc(apiLayer.VoterStats)))

	// Poll-now endpoint - authenticated by default; if anon dashboard is allowed, rate-limit it
	if cfg.AllowAnonDashboard {
		publicLimiter := middleware.RateLimiter(cfg.PublicStatsRateLimitRPM)
		mux.Handle("/api/poll-now", publicLimiter(http.HandlerFunc(apiLayer.PollNow)))
	} else {
		mux.Handle("/api/poll-now", authMW(http.HandlerFunc(apiLayer.PollNow)))
	}

	if cfg.AllowAnonDashboard {
		publicLimiter := middleware.RateLimiter(cfg.PublicStatsRateLimitRPM)
		mux.Handle("/api/link-stats", publicLimiter(http.HandlerFunc(apiLayer.LinkStatsHandler)))
		mux.Handle("/api/link-stats/top", publicLimiter(http.HandlerFunc(apiLayer.TopLinkStatsHandler)))
	} else {
		mux.Handle("/api/link-stats", authMW(http.HandlerFunc(apiLayer.LinkStatsHandler)))
		mux.Handle("/api/link-stats/top", authMW(http.HandlerFunc(apiLayer.TopLinkStatsHandler)))
	}

	// Gamification System Initialization
	var tallyService *gamification.TallyService
	if cfg.Gamification.Enabled {
		logger.Info("initializing gamification system...")

		// Initialize gamification repositories
		profileRepo := repository.NewCallsignProfileRepo(gormDB)
		levelConfigRepo := repository.NewLevelConfigRepo(gormDB)
		activityRepo := repository.NewXPActivityRepo(gormDB)

		// Calculate and seed level requirements
		levelRequirements := gamification.CalculateLevelRequirements()
		if err := levelConfigRepo.SeedDefaults(context.Background(), levelRequirements); err != nil {
			logger.Warn("failed to seed level config", zap.Error(err))
		} else {
			logger.Info("level config seeded", zap.Int("levels", len(levelRequirements)))
		}

		// Build gamification config for TallyService
		gameCfg := &gamification.Config{
			RestedEnabled:          cfg.Gamification.RestedBonus.Enabled,
			RestedAccumulationRate: cfg.Gamification.RestedBonus.AccumulationRate,
			RestedMaxSeconds:       cfg.Gamification.RestedBonus.MaxHours * 3600,
			RestedMultiplier:       cfg.Gamification.RestedBonus.Multiplier,
			DREnabled:              cfg.Gamification.DiminishingReturns.Enabled,
			KerchunkEnabled:        cfg.Gamification.KerchunkDetection.Enabled,
			KerchunkThreshold:      cfg.Gamification.KerchunkDetection.ThresholdSec,
			KerchunkWindow:         cfg.Gamification.KerchunkDetection.WindowSec,
			KerchunkSinglePenalty:  cfg.Gamification.KerchunkDetection.SinglePenalty,
			Kerchunk2to3Penalty:    cfg.Gamification.KerchunkDetection.TwoThree,
			Kerchunk4to5Penalty:    cfg.Gamification.KerchunkDetection.FourFive,
			Kerchunk6PlusPenalty:   cfg.Gamification.KerchunkDetection.SixPlus,
			CapsEnabled:            cfg.Gamification.XPCaps.Enabled,
			DailyCapSeconds:        cfg.Gamification.XPCaps.DailyCap,
			WeeklyCapSeconds:       cfg.Gamification.XPCaps.WeeklyCap,
		}

		// Convert DR tiers
		if len(cfg.Gamification.DiminishingReturns.Tiers) > 0 {
			for _, tier := range cfg.Gamification.DiminishingReturns.Tiers {
				gameCfg.DRTiers = append(gameCfg.DRTiers, gamification.DRTier{
					MaxSeconds: tier.MaxSeconds,
					Multiplier: tier.Multiplier,
				})
			}
		} else {
			// Default tiers if not configured
			gameCfg.DRTiers = []gamification.DRTier{
				{MaxSeconds: 1200, Multiplier: 1.0},
				{MaxSeconds: 2400, Multiplier: 0.75},
				{MaxSeconds: 3600, Multiplier: 0.5},
				{MaxSeconds: 999999, Multiplier: 0.25},
			}
		}

		// Initialize and start TallyService
		tallyInterval := time.Duration(cfg.Gamification.TallyIntervalMinutes) * time.Minute
		tallyService = gamification.NewTallyService(
			gormDB,
			txLogRepo,
			profileRepo,
			levelConfigRepo,
			activityRepo,
			gameCfg,
			tallyInterval,
			logger,
		)

		if err := tallyService.Start(); err != nil {
			logger.Error("failed to start tally service", zap.Error(err))
		} else {
			logger.Info("gamification tally service started",
				zap.Duration("interval", tallyInterval),
				zap.Bool("rested_bonus", gameCfg.RestedEnabled),
				zap.Bool("diminishing_returns", gameCfg.DREnabled),
				zap.Bool("kerchunk_detection", gameCfg.KerchunkEnabled),
				zap.Bool("xp_caps", gameCfg.CapsEnabled),
			)
		}

		// Register gamification API endpoints
		gamificationAPI := api.NewGamificationAPI(profileRepo, txLogRepo, levelConfigRepo, activityRepo)

		if cfg.AllowAnonDashboard {
			publicLimiter := middleware.RateLimiter(cfg.PublicStatsRateLimitRPM)
			mux.Handle("/api/gamification/scoreboard", publicLimiter(http.HandlerFunc(gamificationAPI.Scoreboard)))
			mux.Handle("/api/gamification/profile/", publicLimiter(http.HandlerFunc(gamificationAPI.Profile)))
			mux.Handle("/api/gamification/recent-transmissions", publicLimiter(http.HandlerFunc(gamificationAPI.RecentTransmissions)))
			mux.Handle("/api/gamification/level-config", publicLimiter(http.HandlerFunc(gamificationAPI.LevelConfig)))
		} else {
			mux.Handle("/api/gamification/scoreboard", authMW(http.HandlerFunc(gamificationAPI.Scoreboard)))
			mux.Handle("/api/gamification/profile/", authMW(http.HandlerFunc(gamificationAPI.Profile)))
			mux.Handle("/api/gamification/recent-transmissions", authMW(http.HandlerFunc(gamificationAPI.RecentTransmissions)))
			mux.Handle("/api/gamification/level-config", authMW(http.HandlerFunc(gamificationAPI.LevelConfig)))
		}

		logger.Info("gamification API endpoints registered")
	}

	// Serve Vue.js dashboard from embedded frontend/dist
	staticFS, err := fs.Sub(frontendFiles, "frontend/dist")
	if err != nil {
		log.Fatalf("embed fs error: %v", err)
	}
	log.Printf("serving Vue.js dashboard at /")
	mux.Handle("/", http.FileServer(http.FS(staticFS)))

	// AMI + WebSocket wiring (conditional). Always provide a /ws endpoint so the UI never hard-fails.
	var hub *web.Hub
	if cfg.AMIEnabled {
		hub = web.NewHub()
		sm := core.NewStateManager()

		// Initialize transmission log repository and inject into StateManager
		sm.SetTransmissionLogRepo(txLogRepo)
		logger.Info("transmission log repository initialized")

		// Configure node lookup service for server-side enrichment
		nodeLookup := core.NewNodeLookupService(cfg.AstDBPath)
		nodeLookup.SetNodeInfoRepository(nodeInfoRepo)
		sm.SetNodeLookup(nodeLookup)
		logger.Info("node lookup service configured with SQLite backend")
		// Propagate build metadata into StateManager so UI can display it
		if buildVersion != "" {
			sm.SetVersion(buildVersion)
		}
		if buildTime != "" {
			sm.SetBuildTime(buildTime)
			cfg.BuildTime = buildTime
		}
		if cfg.Title != "" {
			sm.SetTitle(cfg.Title)
		}
		if cfg.Subtitle != "" {
			sm.SetSubtitle(cfg.Subtitle)
		}
		// Set primary node ID (use first configured node)
		if len(cfg.Nodes) > 0 {
			sm.SetNodeID(cfg.Nodes[0].NodeID)
		}
		// Initialize keying trackers for all configured source nodes (2 second jitter delay)
		for _, node := range cfg.Nodes {
			sm.AddSourceNode(node.NodeID, 2000)
			logger.Info("initialized keying tracker for source node", zap.Int("node_id", node.NodeID))
		}
		// Seed persisted link stats (if any) so totals survive restarts
		lsRepo := repository.NewLinkStatsRepo(gormDB)
		seedCtx, seedCancel := context.WithTimeout(context.Background(), 2*time.Second)
		if stats, err := lsRepo.GetAll(seedCtx); err == nil && len(stats) > 0 {
			li := make([]core.LinkInfo, 0, len(stats))
			primaryNodeID := 0
			if len(cfg.Nodes) > 0 {
				primaryNodeID = cfg.Nodes[0].NodeID
			}
			for _, s := range stats {
				cs := time.Now()
				if s.ConnectedSince != nil {
					cs = *s.ConnectedSince
				}
				linkInfo := core.LinkInfo{
					Node:           s.Node,
					LocalNode:      primaryNodeID, // Set LocalNode for multi-node compatibility
					ConnectedSince: cs,
					LastTxStart:    s.LastTxStart,
					LastTxEnd:      s.LastTxEnd,
					TotalTxSeconds: s.TotalTxSeconds,
				}
				// Enrich seeded links with node lookup data
				nodeLookup.EnrichLinkInfo(&linkInfo)
				li = append(li, linkInfo)
			}
			sm.SeedLinkStats(li)
		}
		seedCancel()
		// Seed keying tracker with existing links (if any were loaded from persistence)
		// This ensures the keying tracker has data even before AMI events arrive
		if len(cfg.Nodes) > 0 {
			sm.SeedKeyingTrackerFromLinks(cfg.Nodes[0].NodeID)
		}
		go hub.BroadcastLoop(sm.Updates())
		go hub.TalkerLoop(sm.TalkerEvents())
		go hub.LinkUpdateLoop(sm.LinkUpdates())
		go hub.LinkRemovalLoop(sm.LinkRemovals())
		go hub.LinkTxBatchLoop(sm.LinkTxEvents(), 100*time.Millisecond)
		go hub.HeartbeatLoop(sm, 5*time.Second)
		go hub.TalkerLogRefreshLoop(sm, 2*time.Minute)      // Periodic talker log refresh
		go hub.SourceNodeKeyingLoop(sm.KeyingUpdates())     // Source node keying updates
		go hub.SourceNodeKeyingEventLoop(sm.KeyingEvents()) // Session edge events (TX_START/TX_END)
		conn := ami.NewConnector(cfg.AMIHost, cfg.AMIPort, cfg.AMIUser, cfg.AMIPassword, cfg.AMIEvents, cfg.AMIRetryInterval, cfg.AMIRetryMax)
		// Pass AMI connector and StateManager to API layer
		apiLayer.SetAMIConnector(conn)
		apiLayer.SetStateManager(sm)
		ctxAMI, cancelAMI := context.WithCancel(context.Background())

		// Monitor AMI connection status changes
		go func() {
			for status := range conn.ConnectionStatusChan() {
				if status.Connected {
					logger.Info("AMI connection established", zap.Time("timestamp", status.Timestamp))
				} else {
					if status.Error != nil {
						logger.Warn("AMI connection lost", zap.Error(status.Error), zap.Time("timestamp", status.Timestamp))
					} else {
						logger.Info("AMI connection closed", zap.Time("timestamp", status.Timestamp))
					}
				}
			}
		}()

		log.Printf("starting AMI connector (will auto-reconnect on failure)")
		if err := conn.Start(ctxAMI); err != nil {
			log.Printf("AMI start error: %v", err)
		} else {
			log.Printf("AMI connector started successfully")
		}
		// Diagnostic: issue a test AMI command after startup to verify responses are received and parsed.
		// Startup diagnostics removed: relying solely on event-driven AMI processing.
		go sm.Run(conn.Raw())
		logger.Info("using hybrid event-driven + polling AMI processing")

		// Start periodic polling service for data sync and enrichment
		// This provides a hybrid approach:
		// - Events drive real-time updates (ALINKS, TXKEYED, etc.)
		// - Polling (1 min) ensures sync and enriches with XStat/SawStat data (direction, IP, elapsed, mode)
		if !cfg.DisableLinkPoller {
			nodeIDs := make([]int, len(cfg.Nodes))
			for i, node := range cfg.Nodes {
				nodeIDs[i] = node.NodeID
			}
			pollingService := core.NewPollingService(conn, sm, 60*time.Second, nodeIDs)

			// Set cleanup callback to sync database with actual state after first poll
			// This cleans up any stale links that were seeded from database but are no longer connected
			pollingService.SetCleanupCallback(func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				// Get current link state from StateManager
				currentLinks := sm.Snapshot().LinksDetailed

				// Collect active node IDs
				activeNodeIDs := make([]int, len(currentLinks))
				for i, li := range currentLinks {
					activeNodeIDs[i] = li.Node
				}

				// Delete stale entries from database (nodes not in current state)
				deleted, err := lsRepo.DeleteNotIn(ctx, activeNodeIDs)
				if err != nil {
					logger.Warn("failed to clean up stale link stats", zap.Error(err))
				} else if deleted > 0 {
					logger.Info("cleaned up stale link stats from database", zap.Int64("deleted_count", deleted))
				}

				// Update active links in database
				for _, li := range currentLinks {
					stat := models.LinkStat{
						Node:           li.Node,
						TotalTxSeconds: li.TotalTxSeconds,
						LastTxStart:    li.LastTxStart,
						LastTxEnd:      li.LastTxEnd,
						ConnectedSince: &li.ConnectedSince,
					}
					if err := lsRepo.Upsert(ctx, stat); err != nil {
						logger.Warn("failed to sync link stat", zap.Int("node", li.Node), zap.Error(err))
					}
				}

				logger.Info("database synchronized with current link state", zap.Int("active_link_count", len(currentLinks)))
			})

			if err := pollingService.Start(); err != nil {
				logger.Warn("failed to start polling service", zap.Error(err))
			} else {
				logger.Info("polling service started", zap.Duration("interval", 60*time.Second), zap.Ints("nodes", nodeIDs))
			}
			// If a hub exists, wire a trigger so new WS clients cause an immediate
			// on-demand poll shortly after connecting (debounced).
			hub.SetTriggerPoll(func() { pollingService.TriggerPollOnce() })
			// Expose poll trigger to API: node==0 => poll all; else poll specific node
			apiLayer.SetTriggerPoll(func(nodeID int) {
				if nodeID > 0 {
					pollingService.TriggerPollNode(nodeID)
				} else {
					pollingService.TriggerPollOnce()
				}
			})
			// Stop polling service on shutdown
			defer pollingService.Stop()
		} else {
			logger.Info("polling service disabled via config (disable_link_poller=true)")
		}
		// Persist per-link TX stats on edges
		sm.SetPersistHook(func(list []core.LinkInfo) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			for _, li := range list {
				stat := models.LinkStat{Node: li.Node, TotalTxSeconds: li.TotalTxSeconds, LastTxStart: li.LastTxStart, LastTxEnd: li.LastTxEnd, ConnectedSince: &li.ConnectedSince}
				_ = lsRepo.Upsert(ctx, stat)
			}
		})
		validator := func(r *http.Request) (bool, bool) {
			token := r.URL.Query().Get("token")
			if token == "" {
				// allow anonymous if configured
				return cfg.AllowAnonDashboard, false
			}
			_, role, exp, err := auth.ParseJWT(token, cfg.JWTSecret)
			if err != nil || time.Now().After(exp) {
				return false, false
			}
			isAdmin := role == models.RoleAdmin || role == models.RoleSuperAdmin
			return true, isAdmin
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
		if cfg.Title != "" {
			sm.SetTitle(cfg.Title)
		}
		if cfg.Subtitle != "" {
			sm.SetSubtitle(cfg.Subtitle)
		}
		// Set primary node ID (use first configured node)
		if len(cfg.Nodes) > 0 {
			sm.SetNodeID(cfg.Nodes[0].NodeID)
		}
		validator := func(r *http.Request) (bool, bool) {
			token := r.URL.Query().Get("token")
			if token == "" {
				return cfg.AllowAnonDashboard, false
			}
			_, role, exp, err := auth.ParseJWT(token, cfg.JWTSecret)
			if err != nil || time.Now().After(exp) {
				return false, false
			}
			isAdmin := role == models.RoleAdmin || role == models.RoleSuperAdmin
			return true, isAdmin
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

	// Stop gamification tally service
	if tallyService != nil {
		tallyService.Stop()
	}

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
