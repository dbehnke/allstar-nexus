package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dbehnke/allstar-nexus/backend/gamification"
)

// Response structures
type LevelThreshold struct {
	Level int `json:"level"`
	XP    int `json:"xp"`
}

type Metadata struct {
	APIVersion   string `json:"api_version"`
	GeneratedAt  string `json:"generated_at"`
	CacheMaxAge  int    `json:"cache_max_age_seconds"`
	CalcSource   string `json:"calculation_source"`
}

type ThresholdsResponse struct {
	Levels      []LevelThreshold `json:"levels"`
	Calculation string           `json:"calculation"`
	Metadata    Metadata         `json:"metadata"`
}

// LevelingThresholdsHandler returns authoritative level XP thresholds computed
// at startup and stored in gamification package.
func LevelingThresholdsHandler(w http.ResponseWriter, r *http.Request) {
	// Only accept GET requests
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query()

	// default max level
	maxLevel := 60
	if s := q.Get("max_level"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			maxLevel = v
		}
	}

	levelsParam := q.Get("levels")
	var requested []int
	if levelsParam != "" {
		parts := strings.Split(levelsParam, ",")
		for _, p := range parts {
			if v, err := strconv.Atoi(strings.TrimSpace(p)); err == nil && v > 0 {
				requested = append(requested, v)
			}
		}
	}

	// Build list of levels to return
	var levels []int
	if len(requested) > 0 {
		levels = requested
	} else {
		for i := 1; i <= maxLevel; i++ {
			levels = append(levels, i)
		}
	}

	// Read precomputed requirements (copy)
	requirements := gamification.GetLevelRequirements()

	var resp ThresholdsResponse
	for _, lvl := range levels {
		if lvl <= 1 {
			resp.Levels = append(resp.Levels, LevelThreshold{Level: lvl, XP: 0})
			continue
		}
		if xp, ok := requirements[lvl]; ok {
			resp.Levels = append(resp.Levels, LevelThreshold{Level: lvl, XP: xp})
		} else {
			// Level beyond configured range (e.g., renown) -> return 0 as placeholder
			resp.Levels = append(resp.Levels, LevelThreshold{Level: lvl, XP: 0})
		}
	}

	// Cache and version metadata
	cacheMaxAge := 3600 // seconds (1 hour)
	generatedAt := time.Now().UTC().Format(time.RFC3339)
	resp.Calculation = "precomputed via gamification.SetLevelRequirements() (authoritative)"
	resp.Metadata = Metadata{
		APIVersion:  "v1",
		GeneratedAt: generatedAt,
		CacheMaxAge: cacheMaxAge,
		CalcSource:  "precomputed",
	}

	// Set caching and version headers
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-API-Version", "v1")
	w.Header().Set("X-Calculation-Generated-At", generatedAt)
	w.Header().Set("X-Cache-TTL", strconv.Itoa(cacheMaxAge))
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}