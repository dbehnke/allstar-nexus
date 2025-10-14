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

	_ "modernc.org/sqlite"
)

// setupDBForProfileTest initializes a sqlite DB with required migrations and repos
func setupDBForProfileTest(t *testing.T) (*gorm.DB, *repository.LevelConfigRepo, *repository.CallsignProfileRepo, *repository.TransmissionLogRepository, *repository.XPActivityRepo) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "profile_api_test.db")
	gdb, err := gorm.Open(sqlite.New(sqlite.Config{
		DriverName: "sqlite",
		DSN:        dbPath,
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open gorm sqlite: %v", err)
	}
	if err := gdb.AutoMigrate(
		&models.CallsignProfile{},
		&models.LevelConfig{},
		&models.TransmissionLog{},
		&models.XPActivityLog{},
		&models.TallyState{},
	); err != nil {
		t.Fatalf("automigrate: %v", err)
	}

	return gdb,
		repository.NewLevelConfigRepo(gdb),
		repository.NewCallsignProfileRepo(gdb),
		repository.NewTransmissionLogRepository(gdb),
		repository.NewXPActivityRepo(gdb)
}

func TestProfileAPI_ReturnsAccurateAggregates(t *testing.T) {
	gdb, levelRepo, profileRepo, txRepo, activityRepo := setupDBForProfileTest(t)
	stateRepo := repository.NewTallyStateRepo(gdb)

	// Seed default level requirements
	reqs := gamification.CalculateLevelRequirements()
	if err := levelRepo.SeedDefaults(context.Background(), reqs); err != nil {
		t.Fatalf("seed level config: %v", err)
	}

	// Create transmissions for a single callsign (all today)
	callsign := "N0UNIT"
	start := time.Now().Add(-15 * time.Minute)
	durations := []int{30, 40, 50} // total 120s
	total := 0
	for i, d := range durations {
		total += d
		s := start.Add(time.Duration(i) * time.Minute)
		e := s.Add(time.Duration(d) * time.Second)
		if err := txRepo.LogTransmission(100+i, 200+i, callsign, s, e, d); err != nil {
			t.Fatalf("seed tx: %v", err)
		}
	}

	// Ensure profile exists
	if _, err := profileRepo.GetByCallsign(context.Background(), callsign); err != nil {
		t.Fatalf("get profile: %v", err)
	}

	// Run tally once with all multipliers disabled to make awardedXP == rawXP
	cfg := &gamification.Config{
		RestedEnabled:          false,
		DREnabled:              false,
		KerchunkEnabled:        false,
		CapsEnabled:            false,
		RestedAccumulationRate: 0,
		RestedMaxSeconds:       0,
		RestedMultiplier:       1.0,
	}
	ts := gamification.NewTallyService(gdb, txRepo, profileRepo, levelRepo, activityRepo, stateRepo, cfg, 30*time.Minute, zaptestLogger())
	if err := ts.Start(); err != nil {
		t.Fatalf("start tally: %v", err)
	}
	defer ts.Stop()
	if err := ts.ProcessTally(); err != nil {
		t.Fatalf("process tally: %v", err)
	}

	// Build API with profile route and query it
	gapi := api.NewGamificationAPI(profileRepo, txRepo, levelRepo, activityRepo, gamification.DefaultLevelGroupings(), true, 36000, false, 0, 0, 1.0)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/gamification/profile/", gapi.Profile)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/gamification/profile/" + callsign)
	if err != nil {
		t.Fatalf("http get: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != 200 {
		t.Fatalf("status: %d", resp.StatusCode)
	}

	var payload struct {
		Callsign             string `json:"callsign"`
		Level                int    `json:"level"`
		ExperiencePoints     int    `json:"experience_points"`
		RenownLevel          int    `json:"renown_level"`
		NextLevelXP          int    `json:"next_level_xp"`
		TotalTalkTimeSeconds int    `json:"total_talk_time_seconds"`
		RestedBonusSeconds   int    `json:"rested_bonus_seconds"`
		WeeklyXP             int    `json:"weekly_xp"`
		DailyXP              int    `json:"daily_xp"`
		DailyBreakdown       []struct {
			Date      string `json:"date"`
			RawXP     int    `json:"raw_xp"`
			AwardedXP int    `json:"awarded_xp"`
		} `json:"daily_breakdown"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if payload.Callsign != callsign {
		t.Fatalf("callsign expected %s got %s", callsign, payload.Callsign)
	}

	if payload.ExperiencePoints != total {
		t.Fatalf("xp expected %d got %d", total, payload.ExperiencePoints)
	}

	if payload.WeeklyXP != total || payload.DailyXP != total {
		t.Fatalf("weekly/daily xp expected %d got weekly=%d daily=%d", total, payload.WeeklyXP, payload.DailyXP)
	}

	if payload.TotalTalkTimeSeconds != total {
		t.Fatalf("total talk time expected %d got %d", total, payload.TotalTalkTimeSeconds)
	}

	// With defaults, next level from Level 1 is 360 XP
	if payload.Level != 1 || payload.NextLevelXP != 360 {
		t.Fatalf("expected level=1 next_level_xp=360 got level=%d next=%d", payload.Level, payload.NextLevelXP)
	}

	// Rested disabled -> rested seconds should be 0
	if payload.RestedBonusSeconds != 0 {
		t.Fatalf("expected rested 0 got %d", payload.RestedBonusSeconds)
	}

	// Breakdown should have at least an entry for today with awarded sum
	if len(payload.DailyBreakdown) == 0 {
		t.Fatalf("expected non-empty daily breakdown")
	}
	// Sum awards across breakdown should be >= total (at least today's)
	sumBreakdown := 0
	for _, d := range payload.DailyBreakdown {
		sumBreakdown += d.AwardedXP
	}
	if sumBreakdown < total {
		t.Fatalf("breakdown awarded sum %d < total %d", sumBreakdown, total)
	}
}
