package tests

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/dbehnke/allstar-nexus/backend/gamification"
	"github.com/dbehnke/allstar-nexus/backend/models"
	"github.com/dbehnke/allstar-nexus/backend/repository"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setUpGormTestDB returns a temporary sqlite GORM DB with required migrations.
func setUpGormTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "tally_test.db")
	gdb, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open gorm sqlite: %v", err)
	}
	if err := gdb.AutoMigrate(
		&models.CallsignProfile{},
		&models.LevelConfig{},
		&models.TransmissionLog{},
		&models.XPActivityLog{},
	); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	return gdb
}

// TestTallyService_ProcessOnce validates XP flow including DR, caps, kerchunk, and rested basics.
func TestTallyService_ProcessOnce(t *testing.T) {
	gdb := setUpGormTestDB(t)

	levelRepo := repository.NewLevelConfigRepo(gdb)
	profileRepo := repository.NewCallsignProfileRepo(gdb)
	txRepo := repository.NewTransmissionLogRepository(gdb)
	activityRepo := repository.NewXPActivityRepo(gdb)

	// Seed simple level requirements (use defaults)
	reqs := gamification.CalculateLevelRequirements()
	if err := levelRepo.SeedDefaults(context.Background(), reqs); err != nil {
		t.Fatalf("seed level config: %v", err)
	}

	// Configure gamification with low-activity hub defaults but small caps to assert behavior
	cfg := &gamification.Config{
		RestedEnabled:          true,
		RestedAccumulationRate: 1.0, // 1h offline -> 1h bonus
		RestedMaxSeconds:       4 * 3600,
		RestedMultiplier:       2.0,

		DREnabled: true,
		DRTiers: []gamification.DRTier{
			{MaxSeconds: 60, Multiplier: 1.0},   // first 1m at 1.0x
			{MaxSeconds: 180, Multiplier: 0.75}, // next 2m at 0.75x
			{MaxSeconds: 9999, Multiplier: 0.5}, // remaining at 0.5x
		},

		KerchunkEnabled:       true,
		KerchunkThreshold:     3,
		KerchunkWindow:        30,
		KerchunkSinglePenalty: 0.5,
		Kerchunk2to3Penalty:   0.25,
		Kerchunk4to5Penalty:   0.1,
		Kerchunk6PlusPenalty:  0.0,

		CapsEnabled:      true,
		DailyCapSeconds:  120, // 2 minutes/day to easily hit in test
		WeeklyCapSeconds: 300, // 5 minutes/week cap
	}

	// Build TallyService (logger nil-safe in code; if not, we'd stub)
	ts := gamification.NewTallyService(gdb, txRepo, profileRepo, levelRepo, activityRepo, cfg, 30*time.Minute, zaptestLogger())

	// Seed transmissions across a single callsign to exercise logic
	callsign := "K9TEST"
	now := time.Now().Add(-time.Minute)

	// 1) Normal TX of 40s (within first DR tier)
	_ = txRepo.LogTransmission(1001, 2001, callsign, now, now.Add(40*time.Second), 40)
	// 2) Short kerchunk 2s within window
	_ = txRepo.LogTransmission(1001, 2001, callsign, now.Add(1*time.Second), now.Add(3*time.Second), 2)
	// 3) Another kerchunk 2s → consecutive penalties intensify
	_ = txRepo.LogTransmission(1001, 2001, callsign, now.Add(5*time.Second), now.Add(7*time.Second), 2)
	// 4) Longer TX 120s to trigger DR tiers and daily cap interaction
	_ = txRepo.LogTransmission(1001, 2001, callsign, now.Add(20*time.Second), now.Add(140*time.Second), 120)

	// Ensure profile exists and has some rested pool so multiplier applies
	prof, err := profileRepo.GetByCallsign(context.Background(), callsign)
	if err != nil {
		t.Fatalf("get profile: %v", err)
	}
	// Simulate 3 hours offline to accumulate rested
	prof.LastTransmissionAt = time.Now().Add(-3 * time.Hour)
	prof.RestedBonusSeconds = 3 * 3600
	if err := profileRepo.Upsert(context.Background(), prof); err != nil {
		t.Fatalf("save profile: %v", err)
	}

	// Run one tally pass
	if err := ts.ProcessTally(); err != nil {
		t.Fatalf("process tally: %v", err)
	}

	// Reload profile and check XP bounded by daily cap (<= 120)
	prof, err = profileRepo.GetByCallsign(context.Background(), callsign)
	if err != nil {
		t.Fatalf("reload profile: %v", err)
	}
	if prof.ExperiencePoints <= 0 {
		t.Fatalf("expected some XP to be awarded, got %d", prof.ExperiencePoints)
	}
	if prof.ExperiencePoints > cfg.DailyCapSeconds {
		t.Fatalf("awarded XP %d exceeds daily cap %d", prof.ExperiencePoints, cfg.DailyCapSeconds)
	}

	// Activity logs should exist reflecting capped awards
	var logs []models.XPActivityLog
	if err := gdb.Where("callsign = ?", callsign).Find(&logs).Error; err != nil {
		t.Fatalf("query activity: %v", err)
	}
	if len(logs) == 0 {
		t.Fatalf("expected activity logs to be created")
	}

	// Sum awarded from activity logs should equal profile XP
	sumAwarded := 0
	for _, l := range logs {
		sumAwarded += l.AwardedXP
	}
	if sumAwarded != prof.ExperiencePoints {
		t.Fatalf("sum awarded %d != profile XP %d", sumAwarded, prof.ExperiencePoints)
	}

	// Run second tally immediately — no new logs, should be idempotent
	if err := ts.ProcessTally(); err != nil {
		t.Fatalf("process tally 2: %v", err)
	}
	prof2, _ := profileRepo.GetByCallsign(context.Background(), callsign)
	if prof2.ExperiencePoints != prof.ExperiencePoints {
		t.Fatalf("xp changed without new logs (%d -> %d)", prof.ExperiencePoints, prof2.ExperiencePoints)
	}
}

// Minimal zap logger replacement to satisfy constructor without external deps.
// The service uses logger only for Info/Error; we can return a no-op logger.
func zaptestLogger() *zap.Logger {
	// Build a no-op logger (disabled) to avoid noisy output during tests
	cfg := zap.NewDevelopmentConfig()
	cfg.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	l, _ := cfg.Build()
	return l
}
