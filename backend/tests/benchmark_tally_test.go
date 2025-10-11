package tests

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/dbehnke/allstar-nexus/backend/gamification"
	"github.com/dbehnke/allstar-nexus/backend/models"
	"github.com/dbehnke/allstar-nexus/backend/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func BenchmarkProcessTally_ManyLogs(b *testing.B) {
	dir := b.TempDir()
	dbPath := filepath.Join(dir, "bench_tally.db")
	gdb, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		b.Fatalf("open gorm sqlite: %v", err)
	}
	if err := gdb.AutoMigrate(
		&models.CallsignProfile{},
		&models.LevelConfig{},
		&models.TransmissionLog{},
		&models.XPActivityLog{},
	); err != nil {
		b.Fatalf("automigrate: %v", err)
	}

	levelRepo := repository.NewLevelConfigRepo(gdb)
	profileRepo := repository.NewCallsignProfileRepo(gdb)
	txRepo := repository.NewTransmissionLogRepository(gdb)
	activityRepo := repository.NewXPActivityRepo(gdb)

	// Seed defaults
	reqs := gamification.CalculateLevelRequirements()
	if err := levelRepo.SeedDefaults(context.Background(), reqs); err != nil {
		b.Fatalf("seed level config: %v", err)
	}

	// Seed N callsigns with M logs each
	N := 200
	M := 10
	now := time.Now().Add(-time.Minute)
	for i := 0; i < N; i++ {
		callsign := fmt.Sprintf("BENCH%03d", i)
		if _, err := profileRepo.GetByCallsign(context.Background(), callsign); err != nil {
			b.Fatalf("profile: %v", err)
		}
		for j := 0; j < M; j++ {
			s := now.Add(time.Duration(j) * time.Second)
			if err := txRepo.LogTransmission(100+i, 200+j, callsign, s, s.Add(3*time.Second), 3); err != nil {
				b.Fatalf("seed tx: %v", err)
			}
		}
	}

	cfg := &gamification.Config{CapsEnabled: false, RestedEnabled: false, DREnabled: false, KerchunkEnabled: false}
	ts := gamification.NewTallyService(gdb, txRepo, profileRepo, levelRepo, activityRepo, cfg, 30*time.Minute, zaptestLogger())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := ts.ProcessTally(); err != nil {
			b.Fatalf("tally: %v", err)
		}
	}
}
