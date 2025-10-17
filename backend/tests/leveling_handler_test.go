package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dbehnke/allstar-nexus/backend/gamification"
	"github.com/dbehnke/allstar-nexus/backend/handlers"
)

func TestLevelingThresholdsHandler(t *testing.T) {
	// Set up test data - precompute level requirements
	sampleMap := map[int]int{
		1:  100,
		2:  200,
		15: 1500,
		16: 1600,
		60: 6000,
	}
	gamification.SetLevelRequirements(sampleMap)

	// Test with specific levels query parameter
	req := httptest.NewRequest(http.MethodGet, "/api/leveling/thresholds?levels=15,16,60", nil)
	w := httptest.NewRecorder()

	handlers.LevelingThresholdsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var response struct {
		Levels []struct {
			Level int `json:"level"`
			XP    int `json:"xp"`
		} `json:"levels"`
		Calculation string `json:"calculation"`
		Metadata    struct {
			CalcSource string `json:"calculation_source"`
		} `json:"metadata"`
	}

	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Verify we got 3 levels back
	if len(response.Levels) != 3 {
		t.Errorf("expected 3 levels, got %d", len(response.Levels))
	}

	// Verify the XP values match our sample map
	expectedLevels := map[int]int{
		15: 1500,
		16: 1600,
		60: 6000,
	}

	for _, levelData := range response.Levels {
		expectedXP, ok := expectedLevels[levelData.Level]
		if !ok {
			t.Errorf("unexpected level %d in response", levelData.Level)
			continue
		}
		if levelData.XP != expectedXP {
			t.Errorf("level %d: expected XP %d, got %d", levelData.Level, expectedXP, levelData.XP)
		}
	}

	// Verify calculation metadata indicates precomputed
	if response.Metadata.CalcSource != "precomputed" {
		t.Errorf("expected calc_source 'precomputed', got '%s'", response.Metadata.CalcSource)
	}
}

func TestLevelingThresholdsHandler_UnconfiguredLevel(t *testing.T) {
	// Set up test data with only a few levels
	sampleMap := map[int]int{
		1: 100,
		2: 200,
	}
	gamification.SetLevelRequirements(sampleMap)

	// Request a level that's not in the map (should return 0)
	req := httptest.NewRequest(http.MethodGet, "/api/leveling/thresholds?levels=99", nil)
	w := httptest.NewRecorder()

	handlers.LevelingThresholdsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var response struct {
		Levels []struct {
			Level int `json:"level"`
			XP    int `json:"xp"`
		} `json:"levels"`
	}

	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Should return level 99 with XP of 0 (placeholder for unconfigured levels)
	if len(response.Levels) != 1 {
		t.Fatalf("expected 1 level, got %d", len(response.Levels))
	}

	if response.Levels[0].Level != 99 {
		t.Errorf("expected level 99, got %d", response.Levels[0].Level)
	}

	if response.Levels[0].XP != 0 {
		t.Errorf("expected XP 0 for unconfigured level, got %d", response.Levels[0].XP)
	}
}

func TestLevelingThresholdsHandler_NoLevelsParam(t *testing.T) {
	// Set up test data
	sampleMap := map[int]int{
		1: 100,
		2: 200,
		3: 300,
	}
	gamification.SetLevelRequirements(sampleMap)

	// Request without levels parameter (should return all levels up to max_level or 60)
	req := httptest.NewRequest(http.MethodGet, "/api/leveling/thresholds", nil)
	w := httptest.NewRecorder()

	handlers.LevelingThresholdsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var response struct {
		Levels []struct {
			Level int `json:"level"`
			XP    int `json:"xp"`
		} `json:"levels"`
	}

	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Should return all levels from 1 to 60 (default max_level)
	if len(response.Levels) != 60 {
		t.Errorf("expected 60 levels, got %d", len(response.Levels))
	}
}

func TestLevelingThresholdsHandler_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/leveling/thresholds", nil)
	w := httptest.NewRecorder()

	handlers.LevelingThresholdsHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}
