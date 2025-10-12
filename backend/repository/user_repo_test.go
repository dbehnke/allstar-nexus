package repository

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/dbehnke/allstar-nexus/backend/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "repo.db")
	gdb, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open gorm sqlite: %v", err)
	}
	if err := gdb.AutoMigrate(&models.User{}); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	return gdb
}

func TestUserRepoCRUDAndCounts(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepo(db)
	ctx := context.Background()
	// empty counts
	c, err := repo.Count(ctx)
	if err != nil || c != 0 {
		t.Fatalf("expected 0 count got %d err=%v", c, err)
	}
	// create users
	u1, err := repo.Create(ctx, "a@example.com", "hash1", models.RoleSuperAdmin)
	if err != nil || u1.ID == 0 {
		t.Fatalf("create u1 err=%v", err)
	}
	u2, err := repo.Create(ctx, "b@example.com", "hash2", models.RoleAdmin)
	if err != nil || u2.ID == 0 {
		t.Fatalf("create u2 err=%v", err)
	}
	u3, err := repo.Create(ctx, "c@example.com", "hash3", models.RoleUser)
	if err != nil || u3.ID == 0 {
		t.Fatalf("create u3 err=%v", err)
	}
	// count
	c2, _ := repo.Count(ctx)
	if c2 != 3 {
		t.Fatalf("expected 3 got %d", c2)
	}
	// get
	g2, err := repo.GetByEmail(ctx, "b@example.com")
	if err != nil || g2 == nil || g2.Email != "b@example.com" {
		t.Fatalf("get mismatch %+v err=%v", g2, err)
	}
	// role counts
	rc, err := repo.RoleCounts(ctx)
	if err != nil {
		t.Fatalf("rolecounts err=%v", err)
	}
	if rc[models.RoleSuperAdmin] != 1 || rc[models.RoleAdmin] != 1 || rc[models.RoleUser] != 1 {
		t.Fatalf("unexpected role counts: %+v", rc)
	}
	// get missing
	m, err := repo.GetByEmail(ctx, "missing@example.com")
	if err != nil || m != nil {
		t.Fatalf("expected nil missing got=%+v err=%v", m, err)
	}
	// timestamps sanity
	if time.Since(u1.CreatedAt) > time.Minute {
		t.Fatalf("unexpected created at time")
	}
}
