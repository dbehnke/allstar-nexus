package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dbehnke/allstar-nexus/backend/api"
	"github.com/dbehnke/allstar-nexus/backend/auth"
	"github.com/dbehnke/allstar-nexus/backend/bootstrap"
	"github.com/dbehnke/allstar-nexus/backend/config"
	"github.com/dbehnke/allstar-nexus/backend/gamification"
	"github.com/dbehnke/allstar-nexus/backend/middleware"
	"github.com/dbehnke/allstar-nexus/backend/models"
	"github.com/dbehnke/allstar-nexus/backend/repository"
	"github.com/dbehnke/allstar-nexus/backend/server"
	"github.com/dbehnke/allstar-nexus/backend/tools"
	"github.com/dbehnke/allstar-nexus/internal/ami"
	"github.com/dbehnke/allstar-nexus/internal/astdb"
	"github.com/dbehnke/allstar-nexus/internal/core"
	"github.com/dbehnke/allstar-nexus/internal/discord"
	"github.com/dbehnke/allstar-nexus/internal/urfd"
	"github.com/dbehnke/allstar-nexus/internal/web"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	_ "modernc.org/sqlite"
)

//go:embed all:frontend/dist
var frontendFiles embed.FS
var buildVersion = ""
var buildTime = ""

func main() {
	// Command-line flags
	configFile := flag.String("config", "", "Path to config file (default: search ./config.yaml, data/config.yaml, etc.)")
	force := flag.Bool("force", false, "When set, ignore config validation errors and continue startup")
	flag.Usage = func() {
		// Minimal usage with subcommands
		_, _ = os.Stderr.WriteString("Allstar Nexus\n")
		_, _ = os.Stderr.WriteString("\nUsage:\n")
		_, _ = os.Stderr.WriteString("  allstar-nexus [flags]\n")
		_, _ = os.Stderr.WriteString("  allstar-nexus config validate [--config path]\n")
		_, _ = os.Stderr.WriteString("\nFlags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	// Handle optional subcommands: `config validate` and `db backfill-timestamps`
	// We deliberately check os.Args to avoid interfering with standard flag parsing for the server.
	if len(os.Args) >= 3 && os.Args[1] == "config" && os.Args[2] == "validate" {
		// Allow --config to override the path; if not provided, Validate will search defaults.
		if err := config.Validate(*configFile); err != nil {
			log.Printf("config validation: FAIL: %v", err)
			os.Exit(2)
		}
		log.Printf("config validation: PASS")
		return
	}

	if len(os.Args) >= 3 && os.Args[1] == "db" && os.Args[2] == "backfill-timestamps" {
		// Load config to resolve DB path (allow --config override or env)
		cfg := config.Load(*configFile)
		// Open DB
		gormDB, err := gorm.Open(sqlite.New(sqlite.Config{
			DriverName: "sqlite",
			DSN:        cfg.DBPath,
		}), &gorm.Config{})
		if err != nil {
			log.Fatalf("GORM database open error: %v", err)
		}
		// Basic logger to stdout
		logger, _ := zap.NewProduction()
		defer func() { _ = logger.Sync() }()
		if err := tools.BackfillTransmissionLogTimestamps(cfg.DBPath, gormDB, logger); err != nil {
			log.Fatalf("backfill failed: %v", err)
		}
		log.Printf("timestamp backfill completed successfully for %s", cfg.DBPath)
		return
	}

	// Optional subcommand: `db backfill-epochs` to populate start_unix/end_unix from timestamps
	if len(os.Args) >= 3 && os.Args[1] == "db" && os.Args[2] == "backfill-epochs" {
		// Load config to resolve DB path (allow --config override or env)
		cfg := config.Load(*configFile)
		// Open DB
		gormDB, err := gorm.Open(sqlite.New(sqlite.Config{
			DriverName: "sqlite",
			DSN:        cfg.DBPath,
		}), &gorm.Config{})
		if err != nil {
			log.Fatalf("GORM database open error: %v", err)
		}
		// Basic logger to stdout
		logger, _ := zap.NewProduction()
		defer func() { _ = logger.Sync() }()
		if err := tools.BackfillTransmissionEpochs(gormDB, logger); err != nil {
			log.Fatalf("epoch backfill failed: %v", err)
		}
		log.Printf("epoch backfill completed successfully for %s", cfg.DBPath)
		return
	}

	// Load configuration
	// Fail-fast: validate YAML and basic structure before full startup unless --force is provided.
	if err := config.Validate(*configFile); err != nil {
		if *force {
			log.Printf("config validation failed but --force provided; continuing startup. Validation error: %v", err)
		} else {
			log.Fatalf("config validation failed: %v. Use --force to override and continue at your own risk.", err)
		}
	}

	cfg := config.Load(*configFile)

	// Initialize logger (simple for now)
	logger, _ := zap.NewProduction()
	defer func() { _ = logger.Sync() }()

	// Initialize GORM database
	gormDB, err := bootstrap.InitDB(&cfg, logger)
	if err != nil {
		log.Fatalf("database initialization error: %v", err)
	}

	// Ensure epoch columns are populated for efficient time filtering
	if err := tools.BackfillTransmissionEpochs(gormDB, logger); err != nil {
		logger.Warn("failed to backfill epoch columns", zap.Error(err))
	}

	// Repository variables (declare so closures later can access)
	var txLogRepo *repository.TransmissionLogRepository
	var nodeInfoRepo *repository.NodeInfoRepository
	var profileRepo *repository.CallsignProfileRepo
	var levelConfigRepo *repository.LevelConfigRepo

	// Initialize repositories
	txLogRepo = repository.NewTransmissionLogRepository(gormDB)
	nodeInfoRepo = repository.NewNodeInfoRepository(gormDB)

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
	apiLayer.SetBuildInfo(buildVersion, buildTime)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", api.Health)
	mux.HandleFunc("/api/version", apiLayer.Version)
	mux.HandleFunc("/api/status", apiLayer.Status)
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

		// Validate level groupings
		var levelGroupings []config.LevelGrouping
		if len(cfg.Gamification.LevelGroupings) > 0 {
			levelGroupings = cfg.Gamification.LevelGroupings
			if err := gamification.ValidateGroupings(levelGroupings); err != nil {
				log.Fatalf("invalid level groupings configuration: %v", err)
			}
			logger.Info("level groupings validated", zap.Int("groups", len(levelGroupings)))
		} else {
			// Use defaults
			levelGroupings = gamification.DefaultLevelGroupings()
			logger.Info("using default level groupings", zap.Int("groups", len(levelGroupings)))
		}

		// Initialize gamification repositories
		profileRepo = repository.NewCallsignProfileRepo(gormDB)
		levelConfigRepo = repository.NewLevelConfigRepo(gormDB)
		activityRepo := repository.NewXPActivityRepo(gormDB)
		stateRepo := repository.NewTallyStateRepo(gormDB)

		// Calculate and seed level requirements (configurable)
		var levelRequirements map[int]int
		if len(cfg.Gamification.LevelScale) > 0 {
			levelRequirements = gamification.CalculateLevelRequirementsWithScale(cfg.Gamification.LevelScale)
		} else {
			levelRequirements = gamification.CalculateLevelRequirements()
		}
		if err := levelConfigRepo.SeedDefaults(context.Background(), levelRequirements); err != nil {
			logger.Warn("failed to seed level config", zap.Error(err))
		} else {
			logger.Info("level config seeded", zap.Int("levels", len(levelRequirements)))
		}

		// Build gamification config for TallyService
		gameCfg := &gamification.Config{
			RestedEnabled:              cfg.Gamification.RestedBonus.Enabled,
			RestedAccumulationRate:     cfg.Gamification.RestedBonus.AccumulationRate,
			RestedMaxSeconds:           cfg.Gamification.RestedBonus.MaxHours * 3600,
			RestedMultiplier:           cfg.Gamification.RestedBonus.Multiplier,
			RestedIdleThresholdSeconds: cfg.Gamification.RestedBonus.IdleThresholdSec,
			DREnabled:                  cfg.Gamification.DiminishingReturns.Enabled,
			KerchunkEnabled:            cfg.Gamification.KerchunkDetection.Enabled,
			KerchunkThreshold:          cfg.Gamification.KerchunkDetection.ThresholdSec,
			KerchunkWindow:             cfg.Gamification.KerchunkDetection.WindowSec,
			KerchunkSinglePenalty:      cfg.Gamification.KerchunkDetection.SinglePenalty,
			Kerchunk2to3Penalty:        cfg.Gamification.KerchunkDetection.TwoThree,
			Kerchunk4to5Penalty:        cfg.Gamification.KerchunkDetection.FourFive,
			Kerchunk6PlusPenalty:       cfg.Gamification.KerchunkDetection.SixPlus,
			CapsEnabled:                cfg.Gamification.XPCaps.Enabled,
			DailyCapSeconds:            cfg.Gamification.XPCaps.DailyCap,
			WeeklyCapSeconds:           cfg.Gamification.XPCaps.WeeklyCap,
			// Renown settings
			RenownEnabled:       cfg.Gamification.Renown.Enabled,
			RenownXPPerLevel:    cfg.Gamification.Renown.XPPerLevel,
			ScoringSourceNodeID: cfg.Gamification.ScoringSourceNodeID,
			ExcludedCallsigns:   cfg.Gamification.ExcludedCallsigns,
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

		// Initialize Discord notifier if enabled
		var discordNotifier *gamification.DiscordNotifier
		if cfg.Gamification.Discord.Enabled && cfg.Gamification.Discord.WebhookURL != "" {
			// Gamification-specific Discord webhook configured
			discordNotifier = gamification.NewDiscordNotifier(
				cfg.Gamification.Discord.WebhookURL,
				true, // Already checked enabled above
				logger,
			)
			logger.Info("Discord notifications enabled for gamification (dedicated webhook)")
		} else if cfg.Discord.Enabled && cfg.Discord.WebhookURL != "" {
			// Fall back to main Discord webhook if gamification Discord not configured
			// This allows users to use a single webhook for both node activity and gamification
			discordNotifier = gamification.NewDiscordNotifier(
				cfg.Discord.WebhookURL,
				true,
				logger,
			)
			logger.Info("Discord notifications enabled for gamification (using main Discord webhook)")
		}

		// Initialize and start TallyService
		tallyInterval := time.Duration(cfg.Gamification.TallyIntervalMinutes) * time.Minute
		tallyService = gamification.NewTallyService(
			gormDB,
			txLogRepo,
			profileRepo,
			levelConfigRepo,
			activityRepo,
			stateRepo,
			gameCfg,
			tallyInterval,
			logger,
			levelGroupings,
			discordNotifier,
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

		// Register gamification API endpoints (include rested server config values)
		gamificationAPI := api.NewGamificationAPI(
			profileRepo,
			txLogRepo,
			levelConfigRepo,
			activityRepo,
			levelGroupings,
			cfg.Gamification.Renown.Enabled,
			cfg.Gamification.Renown.XPPerLevel,
			cfg.Gamification.RestedBonus.Enabled,
			cfg.Gamification.RestedBonus.AccumulationRate,
			cfg.Gamification.RestedBonus.MaxHours,
			cfg.Gamification.RestedBonus.Multiplier,
			cfg.Gamification.RestedBonus.IdleThresholdSec,
			cfg.Gamification.XPCaps.WeeklyCap,
			cfg.Gamification.XPCaps.DailyCap,
			cfg.Gamification.DiminishingReturns.Tiers,
		)

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
	if _, err := fs.Sub(frontendFiles, "frontend/dist"); err != nil {
		log.Fatalf("embed fs error: %v", err)
	}
	log.Printf("serving Vue.js dashboard at /")
	// Use SPA fallback so direct reloads on client routes (e.g., /talker) work.
	spaHandler, err := server.SPAFileServer(frontendFiles, "frontend/dist")
	if err != nil {
		log.Fatalf("spa file server error: %v", err)
	}
	mux.Handle("/", spaHandler)

	// AMI + WebSocket wiring (conditional). Always provide a /ws endpoint so the UI never hard-fails.
	var hub *web.Hub
	if cfg.AMIEnabled {
		// Log effective AMI configuration (masking sensitive values) to aid troubleshooting
		logger.Info("AMI enabled. Effective configuration",
			zap.String("host", cfg.AMIHost),
			zap.Int("port", cfg.AMIPort),
			zap.String("user", cfg.AMIUser),
			zap.String("events", cfg.AMIEvents),
			zap.Duration("retry_min", cfg.AMIRetryInterval),
			zap.Duration("retry_max", cfg.AMIRetryMax),
		)
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
		for _, node := range cfg.Nodes {
			sm.SeedKeyingTrackerFromLinks(node.NodeID)
		}

		// Initialize Discord notifier if enabled (before starting hub loops)
		var discordNotifier *discord.Notifier
		if cfg.Discord.Enabled {
			nodeName := ""
			if len(cfg.Nodes) > 0 {
				nodeName = fmt.Sprintf("%d", cfg.Nodes[0].NodeID)
				// Use custom name if configured
				if cfg.Nodes[0].Name != "" {
					nodeName = cfg.Nodes[0].Name
				}
			}

			discordConfig := discord.Config{
				WebhookURL:            cfg.Discord.WebhookURL,
				QSOInactiveSeconds:    cfg.Discord.QSOInactiveSeconds,
				NodeIdleSeconds:       cfg.Discord.NodeIdleSeconds,
				MinTalkersForQSO:      cfg.Discord.MinTalkersForQSO,
				NotifyIndividualTalks: cfg.Discord.NotifyIndividualTalks,
			}

			discordNotifier = discord.NewNotifier(discordConfig, cfg.Nodes[0].NodeID, nodeName)

			// Set the node lookup service so Discord can enrich notifications with callsigns
			discordNotifier.SetNodeLookup(nodeLookup)

			discordNotifier.Start()
			logger.Info("Discord webhook notifier started",
				zap.Int("node_id", cfg.Nodes[0].NodeID),
				zap.String("node_name", nodeName),
				zap.Int("qso_inactive_seconds", cfg.Discord.QSOInactiveSeconds),
				zap.Int("node_idle_seconds", cfg.Discord.NodeIdleSeconds),
				zap.Bool("notify_individual_talks", cfg.Discord.NotifyIndividualTalks),
				zap.String("webhook_url_set", func() string {
					if cfg.Discord.WebhookURL != "" {
						return "yes (length: " + fmt.Sprintf("%d", len(cfg.Discord.WebhookURL)) + ")"
					}
					return "no"
				}()),
			)

			// Hook the Discord notifier into the hub's talker event broadcast
			hub.SetOnTalkerEvent(func(evt core.TalkerEvent) {
				discordNotifier.ProcessTalkerEvent(evt)
			})

			// Hook the Discord notifier into the hub's link TX event broadcast
			// This is the primary source of per-node events
			hub.SetOnLinkTxEvent(func(evt core.LinkTxEvent) {
				discordNotifier.ProcessLinkTxEvent(evt)
			})
			logger.Info("Discord notifier hooks registered with hub")

			// Register cleanup on shutdown
			defer discordNotifier.Stop()
		}

		// Initialize URFD NNG Control Client (if enabled)
		urfdClient := urfd.NewClient(cfg.URFD.Address, cfg.URFD.Enabled)
		if cfg.URFD.Enabled {
			if err := urfdClient.Connect(); err != nil {
				logger.Warn("failed to connect to urfd control channel", zap.Error(err))
			} else {
				logger.Info("connected to urfd control channel", zap.String("address", cfg.URFD.Address))
			}

			// Hook into Hub's Keying Events to trigger registration
			hub.SetOnKeyingEvent(func(evt core.SourceNodeKeyingEvent) {
				// We only care about TX_START (Key-Up) events that have a valid IP and Callsign
				if evt.Type == "TX_START" && evt.IP != "" && evt.Callsign != "" {
					// Use configured ReportAddress if set, otherwise use the detected IP
					targetIP := evt.IP
					if cfg.URFD.ReportAddress != "" {
						targetIP = cfg.URFD.ReportAddress
					}

					// Async call to avoid blocking the event loop
					go func() {
						if err := urfdClient.Register(targetIP, evt.Callsign); err != nil {
							logger.Error("failed to register USRP client with urfd",
								zap.String("ip", targetIP),
								zap.String("original_ip", evt.IP),
								zap.String("callsign", evt.Callsign),
								zap.Error(err),
							)
						}
					}()
				}
			})
			defer urfdClient.Close()
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

		// Create one AMI connector per configured node
		ctxAMI, cancelAMI := context.WithCancel(context.Background())
		for _, node := range cfg.Nodes {
			amiHost := node.AMIHost
			if amiHost == "" {
				amiHost = cfg.AMIHost
			}
			amiPort := node.AMIPort
			if amiPort == 0 {
				amiPort = cfg.AMIPort
			}

			connector := ami.NewConnector(amiHost, amiPort, cfg.AMIUser, cfg.AMIPassword, cfg.AMIEvents, cfg.AMIRetryInterval, cfg.AMIRetryMax)
			connector.WithSourceNodeID(node.NodeID)

			// Feed this connector's events into StateManager
			go sm.Run(connector.Raw())

			// Start connector (auto-reconnects on failure)
			go func() {
				if err := connector.Start(ctxAMI); err != nil {
					logger.Warn("AMI connector start error", zap.Error(err))
				}
			}()
		}
		// Pass StateManager to API layer
		apiLayer.SetStateManager(sm)

		// If tally service is running, broadcast a WS event when it completes
		if tallyService != nil {
			// When a tally completes, broadcast the summary and include the current leaderboard
			// so clients can update immediately without an extra HTTP fetch.
			tallyService.OnTallyComplete = func(summary gamification.TallySummary) {
				if hub == nil {
					return
				}
				// Build a lightweight scoreboard snapshot to send over WS
				ctx := context.Background()
				profiles, err := profileRepo.GetLeaderboard(ctx, 50)
				if err != nil {
					// fallback: broadcast only summary
					hub.BroadcastTallyCompleted(summary)
					return
				}
				levelCfg, _ := levelConfigRepo.GetAllAsMap(ctx)
				// Build entries similar to API response shape
				entries := make([]map[string]any, 0, len(profiles))
				for _, p := range profiles {
					nextXP := 0
					if xp, ok := levelCfg[p.Level+1]; ok {
						nextXP = xp
					}
					totalTime, _ := txLogRepo.GetTotalTransmissionTime(p.Callsign)
					entries = append(entries, map[string]any{
						"callsign":                p.Callsign,
						"level":                   p.Level,
						"experience_points":       p.ExperiencePoints,
						"renown_level":            p.RenownLevel,
						"next_level_xp":           nextXP,
						"total_talk_time_seconds": totalTime,
					})
				}
				hub.BroadcastTallyCompleted(map[string]any{"summary": summary, "scoreboard": entries})
			}
		}

		logger.Info("AMI connectors started for all configured nodes")

		// TODO(multi-node): Polling service needs architectural changes to work with per-node connectors
		// The polling service was designed for a single connector and uses conn parameter.
		// Re-enable once multi-node polling is implemented.
		if false && !cfg.DisableLinkPoller {
		} else {
			logger.Info("polling service disabled (multi-node polling not yet implemented)")
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
	defer func() { _ = zapLogger.Sync() }()
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
