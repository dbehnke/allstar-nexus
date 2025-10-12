package database

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestOpenAndClose(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	if db == nil {
		t.Fatal("db is nil")
	}

	// Verify we can perform a basic query
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var version string
	err = db.WithContext(ctx).Raw("SELECT sqlite_version()").Scan(&version).Error
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}

	if version == "" {
		t.Fatal("sqlite version is empty")
	}

	t.Logf("SQLite version: %s", version)

	// Close the database
	if err := db.CloseSafe(); err != nil {
		t.Fatalf("CloseSafe failed: %v", err)
	}
}

func TestCountUsers(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.CloseSafe()

	// Create users table using a named type
	type User struct {
		ID           int64  `gorm:"primaryKey;autoIncrement"`
		Email        string `gorm:"unique;not null"`
		PasswordHash string `gorm:"not null"`
		Role         string `gorm:"not null;default:user"`
		CreatedAt    time.Time
	}

	err = db.AutoMigrate(&User{})
	if err != nil {
		t.Fatalf("AutoMigrate failed: %v", err)
	}

	// Count should be 0
	ctx := context.Background()
	count, err := db.CountUsers(ctx)
	if err != nil {
		t.Fatalf("CountUsers failed: %v", err)
	}

	if count != 0 {
		t.Fatalf("expected count 0, got %d", count)
	}

	// Add a user
	err = db.Exec("INSERT INTO users (email, password_hash, role, created_at) VALUES (?, ?, ?, ?)",
		"test@example.com", "hash123", "user", time.Now()).Error
	if err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	// Count should be 1
	count, err = db.CountUsers(ctx)
	if err != nil {
		t.Fatalf("CountUsers failed: %v", err)
	}

	if count != 1 {
		t.Fatalf("expected count 1, got %d", count)
	}
}
