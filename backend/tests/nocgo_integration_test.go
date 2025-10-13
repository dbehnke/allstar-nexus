package tests

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/dbehnke/allstar-nexus/backend/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	_ "modernc.org/sqlite"
)

// TestNoCGOIntegration verifies that the entire stack works without CGO.
// This test should pass when compiled with CGO_ENABLED=0.
func TestNoCGOIntegration(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "nocgo_test.db")

	// Open database using modernc.org/sqlite (pure Go)
	gormDB, err := gorm.Open(sqlite.New(sqlite.Config{
		DriverName: "sqlite",
		DSN:        dbPath,
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	// Set PRAGMA settings
	sqlDB, err := gormDB.DB()
	if err != nil {
		t.Fatalf("failed to get sql.DB: %v", err)
	}

	if _, err := sqlDB.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		t.Fatalf("failed to set journal_mode: %v", err)
	}

	if _, err := sqlDB.Exec("PRAGMA synchronous=NORMAL;"); err != nil {
		t.Fatalf("failed to set synchronous: %v", err)
	}

	// Auto-migrate all models
	if err := gormDB.AutoMigrate(
		&models.User{},
		&models.TransmissionLog{},
		&models.NodeInfo{},
		&models.LinkStat{},
		&models.CallsignProfile{},
		&models.LevelConfig{},
		&models.XPActivityLog{},
		&models.TallyState{},
	); err != nil {
		t.Fatalf("auto-migrate failed: %v", err)
	}

	// Test user operations
	ctx := context.Background()
	user := &models.User{
		Email:        "test@example.com",
		PasswordHash: "hash123",
		Role:         models.RoleUser,
		CreatedAt:    time.Now(),
	}

	if err := gormDB.WithContext(ctx).Create(user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	var retrievedUser models.User
	if err := gormDB.WithContext(ctx).Where("email = ?", "test@example.com").First(&retrievedUser).Error; err != nil {
		t.Fatalf("failed to retrieve user: %v", err)
	}

	if retrievedUser.Email != user.Email {
		t.Fatalf("email mismatch: expected %s, got %s", user.Email, retrievedUser.Email)
	}

	// Test link stats operations
	now := time.Now()
	linkStat := &models.LinkStat{
		Node:           12345,
		TotalTxSeconds: 120,
		LastTxStart:    &now,
		LastTxEnd:      &now,
		ConnectedSince: &now,
	}

	if err := gormDB.WithContext(ctx).Create(linkStat).Error; err != nil {
		t.Fatalf("failed to create link stat: %v", err)
	}

	var retrievedLinkStat models.LinkStat
	if err := gormDB.WithContext(ctx).Where("node = ?", 12345).First(&retrievedLinkStat).Error; err != nil {
		t.Fatalf("failed to retrieve link stat: %v", err)
	}

	if retrievedLinkStat.Node != linkStat.Node {
		t.Fatalf("node mismatch: expected %d, got %d", linkStat.Node, retrievedLinkStat.Node)
	}

	// Test gamification profile
	profile := &models.CallsignProfile{
		Callsign:         "N0CALL",
		Level:            5,
		ExperiencePoints: 1000,
		RenownLevel:      0,
		CreatedAt:        time.Now(),
	}

	if err := gormDB.WithContext(ctx).Create(profile).Error; err != nil {
		t.Fatalf("failed to create callsign profile: %v", err)
	}

	var retrievedProfile models.CallsignProfile
	if err := gormDB.WithContext(ctx).Where("callsign = ?", "N0CALL").First(&retrievedProfile).Error; err != nil {
		t.Fatalf("failed to retrieve callsign profile: %v", err)
	}

	if retrievedProfile.Callsign != profile.Callsign {
		t.Fatalf("callsign mismatch: expected %s, got %s", profile.Callsign, retrievedProfile.Callsign)
	}

	// Verify SQLite version (should be modernc.org/sqlite)
	var version string
	if err := gormDB.Raw("SELECT sqlite_version()").Scan(&version).Error; err != nil {
		t.Fatalf("failed to get SQLite version: %v", err)
	}

	t.Logf("Successfully tested with SQLite version %s (pure Go, no CGO)", version)

	// Close database
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("failed to close database: %v", err)
	}
}
