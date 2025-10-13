package tests

import (
	"encoding/json"
	"testing"
)

// Tests for negative / unauthorized auth scenarios to expand coverage.
func TestAuthNegativeCases(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()
	client := srv.Client()

	// Missing token
	r1, _ := getAuth(t, client, srv.URL+"/api/me", "")
	if r1.StatusCode != 401 {
		t.Fatalf("expected 401 missing token got %d", r1.StatusCode)
	}

	// Register + login to get valid token
	postJSON(t, client, srv.URL+"/api/auth/register", map[string]string{"email": "neg@example.com", "password": "Password!1"})
	_, loginEnv := postJSON(t, client, srv.URL+"/api/auth/login", map[string]string{"email": "neg@example.com", "password": "Password!1"})
	var loginData map[string]string
	_ = json.Unmarshal(loginEnv.Data, &loginData)
	tok := loginData["token"]

	// Malformed token (truncate)
	bad := tok[:len(tok)/2]
	r2, _ := getAuth(t, client, srv.URL+"/api/me", bad)
	if r2.StatusCode != 401 {
		t.Fatalf("expected 401 malformed token got %d", r2.StatusCode)
	}

	// Corrupt signature (flip last char)
	corrupt := tok
	if len(corrupt) > 0 {
		if corrupt[len(corrupt)-1] == 'a' {
			corrupt = corrupt[:len(corrupt)-1] + "b"
		} else {
			corrupt = corrupt[:len(corrupt)-1] + "a"
		}
	}
	r3, _ := getAuth(t, client, srv.URL+"/api/me", corrupt)
	if r3.StatusCode != 401 {
		t.Fatalf("expected 401 bad signature got %d", r3.StatusCode)
	}

	// Invalid format token with fewer segments
	r4, _ := getAuth(t, client, srv.URL+"/api/me", "abc|def")
	if r4.StatusCode != 401 {
		t.Fatalf("expected 401 invalid format got %d", r4.StatusCode)
	}
}
