package tests

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/dbehnke/allstar-nexus/backend/api"
	cfgpkg "github.com/dbehnke/allstar-nexus/backend/config"
	"github.com/dbehnke/allstar-nexus/backend/gamification"
	"github.com/dbehnke/allstar-nexus/backend/models"
	"github.com/dbehnke/allstar-nexus/backend/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	_ "modernc.org/sqlite"
)

func setupLevelingThresholdsTest(t *testing.T) (*gorm.DB, *api.GamificationAPI) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "leveling_test.db")
	gdb, err := gorm.Open(sqlite.New(sqlite.Config{DriverName: "sqlite", DSN: dbPath}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open gorm sqlite: %v", err)
	}
	if err := gdb.AutoMigrate(&models.LevelConfig{}); err != nil {
		t.Fatalf("automigrate: %v", err)
	}

	// Seed level requirements using authoritative calculation
	levelRepo := repository.NewLevelConfigRepo(gdb)
	reqs := gamification.CalculateLevelRequirements()
	if err := levelRepo.SeedDefaults(context.Background(), reqs); err != nil {
		t.Fatalf("seed level config: %v", err)
	}

	// Create API instance
	profileRepo := repository.NewCallsignProfileRepo(gdb)
	txLogRepo := repository.NewTransmissionLogRepository(gdb)
	activityRepo := repository.NewXPActivityRepo(gdb)

	gamificationAPI := api.NewGamificationAPI(
		profileRepo,
		txLogRepo,
		levelRepo,
		activityRepo,
		[]cfgpkg.LevelGrouping{},
		true,  // renown enabled
		36000, // renown XP
		false, // rested disabled
		0,     // rested accumulation
		0,     // rested max
		1.0,   // rested multiplier
		0,     // rested idle threshold
		0,     // weekly cap
		0,     // daily cap
		[]cfgpkg.DRTier{},
	)

	return gdb, gamificationAPI
}

func TestLevelingThresholds_Default(t *testing.T) {
	_, gamificationAPI := setupLevelingThresholdsTest(t)

	req := httptest.NewRequest(http.MethodGet, "/api/leveling/thresholds", nil)
	w := httptest.NewRecorder()

	gamificationAPI.LevelingThresholds(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	// Parse response
	var resp struct {
		Levels []struct {
			Level int `json:"level"`
			XP    int `json:"xp"`
		} `json:"levels"`
		Calculation string `json:"calculation"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Should return 60 levels by default
	if len(resp.Levels) != 60 {
		t.Errorf("expected 60 levels, got %d", len(resp.Levels))
	}

	// Verify calculation field
	if resp.Calculation == "" {
		t.Error("expected calculation field to be non-empty")
	}

	// Verify first level is 1 with 360 XP (6 minutes)
	if len(resp.Levels) > 0 {
		if resp.Levels[0].Level != 1 {
			t.Errorf("expected first level to be 1, got %d", resp.Levels[0].Level)
		}
		if resp.Levels[0].XP != 360 {
			t.Errorf("expected level 1 XP to be 360, got %d", resp.Levels[0].XP)
		}
	}

	// Verify level 10 is 360 XP
	if len(resp.Levels) >= 10 {
		if resp.Levels[9].Level != 10 || resp.Levels[9].XP != 360 {
			t.Errorf("expected level 10 to have 360 XP, got level=%d xp=%d", resp.Levels[9].Level, resp.Levels[9].XP)
		}
	}
}

func TestLevelingThresholds_MaxLevel(t *testing.T) {
	_, gamificationAPI := setupLevelingThresholdsTest(t)

	req := httptest.NewRequest(http.MethodGet, "/api/leveling/thresholds?max_level=15", nil)
	w := httptest.NewRecorder()

	gamificationAPI.LevelingThresholds(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp struct {
		Levels []struct {
			Level int `json:"level"`
			XP    int `json:"xp"`
		} `json:"levels"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Should return 15 levels
	if len(resp.Levels) != 15 {
		t.Errorf("expected 15 levels with max_level=15, got %d", len(resp.Levels))
	}

	// Verify last level is 15
	if len(resp.Levels) > 0 {
		lastLevel := resp.Levels[len(resp.Levels)-1]
		if lastLevel.Level != 15 {
			t.Errorf("expected last level to be 15, got %d", lastLevel.Level)
		}
	}
}

func TestLevelingThresholds_SpecificLevels(t *testing.T) {
	_, gamificationAPI := setupLevelingThresholdsTest(t)

	req := httptest.NewRequest(http.MethodGet, "/api/leveling/thresholds?levels=5,10,20", nil)
	w := httptest.NewRecorder()

	gamificationAPI.LevelingThresholds(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp struct {
		Levels []struct {
			Level int `json:"level"`
			XP    int `json:"xp"`
		} `json:"levels"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Should return exactly 3 levels
	if len(resp.Levels) != 3 {
		t.Errorf("expected 3 levels, got %d", len(resp.Levels))
	}

	// Verify the levels returned are 5, 10, and 20
	expectedLevels := map[int]bool{5: true, 10: true, 20: true}
	for _, lvl := range resp.Levels {
		if !expectedLevels[lvl.Level] {
			t.Errorf("unexpected level %d in response", lvl.Level)
		}
	}
}

func TestLevelingThresholds_CacheHeaders(t *testing.T) {
	_, gamificationAPI := setupLevelingThresholdsTest(t)

	req := httptest.NewRequest(http.MethodGet, "/api/leveling/thresholds", nil)
	w := httptest.NewRecorder()

	gamificationAPI.LevelingThresholds(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	// Verify Cache-Control header is set
	cacheControl := w.Header().Get("Cache-Control")
	if cacheControl != "public, max-age=300" {
		t.Errorf("expected Cache-Control header 'public, max-age=300', got '%s'", cacheControl)
	}
}

func TestLevelingThresholds_MethodNotAllowed(t *testing.T) {
	_, gamificationAPI := setupLevelingThresholdsTest(t)

	req := httptest.NewRequest(http.MethodPost, "/api/leveling/thresholds", nil)
	w := httptest.NewRecorder()

	gamificationAPI.LevelingThresholds(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}
