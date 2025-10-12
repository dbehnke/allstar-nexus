package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	cfgpkg "github.com/dbehnke/allstar-nexus/backend/config"
	"github.com/dbehnke/allstar-nexus/backend/gamification"
	"github.com/dbehnke/allstar-nexus/backend/repository"
)

type GamificationAPI struct {
	profileRepo    *repository.CallsignProfileRepo
	txLogRepo      *repository.TransmissionLogRepository
	levelRepo      *repository.LevelConfigRepo
	activityRepo   *repository.XPActivityRepo
	levelGroupings []cfgpkg.LevelGrouping
}

func NewGamificationAPI(
	profileRepo *repository.CallsignProfileRepo,
	txLogRepo *repository.TransmissionLogRepository,
	levelRepo *repository.LevelConfigRepo,
	activityRepo *repository.XPActivityRepo,
	levelGroupings []cfgpkg.LevelGrouping,
) *GamificationAPI {
	return &GamificationAPI{
		profileRepo:    profileRepo,
		txLogRepo:      txLogRepo,
		levelRepo:      levelRepo,
		activityRepo:   activityRepo,
		levelGroupings: levelGroupings,
	}
}

// Scoreboard returns top N callsigns ranked by renown, level, and XP
// GET /api/gamification/scoreboard?limit=50
func (g *GamificationAPI) Scoreboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := context.Background()

	// Parse limit parameter
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}

	// Get leaderboard
	profiles, err := g.profileRepo.GetLeaderboard(ctx, limit)
	if err != nil {
		http.Error(w, "Failed to get leaderboard", http.StatusInternalServerError)
		return
	}

	// Get level config for XP requirements
	levelConfig, err := g.levelRepo.GetAllAsMap(ctx)
	if err != nil {
		http.Error(w, "Failed to get level config", http.StatusInternalServerError)
		return
	}

	// Build response with rank and next level XP
	type ScoreboardEntry struct {
		Rank             int                         `json:"rank"`
		Callsign         string                      `json:"callsign"`
		Level            int                         `json:"level"`
		ExperiencePoints int                         `json:"experience_points"`
		RenownLevel      int                         `json:"renown_level"`
		NextLevelXP      int                         `json:"next_level_xp"`
		TotalTalkTime    int                         `json:"total_talk_time_seconds,omitempty"`
		Grouping         *gamification.GroupingInfo  `json:"grouping,omitempty"`
	}

	var entries []ScoreboardEntry
	for i, profile := range profiles {
		nextLevelXP := 0
		if xp, ok := levelConfig[profile.Level+1]; ok {
			nextLevelXP = xp
		}

		// Get total talk time for this callsign
		totalTime, _ := g.txLogRepo.GetTotalTransmissionTime(profile.Callsign)

		// Get grouping for this level (if renown is 0)
		var grouping *gamification.GroupingInfo
		if profile.RenownLevel == 0 {
			grouping = gamification.GetGroupingForLevel(profile.Level, g.levelGroupings)
		}

		entries = append(entries, ScoreboardEntry{
			Rank:             i + 1,
			Callsign:         profile.Callsign,
			Level:            profile.Level,
			ExperiencePoints: profile.ExperiencePoints,
			RenownLevel:      profile.RenownLevel,
			NextLevelXP:      nextLevelXP,
			TotalTalkTime:    totalTime,
			Grouping:         grouping,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"scoreboard": entries,
		"enabled":    true,
	})
}

// Profile returns detailed profile for a specific callsign
// GET /api/gamification/profile/:callsign
func (g *GamificationAPI) Profile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := context.Background()

	// Extract callsign from URL path
	// Expected: /api/gamification/profile/K8FBI
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 {
		http.Error(w, "Callsign required", http.StatusBadRequest)
		return
	}
	callsign := parts[4]

	if callsign == "" {
		http.Error(w, "Callsign required", http.StatusBadRequest)
		return
	}

	// Get profile
	profile, err := g.profileRepo.GetByCallsign(ctx, callsign)
	if err != nil {
		http.Error(w, "Failed to get profile", http.StatusInternalServerError)
		return
	}

	// Get level config
	levelConfig, err := g.levelRepo.GetAllAsMap(ctx)
	if err != nil {
		http.Error(w, "Failed to get level config", http.StatusInternalServerError)
		return
	}

	// Get next level XP
	nextLevelXP := 0
	if xp, ok := levelConfig[profile.Level+1]; ok {
		nextLevelXP = xp
	}

	// Get total talk time
	totalTime, _ := g.txLogRepo.GetTotalTransmissionTime(profile.Callsign)

	// Get weekly/daily XP
	weeklyXP, _ := g.activityRepo.GetWeeklyXP(ctx, callsign)
	dailyXP, _ := g.activityRepo.GetDailyXP(ctx, callsign)

	// Get recent activity breakdown
	breakdown, _ := g.activityRepo.GetDailyBreakdown(ctx, callsign, 7)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"callsign":                profile.Callsign,
		"level":                   profile.Level,
		"experience_points":       profile.ExperiencePoints,
		"renown_level":            profile.RenownLevel,
		"next_level_xp":           nextLevelXP,
		"total_talk_time_seconds": totalTime,
		"rested_bonus_hours":      profile.RestedBonusSeconds / 3600,
		"rested_bonus_seconds":    profile.RestedBonusSeconds,
		"last_transmission_at":    profile.LastTransmissionAt,
		"weekly_xp":               weeklyXP,
		"daily_xp":                dailyXP,
		"daily_breakdown":         breakdown,
	})
}

// RecentTransmissions returns paginated recent transmissions
// GET /api/gamification/recent-transmissions?limit=50&offset=0
func (g *GamificationAPI) RecentTransmissions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse parameters
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}

	// Get recent logs
	logs, err := g.txLogRepo.GetRecentLogs(limit)
	if err != nil {
		http.Error(w, "Failed to get transmissions", http.StatusInternalServerError)
		return
	}

	// Convert to response format
	type TransmissionEntry struct {
		Callsign        string `json:"callsign"`
		Node            int    `json:"node"`
		TimestampStart  string `json:"timestamp_start"`
		DurationSeconds int    `json:"duration_seconds"`
	}

	var entries []TransmissionEntry
	for _, log := range logs {
		// Ensure we emit a correctly labeled UTC timestamp in RFC3339 format
		ts := log.TimestampStart.UTC().Format(time.RFC3339)
		entries = append(entries, TransmissionEntry{
			Callsign:        log.Callsign,
			Node:            log.AdjacentLinkID,
			TimestampStart:  ts,
			DurationSeconds: log.DurationSeconds,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"transmissions": entries,
		"limit":         limit,
	})
}

// LevelConfig returns the level configuration (XP requirements per level)
// GET /api/gamification/level-config
func (g *GamificationAPI) LevelConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := context.Background()

	levelConfig, err := g.levelRepo.GetAllAsMap(ctx)
	if err != nil {
		http.Error(w, "Failed to get level config", http.StatusInternalServerError)
		return
	}

	// Build groupings map for quick lookup
	groupingsMap := gamification.BuildGroupingsMap(g.levelGroupings)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"config":    levelConfig,
		"groupings": groupingsMap,
	})
}
