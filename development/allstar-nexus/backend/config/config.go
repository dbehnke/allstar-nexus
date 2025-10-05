package config

import (
	"fmt"
	"log"
	"os"
	"time"
)

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
	DisableLinkPoller       bool
	AllowAnonDashboard      bool
}

// Load loads configuration from environment variables with sane defaults.
func Load() Config {
	cfg := Config{
		Port:                    getEnv("PORT", "8080"),
		DBPath:                  getEnv("DB_PATH", "data/allstar.db"),
		AstDBPath:               getEnv("ASTDB_PATH", "data/astdb.txt"),
		AstDBURL:                getEnv("ASTDB_URL", "http://allmondb.allstarlink.org/"),
		AstDBUpdateHours:        parseInt(getEnv("ASTDB_UPDATE_HOURS", "24"), 24),
		JWTSecret:               getEnv("JWT_SECRET", "dev-secret-change-me"),
		Env:                     getEnv("APP_ENV", "development"),
		BuildTime:               getEnv("BUILD_TIME", ""),
		StartTime:               time.Now(),
		TokenTTL:                parseDurationSeconds(getEnv("TOKEN_TTL_SECONDS", "86400")),
		AuthRateLimitRPM:        parseInt(getEnv("AUTH_RPM", "60"), 60),
		PublicStatsRateLimitRPM: parseInt(getEnv("PUBLIC_STATS_RPM", "120"), 120),
		AMIEnabled:              getEnv("AMI_ENABLED", "false") == "true",
		AMIHost:                 getEnv("AMI_HOST", "127.0.0.1"),
		AMIPort:                 parseInt(getEnv("AMI_PORT", "5038"), 5038),
		AMIUser:                 getEnv("AMI_USERNAME", "admin"),
		AMIPassword:             getEnv("AMI_PASSWORD", "change-me"),
		AMIEvents:               getEnv("AMI_EVENTS", "on"),
		AMIRetryInterval:        parseGoDuration(getEnv("AMI_RETRY_INTERVAL", "15s"), 15*time.Second),
		AMIRetryMax:             parseGoDuration(getEnv("AMI_RETRY_MAX", "60s"), 60*time.Second),
		DisableLinkPoller:       getEnv("DISABLE_LINK_POLLER", "false") == "true",
		AllowAnonDashboard:      getEnv("ALLOW_ANON_DASHBOARD", "true") == "true",
	}
	// Ensure data directory exists
	if err := os.MkdirAll(dirOf(cfg.DBPath), 0o755); err != nil {
		log.Printf("warning: unable to create data dir: %v", err)
	}
	return cfg
}

func getEnv(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}

func dirOf(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[:i]
		}
	}
	return "."
}

func parseDurationSeconds(s string) time.Duration {
	// fallback 24h
	if s == "" {
		return 24 * time.Hour
	}
	var secs int64
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return 24 * time.Hour
		}
	}
	_, err := fmt.Sscanf(s, "%d", &secs)
	if err != nil || secs <= 0 {
		return 24 * time.Hour
	}
	return time.Duration(secs) * time.Second
}

func parseInt(s string, def int) int {
	if s == "" {
		return def
	}
	var v int
	if _, err := fmt.Sscanf(s, "%d", &v); err != nil || v <= 0 {
		return def
	}
	return v
}

func parseGoDuration(s string, def time.Duration) time.Duration {
	if s == "" {
		return def
	}
	d, err := time.ParseDuration(s)
	if err != nil || d <= 0 {
		return def
	}
	return d
}
