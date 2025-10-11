package config

import (
	"log"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// NodeConfig represents configuration for a single AllStar node
type NodeConfig struct {
	NodeID int    `mapstructure:"node_id" yaml:"node_id" json:"node_id"`
	Name   string `mapstructure:"name" yaml:"name,omitempty" json:"name,omitempty"` // Optional - if empty, lookup from astdb
}

// GamificationConfig holds gamification system settings
type GamificationConfig struct {
	Enabled              bool                   `mapstructure:"enabled" yaml:"enabled"`
	TallyIntervalMinutes int                    `mapstructure:"tally_interval_minutes" yaml:"tally_interval_minutes"`
	RestedBonus          RestedBonusConfig      `mapstructure:"rested_bonus" yaml:"rested_bonus"`
	DiminishingReturns   DiminishingReturnsConfig `mapstructure:"diminishing_returns" yaml:"diminishing_returns"`
	KerchunkDetection    KerchunkConfig         `mapstructure:"kerchunk_detection" yaml:"kerchunk_detection"`
	XPCaps               XPCapsConfig           `mapstructure:"xp_caps" yaml:"xp_caps"`
}

type RestedBonusConfig struct {
	Enabled          bool    `mapstructure:"enabled" yaml:"enabled"`
	AccumulationRate float64 `mapstructure:"accumulation_rate" yaml:"accumulation_rate"`
	MaxHours         int     `mapstructure:"max_hours" yaml:"max_hours"`
	Multiplier       float64 `mapstructure:"multiplier" yaml:"multiplier"`
}

type DiminishingReturnsConfig struct {
	Enabled bool     `mapstructure:"enabled" yaml:"enabled"`
	Tiers   []DRTier `mapstructure:"tiers" yaml:"tiers"`
}

type DRTier struct {
	MaxSeconds int     `mapstructure:"max_seconds" yaml:"max_seconds"`
	Multiplier float64 `mapstructure:"multiplier" yaml:"multiplier"`
}

type KerchunkConfig struct {
	Enabled       bool    `mapstructure:"enabled" yaml:"enabled"`
	ThresholdSec  int     `mapstructure:"threshold_seconds" yaml:"threshold_seconds"`
	WindowSec     int     `mapstructure:"consecutive_window" yaml:"consecutive_window"`
	SinglePenalty float64 `mapstructure:"single" yaml:"single"`
	TwoThree      float64 `mapstructure:"two_to_three" yaml:"two_to_three"`
	FourFive      float64 `mapstructure:"four_to_five" yaml:"four_to_five"`
	SixPlus       float64 `mapstructure:"six_plus" yaml:"six_plus"`
}

type XPCapsConfig struct {
	Enabled     bool   `mapstructure:"enabled" yaml:"enabled"`
	DailyCap    int    `mapstructure:"daily_cap_seconds" yaml:"daily_cap_seconds"`
	WeeklyCap   int    `mapstructure:"weekly_cap_seconds" yaml:"weekly_cap_seconds"`
	ResetHour   int    `mapstructure:"reset_hour" yaml:"reset_hour"`
	WeekStarts  string `mapstructure:"week_starts" yaml:"week_starts"`
}

// Config holds runtime configuration values.
type Config struct {
	Port                    string
	DBPath                  string
	AstDBPath               string
	AstDBURL                string
	AstDBUpdateHours        int
	JWTSecret               string
	Env                     string
	BuildTime               string
	StartTime               time.Time
	TokenTTL                time.Duration
	AuthRateLimitRPM        int
	PublicStatsRateLimitRPM int
	AMIEnabled              bool
	AMIHost                 string
	AMIPort                 int
	AMIUser                 string
	AMIPassword             string
	AMIEvents               string
	AMIRetryInterval        time.Duration
	AMIRetryMax             time.Duration
	Nodes                   []NodeConfig // Multiple nodes support
	DisableLinkPoller       bool
	AllowAnonDashboard      bool
	Title                   string
	Subtitle                string
	Gamification            GamificationConfig
}

// Load loads configuration from config file and environment variables using Viper
// Optionally accepts a config file path as first argument
func Load(configPath ...string) Config {
	// Set default values
	viper.SetDefault("port", "8080")
	viper.SetDefault("db_path", "data/allstar.db")
	viper.SetDefault("astdb_path", "data/astdb.txt")
	viper.SetDefault("astdb_url", "http://allmondb.allstarlink.org/")
	viper.SetDefault("astdb_update_hours", 24)
	viper.SetDefault("jwt_secret", "dev-secret-change-me")
	viper.SetDefault("app_env", "development")
	viper.SetDefault("token_ttl_seconds", 86400)
	viper.SetDefault("auth_rpm", 60)
	viper.SetDefault("public_stats_rpm", 120)
	viper.SetDefault("ami_enabled", true)
	viper.SetDefault("ami_host", "127.0.0.1")
	viper.SetDefault("ami_port", 5038)
	viper.SetDefault("ami_username", "admin")
	viper.SetDefault("ami_password", "change-me")
	viper.SetDefault("ami_events", "on")
	viper.SetDefault("ami_retry_interval", "15s")
	viper.SetDefault("ami_retry_max", "60s")
	viper.SetDefault("ami_node_id", 0)
	viper.SetDefault("disable_link_poller", false)
	viper.SetDefault("allow_anon_dashboard", true)
	viper.SetDefault("title", "Allstar Nexus")
	viper.SetDefault("subtitle", "")

	// Gamification defaults (low-activity hub configuration)
	viper.SetDefault("gamification.enabled", false) // Disabled by default
	viper.SetDefault("gamification.tally_interval_minutes", 30)
	viper.SetDefault("gamification.rested_bonus.enabled", true)
	viper.SetDefault("gamification.rested_bonus.accumulation_rate", 1.5)
	viper.SetDefault("gamification.rested_bonus.max_hours", 336)
	viper.SetDefault("gamification.rested_bonus.multiplier", 2.0)
	viper.SetDefault("gamification.diminishing_returns.enabled", true)
	viper.SetDefault("gamification.kerchunk_detection.enabled", true)
	viper.SetDefault("gamification.kerchunk_detection.threshold_seconds", 3)
	viper.SetDefault("gamification.kerchunk_detection.consecutive_window", 30)
	viper.SetDefault("gamification.xp_caps.enabled", true)
	viper.SetDefault("gamification.xp_caps.daily_cap_seconds", 1200)
	viper.SetDefault("gamification.xp_caps.weekly_cap_seconds", 7200)
	viper.SetDefault("gamification.xp_caps.reset_hour", 0)
	viper.SetDefault("gamification.xp_caps.week_starts", "sunday")

	// Config file search paths
	if len(configPath) > 0 && configPath[0] != "" {
		// Use specified config file
		viper.SetConfigFile(configPath[0])
	} else {
		// Search for config in standard locations
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
		viper.AddConfigPath("data")
		viper.AddConfigPath("$HOME/.allstar-nexus")
		viper.AddConfigPath("/etc/allstar-nexus")
	}

	// Read config file if it exists (optional)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; using defaults and env vars
			log.Printf("No config file found, using defaults and environment variables")
		} else {
			// Config file found but error reading it
			log.Printf("Error reading config file: %v", err)
		}
	} else {
		log.Printf("Using config file: %s", viper.ConfigFileUsed())
	}

	// Environment variables override config file
	viper.SetEnvPrefix("") // No prefix to match existing env vars
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Build config struct
	cfg := Config{
		Port:                    viper.GetString("port"),
		DBPath:                  viper.GetString("db_path"),
		AstDBPath:               viper.GetString("astdb_path"),
		AstDBURL:                viper.GetString("astdb_url"),
		AstDBUpdateHours:        viper.GetInt("astdb_update_hours"),
		JWTSecret:               viper.GetString("jwt_secret"),
		Env:                     viper.GetString("app_env"),
		BuildTime:               viper.GetString("build_time"),
		StartTime:               time.Now(),
		TokenTTL:                time.Duration(viper.GetInt("token_ttl_seconds")) * time.Second,
		AuthRateLimitRPM:        viper.GetInt("auth_rpm"),
		PublicStatsRateLimitRPM: viper.GetInt("public_stats_rpm"),
		AMIEnabled:              viper.GetBool("ami_enabled"),
		AMIHost:                 viper.GetString("ami_host"),
		AMIPort:                 viper.GetInt("ami_port"),
		AMIUser:                 viper.GetString("ami_username"),
		AMIPassword:             viper.GetString("ami_password"),
		AMIEvents:               viper.GetString("ami_events"),
		AMIRetryInterval:        viper.GetDuration("ami_retry_interval"),
		AMIRetryMax:             viper.GetDuration("ami_retry_max"),
		DisableLinkPoller:       viper.GetBool("disable_link_poller"),
		AllowAnonDashboard:      viper.GetBool("allow_anon_dashboard"),
		Title:                   viper.GetString("title"),
		Subtitle:                viper.GetString("subtitle"),
	}

	// Load gamification configuration
	if err := viper.UnmarshalKey("gamification", &cfg.Gamification); err != nil {
		log.Printf("warning: failed to load gamification config: %v (using defaults)", err)
	}

	// Load nodes configuration - supports multiple formats:
	// 1. Simple array of integers: nodes: [43732, 48412]
	// 2. Array of objects with optional names: nodes: [{node_id: 43732, name: "My Node"}, {node_id: 48412}]
	// 3. Legacy single node: AMI_NODE_ID=43732

	// First try to unmarshal as array of integers
	var nodeIDs []int
	if err := viper.UnmarshalKey("nodes", &nodeIDs); err == nil && len(nodeIDs) > 0 {
		// Simple integer array format
		for _, id := range nodeIDs {
			cfg.Nodes = append(cfg.Nodes, NodeConfig{NodeID: id})
		}
		log.Printf("Loaded %d node(s) from configuration (simple format)", len(cfg.Nodes))
	} else {
		// Try array of objects format
		var nodes []NodeConfig
		if err := viper.UnmarshalKey("nodes", &nodes); err == nil && len(nodes) > 0 {
			cfg.Nodes = nodes
			log.Printf("Loaded %d node(s) from configuration (object format)", len(nodes))
		} else {
			// Fallback: Check for legacy single node ID (AMI_NODE_ID env var)
			if nodeID := viper.GetInt("ami_node_id"); nodeID > 0 {
				cfg.Nodes = []NodeConfig{{NodeID: nodeID}}
				log.Printf("Using legacy AMI_NODE_ID=%d (consider migrating to nodes array)", nodeID)
			}
		}
	}

	// Ensure data directory exists
	if err := os.MkdirAll(dirOf(cfg.DBPath), 0o755); err != nil {
		log.Printf("warning: unable to create data dir: %v", err)
	}

	// Validation
	if len(cfg.Nodes) == 0 {
		log.Printf("WARNING: No nodes configured - enhanced features (COS/PTT, link modes) will not work")
		log.Printf("Add nodes to config.yaml or set AMI_NODE_ID environment variable")
	} else {
		for _, node := range cfg.Nodes {
			if node.Name != "" {
				log.Printf("Configured node: %d (%s)", node.NodeID, node.Name)
			} else {
				log.Printf("Configured node: %d (name will be looked up from astdb)", node.NodeID)
			}
		}
	}

	return cfg
}

func dirOf(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[:i]
		}
	}
	return "."
}

// SaveExampleConfig creates an example config.yaml file
func SaveExampleConfig(path string) error {
	exampleConfig := `# Allstar Nexus Configuration File
# This file uses YAML format
# Environment variables will override these values

# Server Configuration
port: 8080
app_env: production

# Database
db_path: data/allstar.db
astdb_path: data/astdb.txt
astdb_url: http://allmondb.allstarlink.org/
astdb_update_hours: 24

# Security
jwt_secret: change-me-in-production
token_ttl_seconds: 86400  # 24 hours

# Rate Limiting
auth_rpm: 60
public_stats_rpm: 120

# AMI Configuration
ami_enabled: true
ami_host: 127.0.0.1
ami_port: 5038
ami_username: admin
ami_password: change-me

# Node Configuration - MULTIPLE NODES SUPPORTED!
# Simple format: just list your node numbers (names auto-lookup from astdb)
nodes: [43732, 48412]

# Advanced format: specify custom names (optional)
# nodes:
#   - node_id: 43732
#     name: "K8FBI Flying Beers International Hub"
#   - node_id: 48412
#     name: "FBI HQ"

# Legacy single node support (for backwards compatibility)
# ami_node_id: 43732

ami_events: "on"
ami_retry_interval: 15s
ami_retry_max: 60s

# Feature Toggles
disable_link_poller: false  # false = hybrid polling enabled (polls XStat/SawStat every 60s for enriched data)
allow_anon_dashboard: true
`
	return os.WriteFile(path, []byte(exampleConfig), 0644)
}
