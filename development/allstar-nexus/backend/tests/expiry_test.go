package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/dbehnke/allstar-nexus/backend/api"
	"github.com/dbehnke/allstar-nexus/backend/database"
)

// Helper to create API/server with custom TTL
func newServerWithTTL(t *testing.T, ttl time.Duration) (*httptest.Server, func()) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "ttl.db")
	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	apiLayer := api.New(db.DB, "secret", ttl)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/auth/register", apiLayer.Register)
	mux.HandleFunc("/api/auth/login", apiLayer.Login)
	mux.HandleFunc("/api/me", apiLayer.Me)
	srv := httptest.NewServer(mux)
	cleanup := func() { srv.Close(); db.CloseSafe() }
	return srv, cleanup
}

type env struct {
	OK   bool            `json:"ok"`
	Data json.RawMessage `json:"data"`
}

func TestShortTTLExpiry(t *testing.T) {
	srv, cleanup := newServerWithTTL(t, 1*time.Second)
	defer cleanup()
	client := srv.Client()
	// register
	client.Post(srv.URL+"/api/auth/register", "application/json", bytesReader(`{"email":"e@x","password":"Password!1"}`))
	// login
	resp, _ := client.Post(srv.URL+"/api/auth/login", "application/json", bytesReader(`{"email":"e@x","password":"Password!1"}`))
	var e env
	decode(resp, &e)
	var payload map[string]string
	json.Unmarshal(e.Data, &payload)
	tok := payload["token"]
	// immediate /api/me should pass
	rMe1, _ := getWithToken(client, srv.URL+"/api/me", tok)
	if rMe1.StatusCode != 200 {
		t.Fatalf("expected 200 got %d", rMe1.StatusCode)
	}
	time.Sleep(1500 * time.Millisecond)
	// after expiry
	rMe2, _ := getWithToken(client, srv.URL+"/api/me", tok)
	if rMe2.StatusCode != 401 {
		t.Fatalf("expected 401 after expiry got %d", rMe2.StatusCode)
	}
}

// local helpers
func bytesReader(s string) *bytes.Reader { return bytes.NewReader([]byte(s)) }
func decode(resp *http.Response, v any) {
	defer resp.Body.Close()
	json.NewDecoder(resp.Body).Decode(v)
}
func getWithToken(c *http.Client, url, tok string) (*http.Response, env) {
	req, _ := http.NewRequest("GET", url, nil)
	if tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	resp, _ := c.Do(req)
	var e env
	json.NewDecoder(resp.Body).Decode(&e)
	resp.Body.Close()
	return resp, e
}
