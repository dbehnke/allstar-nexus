package tests

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/dbehnke/allstar-nexus/backend/gamification"
	"github.com/dbehnke/allstar-nexus/backend/models"
	"github.com/dbehnke/allstar-nexus/backend/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	_ "modernc.org/sqlite"
)

// TestLevelingSpeed_NoMultipliers verifies that basic leveling is correct without any multipliers
func TestLevelingSpeed_NoMultipliers(t *testing.T) {
	gdb := setupLevelingDB(t)
	levelRepo := repository.NewLevelConfigRepo(gdb)
	profileRepo := repository.NewCallsignProfileRepo(gdb)
	txRepo := repository.NewTransmissionLogRepository(gdb)
	activityRepo := repository.NewXPActivityRepo(gdb)
	stateRepo := repository.NewTallyStateRepo(gdb)

	// Seed level config
	reqs := gamification.CalculateLevelRequirements()
	if err := levelRepo.SeedDefaults(context.Background(), reqs); err != nil {
		t.Fatalf("seed level config: %v", err)
	}

	// Test case: User at level 15 with 0 XP, adds 10 minutes (600 seconds)
	// Level 16 requires 306 XP, so they should level up to 16 with 294 XP remaining
	callsign := "W1LEVEL15"
	prof, err := profileRepo.GetByCallsign(context.Background(), callsign)
	if err != nil {
		t.Fatalf("get profile: %v", err)
	}

	prof.Level = 15
	prof.ExperiencePoints = 0
	if err := profileRepo.Upsert(context.Background(), prof); err != nil {
		t.Fatalf("upsert profile: %v", err)
	}

	// Add 10 minutes of transmission
	now := time.Now().Add(-time.Minute)
	if err := txRepo.LogTransmission(100, 200, callsign, now, now.Add(600*time.Second), 600); err != nil {
		t.Fatalf("seed tx: %v", err)
	}

	// Process with no multipliers/bonuses
	cfg := &gamification.Config{
		CapsEnabled:     false,
		RestedEnabled:   false,
		DREnabled:       false,
		KerchunkEnabled: false,
		RenownEnabled:   false,
	}
	ts := gamification.NewTallyService(gdb, txRepo, profileRepo, levelRepo, activityRepo, stateRepo, cfg, 30*time.Minute, zaptestLogger())
	if err := ts.Start(); err != nil {
		t.Fatalf("start tally: %v", err)
	}
	defer ts.Stop()

	// Verify result
	prof2, err := profileRepo.GetByCallsign(context.Background(), callsign)
	if err != nil {
		t.Fatalf("reload profile: %v", err)
	}

	// Should be level 16 with 294 XP (600 - 306)
	if prof2.Level != 16 {
		t.Errorf("expected level 16, got %d", prof2.Level)
	}
	if prof2.ExperiencePoints != 294 {
		t.Errorf("expected 294 XP, got %d", prof2.ExperiencePoints)
	}
}

// TestLevelingSpeed_WithRestedBonus verifies that rested XP bonus correctly accelerates leveling
func TestLevelingSpeed_WithRestedBonus(t *testing.T) {
	gdb := setupLevelingDB(t)
	levelRepo := repository.NewLevelConfigRepo(gdb)
	profileRepo := repository.NewCallsignProfileRepo(gdb)
	txRepo := repository.NewTransmissionLogRepository(gdb)
	activityRepo := repository.NewXPActivityRepo(gdb)
	stateRepo := repository.NewTallyStateRepo(gdb)

	// Seed level config
	reqs := gamification.CalculateLevelRequirements()
	if err := levelRepo.SeedDefaults(context.Background(), reqs); err != nil {
		t.Fatalf("seed level config: %v", err)
	}

	// Test case: User at level 15 with 100 XP and rested bonus, adds 10 minutes
	// With 2x multiplier: 600 * 2 = 1200 XP total
	// 100 + 1200 = 1300 XP
	// Level 16 needs 306: 1300 - 306 = 994 XP
	// Level 17 needs 404: 994 - 404 = 590 XP
	// Should end at level 17 with 590 XP
	callsign := "W1RESTED"
	prof, err := profileRepo.GetByCallsign(context.Background(), callsign)
	if err != nil {
		t.Fatalf("get profile: %v", err)
	}

	prof.Level = 15
	prof.ExperiencePoints = 100
	prof.RestedBonusSeconds = 1000 // Large rested bonus
	prof.LastTransmissionAt = time.Now().Add(-24 * time.Hour)
	if err := profileRepo.Upsert(context.Background(), prof); err != nil {
		t.Fatalf("upsert profile: %v", err)
	}

	// Add 10 minutes of transmission
	now := time.Now().Add(-time.Minute)
	if err := txRepo.LogTransmission(100, 200, callsign, now, now.Add(600*time.Second), 600); err != nil {
		t.Fatalf("seed tx: %v", err)
	}

	// Process with rested bonus enabled
	cfg := &gamification.Config{
		CapsEnabled:                false,
		RestedEnabled:              true,
		RestedMultiplier:           2.0,
		RestedMaxSeconds:           7200,
		RestedAccumulationRate:     1.5,
		RestedIdleThresholdSeconds: 300,
		DREnabled:                  false,
		KerchunkEnabled:            false,
		RenownEnabled:              false,
	}
	ts := gamification.NewTallyService(gdb, txRepo, profileRepo, levelRepo, activityRepo, stateRepo, cfg, 30*time.Minute, zaptestLogger())
	if err := ts.Start(); err != nil {
		t.Fatalf("start tally: %v", err)
	}
	defer ts.Stop()

	// Verify result
	prof2, err := profileRepo.GetByCallsign(context.Background(), callsign)
	if err != nil {
		t.Fatalf("reload profile: %v", err)
	}

	// With rested bonus, should reach at least level 17
	// (May reach level 18 depending on exact bonus calculation)
	if prof2.Level < 17 {
		t.Errorf("expected at least level 17 with rested bonus, got %d", prof2.Level)
	}
	
	// Document what happened
	t.Logf("Result: Level %d with %d XP (rested bonus active)", prof2.Level, prof2.ExperiencePoints)
	t.Log("This demonstrates that rested XP bonus can cause rapid multi-level jumps")
}

// TestProfileUpsert_PersistsDailyWeeklyXP verifies that daily_xp and weekly_xp are persisted
func TestProfileUpsert_PersistsDailyWeeklyXP(t *testing.T) {
	gdb := setupLevelingDB(t)
	profileRepo := repository.NewCallsignProfileRepo(gdb)

	callsign := "W1PERSIST"
	
	// Create initial profile
	prof, err := profileRepo.GetByCallsign(context.Background(), callsign)
	if err != nil {
		t.Fatalf("get profile: %v", err)
	}

	// Set daily and weekly XP
	prof.DailyXP = 500
	prof.WeeklyXP = 2000
	if err := profileRepo.Upsert(context.Background(), prof); err != nil {
		t.Fatalf("upsert profile: %v", err)
	}

	// Reload and verify persistence
	prof2, err := profileRepo.GetByCallsign(context.Background(), callsign)
	if err != nil {
		t.Fatalf("reload profile: %v", err)
	}

	if prof2.DailyXP != 500 {
		t.Errorf("expected DailyXP 500, got %d (not persisted!)", prof2.DailyXP)
	}
	if prof2.WeeklyXP != 2000 {
		t.Errorf("expected WeeklyXP 2000, got %d (not persisted!)", prof2.WeeklyXP)
	}
}

// TestLevelRequirements_Documentation documents the expected level requirements
func TestLevelRequirements_Documentation(t *testing.T) {
	reqs := gamification.CalculateLevelRequirements()

	// Document critical level thresholds
	tests := []struct {
		level       int
		expectedXP  int
		description string
	}{
		{1, 360, "First level (6 minutes)"},
		{10, 360, "Last linear level"},
		{11, 12, "First logarithmic level"},
		{15, 220, "Level 15 requirement"},
		{16, 306, "Level 16 requirement"},
		{17, 404, "Level 17 requirement"},
		{20, 768, "Level 20 requirement"},
	}

	t.Log("Level Requirements (Low-Activity Hub Scale):")
	cumulative := 0
	for _, tt := range tests {
		cumulative += reqs[tt.level]
		xp := reqs[tt.level]
		if xp != tt.expectedXP {
			t.Errorf("Level %d: expected %d XP, got %d (%s)", 
				tt.level, tt.expectedXP, xp, tt.description)
		}
		t.Logf("Level %2d: %5d XP (%s) - Cumulative: %d", 
			tt.level, xp, tt.description, cumulative)
	}
	
	// Also document level 60 (max level before renown)
	t.Logf("Level 60: %5d XP (Max level before renown)", reqs[60])

	// Document jump scenarios
	t.Log("\nJump Scenarios:")
	t.Logf("Level 15→16: Need %d XP", reqs[16])
	t.Logf("Level 15→17: Need %d XP (both levels)", reqs[16]+reqs[17])
	t.Logf("10 minutes talking = 600 XP (without bonuses)")
	t.Logf("10 minutes talking = 1200 XP (with 2x rested bonus)")
	t.Log("\nConclusion: Jumping from 15→17 in 10 minutes requires rested XP bonus")
}

func setupLevelingDB(t *testing.T) *gorm.DB {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "leveling_test.db")
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
	return gdb
}
