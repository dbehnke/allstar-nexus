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

func setupDB(t *testing.T) *gorm.DB {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "levelup_renown_test.db")
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

func TestLevelUp_CarryoverXP(t *testing.T) {
	gdb := setupDB(t)
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

	callsign := "W1CARRY"
	prof, err := profileRepo.GetByCallsign(context.Background(), callsign)
	if err != nil {
		t.Fatalf("get profile: %v", err)
	}
	// Start at Level 1 with 350 XP; need 360 to reach Level 2, expect carryover 10
	prof.Level = 1
	prof.ExperiencePoints = 350
	if err := profileRepo.Upsert(context.Background(), prof); err != nil {
		t.Fatalf("upsert profile: %v", err)
	}

	// Seed a 20s transmission within last minute
	now := time.Now().Add(-time.Minute)
	if err := txRepo.LogTransmission(100, 200, callsign, now, now.Add(20*time.Second), 20); err != nil {
		t.Fatalf("seed tx: %v", err)
	}

	cfg := &gamification.Config{CapsEnabled: false, RestedEnabled: false, DREnabled: false, KerchunkEnabled: false}
	ts := gamification.NewTallyService(gdb, txRepo, profileRepo, levelRepo, activityRepo, stateRepo, cfg, 30*time.Minute, zaptestLogger())
	if err := ts.Start(); err != nil {
		t.Fatalf("start tally: %v", err)
	}
	defer ts.Stop()

	prof2, err := profileRepo.GetByCallsign(context.Background(), callsign)
	if err != nil {
		t.Fatalf("reload profile: %v", err)
	}
	if prof2.Level != 2 {
		t.Fatalf("expected level 2, got %d", prof2.Level)
	}
	if prof2.ExperiencePoints != 10 {
		t.Fatalf("expected carryover XP 10, got %d", prof2.ExperiencePoints)
	}
}

func TestRenownTransition_Level59To60(t *testing.T) {
	gdb := setupDB(t)
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

	callsign := "W1RENOWN"
	prof, err := profileRepo.GetByCallsign(context.Background(), callsign)
	if err != nil {
		t.Fatalf("get profile: %v", err)
	}
	// Place at level 59 with XP just below requirement for level 60
	need := reqs[60]
	prof.Level = 59
	prof.ExperiencePoints = need - 5
	if err := profileRepo.Upsert(context.Background(), prof); err != nil {
		t.Fatalf("upsert profile: %v", err)
	}

	// Award exactly 5s to reach level 60 and trigger renown reset
	now := time.Now().Add(-time.Minute)
	if err := txRepo.LogTransmission(101, 201, callsign, now, now.Add(5*time.Second), 5); err != nil {
		t.Fatalf("seed tx: %v", err)
	}

	cfg := &gamification.Config{CapsEnabled: false, RestedEnabled: false, DREnabled: false, KerchunkEnabled: false}
	ts := gamification.NewTallyService(gdb, txRepo, profileRepo, levelRepo, activityRepo, stateRepo, cfg, 30*time.Minute, zaptestLogger())
	if err := ts.Start(); err != nil {
		t.Fatalf("start tally: %v", err)
	}
	defer ts.Stop()

	prof2, err := profileRepo.GetByCallsign(context.Background(), callsign)
	if err != nil {
		t.Fatalf("reload profile: %v", err)
	}
	if prof2.RenownLevel != 1 {
		t.Fatalf("expected renown 1, got %d", prof2.RenownLevel)
	}
	if prof2.Level != 1 {
		t.Fatalf("expected level reset to 1, got %d", prof2.Level)
	}
	if prof2.ExperiencePoints != 0 {
		t.Fatalf("expected XP reset to 0, got %d", prof2.ExperiencePoints)
	}
}
