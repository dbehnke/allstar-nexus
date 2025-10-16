package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dbehnke/allstar-nexus/backend/gamification"
	"github.com/dbehnke/allstar-nexus/backend/handlers"
)

func TestLevelingThresholdsHandler_SpecificLevels(t *testing.T) {
	// Setup: precompute a sample map
	sampleMap := map[int]int{
		1:  100,
		15: 1500,
		16: 1600,
		60: 6000,
	}
	gamification.SetLevelRequirements(sampleMap)

	// Create request with specific levels
	req := httptest.NewRequest(http.MethodGet, "/api/leveling-thresholds?levels=15,16,60", nil)
	w := httptest.NewRecorder()

	// Call handler
	handlers.LevelingThresholdsHandler(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Verify response structure
	if response["calculation"] != "precomputed" {
		t.Errorf("expected calculation='precomputed', got %v", response["calculation"])
	}

	// Verify levels returned
	levels, ok := response["levels"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected levels to be a map")
	}

	// Check specific values (JSON decoding gives us float64 for numbers)
	expectedLevels := map[string]float64{
		"15": 1500,
		"16": 1600,
		"60": 6000,
	}

	for levelStr, expectedXP := range expectedLevels {
		actualXP, ok := levels[levelStr].(float64)
		if !ok {
			t.Errorf("level %s not found in response", levelStr)
			continue
		}
		if actualXP != expectedXP {
			t.Errorf("level %s: expected XP=%v, got %v", levelStr, expectedXP, actualXP)
		}
	}

	// Ensure level 1 is not in response (wasn't requested)
	if _, ok := levels["1"]; ok {
		t.Errorf("level 1 should not be in response when levels=15,16,60")
	}
}

func TestLevelingThresholdsHandler_MaxLevel(t *testing.T) {
	// Setup: precompute a sample map
	sampleMap := map[int]int{
		1: 100,
		2: 200,
		3: 300,
		4: 400,
		5: 500,
	}
	gamification.SetLevelRequirements(sampleMap)

	// Create request with max_level=3
	req := httptest.NewRequest(http.MethodGet, "/api/leveling-thresholds?max_level=3", nil)
	w := httptest.NewRecorder()

	// Call handler
	handlers.LevelingThresholdsHandler(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Verify levels returned
	levels, ok := response["levels"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected levels to be a map")
	}

	// Should have levels 1, 2, 3
	if len(levels) != 3 {
		t.Errorf("expected 3 levels, got %d", len(levels))
	}

	// Check that levels 4 and 5 are not included
	if _, ok := levels["4"]; ok {
		t.Errorf("level 4 should not be in response when max_level=3")
	}
	if _, ok := levels["5"]; ok {
		t.Errorf("level 5 should not be in response when max_level=3")
	}
}

func TestLevelingThresholdsHandler_Default(t *testing.T) {
	// Setup: precompute requirements with default calculation
	requirements := gamification.CalculateLevelRequirements()
	gamification.SetLevelRequirements(requirements)

	// Create request with no parameters
	req := httptest.NewRequest(http.MethodGet, "/api/leveling-thresholds", nil)
	w := httptest.NewRecorder()

	// Call handler
	handlers.LevelingThresholdsHandler(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Verify calculation field
	if response["calculation"] != "precomputed" {
		t.Errorf("expected calculation='precomputed', got %v", response["calculation"])
	}

	// Verify we got all 60 levels
	levels, ok := response["levels"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected levels to be a map")
	}

	if len(levels) != 60 {
		t.Errorf("expected 60 levels by default, got %d", len(levels))
	}
}

func TestLevelingThresholdsHandler_MethodNotAllowed(t *testing.T) {
	// Try POST request
	req := httptest.NewRequest(http.MethodPost, "/api/leveling-thresholds", nil)
	w := httptest.NewRecorder()

	handlers.LevelingThresholdsHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}
