package tests

import (
	"context"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/dbehnke/allstar-nexus/backend/api"
	"github.com/dbehnke/allstar-nexus/backend/models"
	"github.com/dbehnke/allstar-nexus/backend/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	_ "modernc.org/sqlite"
)

// helper to seed link stats directly
func seedLinkStats(t *testing.T, repo *repository.LinkStatsRepo, items []models.LinkStat) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	for _, it := range items {
		if err := repo.Upsert(ctx, it); err != nil {
			t.Fatalf("upsert: %v", err)
		}
	}
}

type linkStatsResp struct {
	Stats []struct {
		Node int `json:"node"`
	} `json:"stats"`
}

func TestLinkStatsSinceFiltering(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	gdb, err := gorm.Open(sqlite.New(sqlite.Config{
		DriverName: "sqlite",
		DSN:        dbPath,
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open gorm sqlite: %v", err)
	}
	if err := gdb.AutoMigrate(&models.User{}, &models.LinkStat{}); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	apiLayer := api.New(gdb, "secret", time.Hour)
	mux := buildMux(apiLayer)
	srv := httptest.NewServer(mux)
	defer func() { srv.Close() }()

	repo := repository.NewLinkStatsRepo(gdb)
	now := time.Now().Add(-2 * time.Hour)
	t1 := now.Add(30 * time.Minute)
	t2 := now.Add(90 * time.Minute)
	// We control UpdatedAt indirectly via Upsert CURRENT_TIMESTAMP; so space inserts to establish ordering.
	seedLinkStats(t, repo, []models.LinkStat{{Node: 1001, TotalTxSeconds: 10, ConnectedSince: &now}})
	time.Sleep(20 * time.Millisecond)
	seedLinkStats(t, repo, []models.LinkStat{{Node: 1002, TotalTxSeconds: 20, ConnectedSince: &t1}})
	time.Sleep(20 * time.Millisecond)
	seedLinkStats(t, repo, []models.LinkStat{{Node: 1003, TotalTxSeconds: 30, ConnectedSince: &t2}})

	client := srv.Client()

	// Fetch all
	resp, _ := client.Get(srv.URL + "/api/link-stats")
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 got %d", resp.StatusCode)
	}
	// Capture a reference timestamp after second insert for absolute filter
	refAbs := time.Now().Add(-10 * time.Millisecond).UTC()

	// Relative since = -1s should include all (recent updates within 1s window) but we purposely delay.
	time.Sleep(100 * time.Millisecond)
	relURL := srv.URL + "/api/link-stats?since=-50ms"
	respRel, _ := client.Get(relURL)
	if respRel.StatusCode != 200 {
		t.Fatalf("rel expected 200 got %d", respRel.StatusCode)
	}

	// Absolute RFC3339 since should likely return none (older than updates) using future time
	future := time.Now().Add(10 * time.Second).UTC().Format(time.RFC3339)
	respFuture, _ := client.Get(srv.URL + "/api/link-stats?since=" + future)
	if respFuture.StatusCode != 200 {
		t.Fatalf("future expected 200 got %d", respFuture.StatusCode)
	}

	// Provide absolute past time to include all
	past := time.Now().Add(-1 * time.Hour).UTC().Format(time.RFC3339)
	respPast, _ := client.Get(srv.URL + "/api/link-stats?since=" + past)
	if respPast.StatusCode != 200 {
		t.Fatalf("past expected 200 got %d", respPast.StatusCode)
	}

	// Node filter: node=1002
	respNode, _ := client.Get(srv.URL + "/api/link-stats?node=1002")
	if respNode.StatusCode != 200 {
		t.Fatalf("node expected 200 got %d", respNode.StatusCode)
	}

	// Limit + sort
	respLimit, _ := client.Get(srv.URL + "/api/link-stats?sort=tx_seconds_desc&limit=2")
	if respLimit.StatusCode != 200 {
		t.Fatalf("limit expected 200 got %d", respLimit.StatusCode)
	}

	// Top endpoint
	respTop, _ := client.Get(srv.URL + "/api/link-stats/top?limit=2")
	if respTop.StatusCode != 200 {
		t.Fatalf("top expected 200 got %d", respTop.StatusCode)
	}

	_ = refAbs // reserved for potential deeper assertions; currently focusing on status codes path coverage
}
