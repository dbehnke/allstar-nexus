package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dbehnke/allstar-nexus/backend/api"
	"github.com/dbehnke/allstar-nexus/backend/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type envelope struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// buildMux centralizes route registration to keep tests aligned with main server.
func buildMux(apiLayer *api.API) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/auth/register", apiLayer.Register)
	mux.HandleFunc("/api/auth/login", apiLayer.Login)
	mux.HandleFunc("/api/me", apiLayer.Me)
	mux.HandleFunc("/api/admin/summary", apiLayer.AdminSummary)
	mux.HandleFunc("/api/dashboard/summary", apiLayer.DashboardSummary)
	mux.HandleFunc("/api/link-stats", apiLayer.LinkStatsHandler) // auth not applied in this isolated test mux
	mux.HandleFunc("/api/link-stats/top", apiLayer.TopLinkStatsHandler)
	return mux
}

func newTestServer(t *testing.T) (*httptest.Server, func()) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	gdb, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open gorm sqlite: %v", err)
	}
	if err := gdb.AutoMigrate(&models.User{}, &models.LinkStat{}); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	apiLayer := api.New(gdb, "test-secret", 2*time.Hour)
	mux := buildMux(apiLayer)
	srv := httptest.NewServer(mux)
	cleanup := func() {
		srv.Close()
		// nothing to close for gorm/sqlite file; remove temp dir
		os.RemoveAll(dir)
	}
	return srv, cleanup
}

func postJSON(t *testing.T, client *http.Client, url string, body any) (*http.Response, envelope) {
	t.Helper()
	b, _ := json.Marshal(body)
	resp, err := client.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	var env envelope
	data, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	_ = json.Unmarshal(data, &env)
	return resp, env
}

func getAuth(t *testing.T, client *http.Client, url, token string) (*http.Response, envelope) {
	t.Helper()
	req, _ := http.NewRequest("GET", url, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	var env envelope
	data, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	_ = json.Unmarshal(data, &env)
	return resp, env
}

func TestHandlersBootstrapAndRoles(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()
	client := srv.Client()

	// First user becomes superadmin (email normalization check with uppercase)
	r1, env1 := postJSON(t, client, srv.URL+"/api/auth/register", map[string]string{"email": "Admin@Example.COM", "password": "Password!1"})
	if r1.StatusCode != 201 || !env1.OK {
		t.Fatalf("expected 201 ok got %d env=%+v", r1.StatusCode, env1)
	}
	type user struct {
		ID    int64  `json:"id"`
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	var u1 user
	json.Unmarshal(env1.Data, &u1)
	if u1.Role != "superadmin" {
		t.Fatalf("first user role = %s want superadmin", u1.Role)
	}
	if u1.Email != "admin@example.com" {
		t.Fatalf("email normalization failed: %s", u1.Email)
	}

	// Second user requests admin
	r2, env2 := postJSON(t, client, srv.URL+"/api/auth/register", map[string]string{"email": "second@example.com", "password": "Password!1", "role": "admin"})
	if r2.StatusCode != 201 {
		t.Fatalf("expected 201 second user got %d", r2.StatusCode)
	}
	var u2 user
	json.Unmarshal(env2.Data, &u2)
	if u2.Role != "admin" {
		t.Fatalf("second user should be admin got %s", u2.Role)
	}

	// Third user defaults to user even if requesting admin
	r3, env3 := postJSON(t, client, srv.URL+"/api/auth/register", map[string]string{"email": "third@example.com", "password": "Password!1", "role": "admin"})
	if r3.StatusCode != 201 {
		t.Fatalf("expected 201 third user got %d", r3.StatusCode)
	}
	var u3 user
	json.Unmarshal(env3.Data, &u3)
	if u3.Role != "user" {
		t.Fatalf("third user expected role user got %s", u3.Role)
	}

	// Login superadmin
	_, login1 := postJSON(t, client, srv.URL+"/api/auth/login", map[string]string{"email": "admin@example.com", "password": "Password!1"})
	if !login1.OK {
		t.Fatalf("login superadmin failed: %+v", login1)
	}
	var loginData1 map[string]string
	json.Unmarshal(login1.Data, &loginData1)
	tokenSuper := loginData1["token"]

	// Login normal user
	_, login3 := postJSON(t, client, srv.URL+"/api/auth/login", map[string]string{"email": "third@example.com", "password": "Password!1"})
	if !login3.OK {
		t.Fatalf("login third user failed")
	}
	var loginData3 map[string]string
	json.Unmarshal(login3.Data, &loginData3)
	tokenUser := loginData3["token"]

	// /api/me with superadmin token
	rm, meEnv := getAuth(t, client, srv.URL+"/api/me", tokenSuper)
	if rm.StatusCode != 200 || !meEnv.OK {
		t.Fatalf("/api/me failed %d env=%+v", rm.StatusCode, meEnv)
	}

	// /api/admin/summary unauthorized with user token
	raUser, envAdminUser := getAuth(t, client, srv.URL+"/api/admin/summary", tokenUser)
	if raUser.StatusCode != 403 || envAdminUser.OK {
		t.Fatalf("expected 403 for normal user got %d", raUser.StatusCode)
	}

	// /api/admin/summary with superadmin token
	raAdmin, envAdmin := getAuth(t, client, srv.URL+"/api/admin/summary", tokenSuper)
	if raAdmin.StatusCode != 200 || !envAdmin.OK {
		t.Fatalf("admin summary access failed %d", raAdmin.StatusCode)
	}
}

func TestDuplicateEmailAndDashboard(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()
	client := srv.Client()
	// first register
	r1, _ := postJSON(t, client, srv.URL+"/api/auth/register", map[string]string{"email": "dup@example.com", "password": "Password!1"})
	if r1.StatusCode != 201 {
		t.Fatalf("expected 201 first register got %d", r1.StatusCode)
	}
	// duplicate register
	r2, env2 := postJSON(t, client, srv.URL+"/api/auth/register", map[string]string{"email": "dup@example.com", "password": "Password!1"})
	if r2.StatusCode != 409 {
		t.Fatalf("expected 409 duplicate got %d", r2.StatusCode)
	}
	if env2.Error == nil || env2.Error.Code != "duplicate_email" {
		t.Fatalf("expected duplicate_email code env=%+v", env2)
	}
	// public dashboard summary
	rd, _ := getAuth(t, client, srv.URL+"/api/admin/summary", "")
	if rd.StatusCode != 401 {
		t.Fatalf("expected 401 for admin summary without token got %d", rd.StatusCode)
	}
	// call dashboard public endpoint (not behind auth) and verify enrichment fields
	reqDash, _ := http.NewRequest("GET", srv.URL+"/api/dashboard/summary", nil)
	respDash, err := client.Do(reqDash)
	if err != nil {
		t.Fatalf("dashboard request error: %v", err)
	}
	if respDash.StatusCode != 200 {
		t.Fatalf("expected 200 dashboard got %d", respDash.StatusCode)
	}
	var dashEnv envelope
	bDash, _ := io.ReadAll(respDash.Body)
	respDash.Body.Close()
	json.Unmarshal(bDash, &dashEnv)
	if !dashEnv.OK {
		t.Fatalf("dashboard not ok env=%+v", dashEnv)
	}
	var dashData map[string]any
	json.Unmarshal(dashEnv.Data, &dashData)
	if _, ok := dashData["total_users"]; !ok {
		t.Fatalf("missing total_users")
	}
	if _, ok := dashData["new_last_24h"]; !ok {
		t.Fatalf("missing new_last_24h")
	}
}

func TestTokenExpiry(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()
	client := srv.Client()
	// Create and login user
	postJSON(t, client, srv.URL+"/api/auth/register", map[string]string{"email": "x@example.com", "password": "Password!1"})
	_, login := postJSON(t, client, srv.URL+"/api/auth/login", map[string]string{"email": "x@example.com", "password": "Password!1"})
	var loginData map[string]string
	json.Unmarshal(login.Data, &loginData)
	token := loginData["token"]
	// Simulate expiry by parsing token then waiting past 1s if we reissued with shorter TTL (not available here). Instead directly check /api/me works now.
	rm, _ := getAuth(t, client, srv.URL+"/api/me", token)
	if rm.StatusCode != 200 {
		t.Fatalf("expected 200 /api/me got %d", rm.StatusCode)
	}
	// NOTE: For a real expiry test we'd need a shorter TTL injection point.
	_ = context.Background()
}
