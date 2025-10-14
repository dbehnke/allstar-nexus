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

func setupDBForRenown(t *testing.T) *gorm.DB {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "renown_test.db")
	gdb, err := gorm.Open(sqlite.New(sqlite.Config{DriverName: "sqlite", DSN: dbPath}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open gorm sqlite: %v", err)
	}
	if err := gdb.AutoMigrate(&models.CallsignProfile{}, &models.LevelConfig{}, &models.TransmissionLog{}, &models.XPActivityLog{}, &models.TallyState{}); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	return gdb
}

// Test that a level 60 profile will renown when awarded exactly the configured renown XP
func TestRenownFixedXPLeveling(t *testing.T) {
	gdb := setupDBForRenown(t)
	levelRepo := repository.NewLevelConfigRepo(gdb)
	profileRepo := repository.NewCallsignProfileRepo(gdb)
	txRepo := repository.NewTransmissionLogRepository(gdb)
	activityRepo := repository.NewXPActivityRepo(gdb)
	stateRepo := repository.NewTallyStateRepo(gdb)

	// Seed default level requirements
	reqs := gamification.CalculateLevelRequirements()
	if err := levelRepo.SeedDefaults(context.Background(), reqs); err != nil {
		t.Fatalf("seed level config: %v", err)
	}

	callsign := "W1RENOWNFIX"
	prof, err := profileRepo.GetByCallsign(context.Background(), callsign)
	if err != nil {
		t.Fatalf("get profile: %v", err)
	}
	// Place at level 60 with 0 XP
	prof.Level = 60
	prof.ExperiencePoints = 0
	if err := profileRepo.Upsert(context.Background(), prof); err != nil {
		t.Fatalf("upsert profile: %v", err)
	}

	// Award renown XP (configured 36000)
	now := time.Now().Add(-time.Minute)
	if err := txRepo.LogTransmission(200, 200+36000, callsign, now, now.Add(time.Second), 36000); err != nil {
		t.Fatalf("seed tx: %v", err)
	}

	cfg := &gamification.Config{CapsEnabled: false, RestedEnabled: false, DREnabled: false, KerchunkEnabled: false, RenownEnabled: true, RenownXPPerLevel: 36000}
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
		t.Fatalf("expected XP reset to 0 after renown, got %d", prof2.ExperiencePoints)
	}
}

// Test that normal leveling from level 1 up to level 2 works (sanity)
func TestSimpleLevelUp1to2(t *testing.T) {
	gdb := setupDBForRenown(t)
	levelRepo := repository.NewLevelConfigRepo(gdb)
	profileRepo := repository.NewCallsignProfileRepo(gdb)
	txRepo := repository.NewTransmissionLogRepository(gdb)
	activityRepo := repository.NewXPActivityRepo(gdb)
	stateRepo := repository.NewTallyStateRepo(gdb)

	reqs := gamification.CalculateLevelRequirements()
	if err := levelRepo.SeedDefaults(context.Background(), reqs); err != nil {
		t.Fatalf("seed level config: %v", err)
	}

	callsign := "W1SIMPLE"
	prof, err := profileRepo.GetByCallsign(context.Background(), callsign)
	if err != nil {
		t.Fatalf("get profile: %v", err)
	}
	prof.Level = 1
	prof.ExperiencePoints = 350 // need 360
	if err := profileRepo.Upsert(context.Background(), prof); err != nil {
		t.Fatalf("upsert profile: %v", err)
	}

	now := time.Now().Add(-time.Minute)
	if err := txRepo.LogTransmission(300, 320, callsign, now, now.Add(20*time.Second), 20); err != nil {
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
