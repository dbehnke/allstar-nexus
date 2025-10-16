package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/dbehnke/allstar-nexus/backend/gamification"
)

// LevelingThresholdsHandler returns level thresholds (XP requirements per level)
// It uses precomputed values from gamification package
// Query parameters:
//   - max_level: int (default 60) - maximum level to include
//   - levels: comma-separated list of specific levels (e.g., "15,16,60")
//
// GET /api/leveling-thresholds?levels=15,16,60
// GET /api/leveling-thresholds?max_level=30
func LevelingThresholdsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get precomputed requirements
	requirements := gamification.GetLevelRequirements()
	if requirements == nil {
		http.Error(w, "Level requirements not initialized", http.StatusInternalServerError)
		return
	}

	// Parse query parameters
	maxLevel := 60
	if maxLevelStr := r.URL.Query().Get("max_level"); maxLevelStr != "" {
		if parsed, err := strconv.Atoi(maxLevelStr); err == nil && parsed > 0 && parsed <= 60 {
			maxLevel = parsed
		}
	}

	// Check if specific levels were requested
	levelsParam := r.URL.Query().Get("levels")
	var result map[int]int

	if levelsParam != "" {
		// Parse comma-separated list of levels
		result = make(map[int]int)
		levelStrs := strings.Split(levelsParam, ",")
		for _, levelStr := range levelStrs {
			levelStr = strings.TrimSpace(levelStr)
			if level, err := strconv.Atoi(levelStr); err == nil && level >= 1 && level <= 60 {
				if xp, ok := requirements[level]; ok {
					result[level] = xp
				}
			}
		}
	} else {
		// Return all levels up to max_level
		result = make(map[int]int)
		for level := 1; level <= maxLevel; level++ {
			if xp, ok := requirements[level]; ok {
				result[level] = xp
			}
		}
	}

	// Build response
	response := map[string]interface{}{
		"levels":      result,
		"calculation": "precomputed",
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
