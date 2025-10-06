package repository

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/dbehnke/allstar-nexus/backend/database"
	"github.com/dbehnke/allstar-nexus/backend/models"
)

func setupTestDB(t *testing.T) *database.DB {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "repo.db")
	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestUserRepoCRUDAndCounts(t *testing.T) {
	db := setupTestDB(t)
	defer db.CloseSafe()
	repo := NewUserRepo(db.DB)
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
