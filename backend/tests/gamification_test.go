package tests

import (
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "path/filepath"
    "testing"
    "time"

    "github.com/dbehnke/allstar-nexus/backend/api"
    "github.com/dbehnke/allstar-nexus/backend/gamification"
    "github.com/dbehnke/allstar-nexus/backend/models"
    "github.com/dbehnke/allstar-nexus/backend/repository"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
)

// testGamificationServer spins up minimal HTTP mux exposing gamification endpoints backed by GORM sqlite temp DB.
func testGamificationServer(t *testing.T) (*httptest.Server, *gorm.DB, func()) {
    t.Helper()
    dir := t.TempDir()
    dbPath := filepath.Join(dir, "test_gamification.db")
    gdb, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
    if err != nil {
        t.Fatalf("open gorm sqlite: %v", err)
    }
    // Auto-migrate necessary models
    if err := gdb.AutoMigrate(
        &models.CallsignProfile{},
        &models.LevelConfig{},
        &models.TransmissionLog{},
        &models.XPActivityLog{},
    ); err != nil {
        t.Fatalf("automigrate: %v", err)
    }

    levelRepo := repository.NewLevelConfigRepo(gdb)
    profileRepo := repository.NewCallsignProfileRepo(gdb)
    txRepo := repository.NewTransmissionLogRepository(gdb)
    activityRepo := repository.NewXPActivityRepo(gdb)

    // Seed level config with default requirements to satisfy endpoints
    reqs := gamification.CalculateLevelRequirements()
    if err := levelRepo.SeedDefaults(context.Background(), reqs); err != nil {
        t.Fatalf("seed level config: %v", err)
    }

    gapi := api.NewGamificationAPI(profileRepo, txRepo, levelRepo, activityRepo)

    mux := http.NewServeMux()
    mux.HandleFunc("/api/gamification/scoreboard", gapi.Scoreboard)
    mux.HandleFunc("/api/gamification/recent-transmissions", gapi.RecentTransmissions)
    mux.HandleFunc("/api/gamification/level-config", gapi.LevelConfig)

    srv := httptest.NewServer(mux)
    cleanup := func() {
        srv.Close()
    }
    return srv, gdb, cleanup
}

func TestLevelConfigEndpoint_ReturnsConfig(t *testing.T) {
    srv, _, cleanup := testGamificationServer(t)
    defer cleanup()

    resp, err := http.Get(srv.URL + "/api/gamification/level-config")
    if err != nil {
        t.Fatalf("http get: %v", err)
    }
    defer resp.Body.Close()
    if resp.StatusCode != 200 {
        t.Fatalf("status %d", resp.StatusCode)
    }
    var payload map[string]map[string]int
    if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
        t.Fatalf("decode: %v", err)
    }
    if len(payload["config"]) == 0 {
        t.Fatalf("expected non-empty config")
    }
}

func TestScoreboardOrdering_ByRenownLevelThenLevelThenXP(t *testing.T) {
    srv, gdb, cleanup := testGamificationServer(t)
    defer cleanup()

    // Seed some profiles
    profiles := []models.CallsignProfile{
        {Callsign: "K1AAA", Level: 10, ExperiencePoints: 100, RenownLevel: 0},
        {Callsign: "K2BBB", Level: 9, ExperiencePoints: 900, RenownLevel: 0},
        {Callsign: "K3CCC", Level: 5, ExperiencePoints: 200, RenownLevel: 1}, // should be first due to renown
    }
    for i := range profiles {
        if err := gdb.Create(&profiles[i]).Error; err != nil {
            t.Fatalf("seed profile: %v", err)
        }
    }

    // Call endpoint
    resp, err := http.Get(srv.URL + "/api/gamification/scoreboard?limit=10")
    if err != nil {
        t.Fatalf("http get: %v", err)
    }
    defer resp.Body.Close()
    if resp.StatusCode != 200 {
        t.Fatalf("status %d", resp.StatusCode)
    }
    var payload struct {
        Scoreboard []struct {
            Rank     int    `json:"rank"`
            Callsign string `json:"callsign"`
            Level    int    `json:"level"`
            Renown   int    `json:"renown_level"`
            XP       int    `json:"experience_points"`
        } `json:"scoreboard"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
        t.Fatalf("decode: %v", err)
    }
    if len(payload.Scoreboard) < 3 {
        t.Fatalf("expected at least 3 entries")
    }
    // Expect K3CCC first due to renown=1
    if payload.Scoreboard[0].Callsign != "K3CCC" {
        t.Fatalf("expected first callsign K3CCC, got %s", payload.Scoreboard[0].Callsign)
    }
    // Between K1AAA (L10, XP100) and K2BBB (L9, XP900): level should win
    if payload.Scoreboard[1].Callsign != "K1AAA" {
        t.Fatalf("expected second callsign K1AAA, got %s", payload.Scoreboard[1].Callsign)
    }
}

func TestRecentTransmissions_ReturnsLimitedList(t *testing.T) {
    srv, gdb, cleanup := testGamificationServer(t)
    defer cleanup()

    txRepo := repository.NewTransmissionLogRepository(gdb)
    now := time.Now().Add(-time.Hour)
    // Seed 3 logs
    for i := 0; i < 3; i++ {
        _ = txRepo.LogTransmission(1000+i, 2000+i, "K1AAA", now.Add(time.Duration(i)*time.Minute), now.Add(time.Duration(i)*time.Minute+30*time.Second), 30)
    }

    resp, err := http.Get(srv.URL + "/api/gamification/recent-transmissions?limit=2")
    if err != nil {
        t.Fatalf("http get: %v", err)
    }
    defer resp.Body.Close()
    if resp.StatusCode != 200 {
        t.Fatalf("status %d", resp.StatusCode)
    }
    var payload struct {
        Transmissions []any `json:"transmissions"`
        Limit         int   `json:"limit"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
        t.Fatalf("decode: %v", err)
    }
    if payload.Limit != 2 || len(payload.Transmissions) != 2 {
        t.Fatalf("expected 2 transmissions, got %d (limit=%d)", len(payload.Transmissions), payload.Limit)
    }
}
