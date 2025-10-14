package tests

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dbehnke/allstar-nexus/internal/core"
	"github.com/dbehnke/allstar-nexus/internal/web"
	gws "github.com/gorilla/websocket"
)

// This test performs a full websocket round-trip using an in-process HTTP server
// and a real websocket client (gorilla/websocket). It verifies that calling
// Hub.BroadcastTallyCompleted with a payload that includes a scoreboard results
// in a GAMIFICATION_TALLY_COMPLETED message delivered to connected clients.
func TestWebsocketTallyBroadcastIntegration(t *testing.T) {
	hub := web.NewHub()
	sm := core.NewStateManager()

	// Seed a LinksDetailed entry with an IP so we can assert masking behavior
	sm.SeedLinkStats([]core.LinkInfo{{Node: 1001, IP: "192.0.2.123"}})

	mux := http.NewServeMux()
	// non-admin endpoint
	mux.HandleFunc("/ws", hub.HandleWS(sm, func(r *http.Request) (bool, bool) { return true, false }))
	// admin endpoint
	mux.HandleFunc("/ws-admin", hub.HandleWS(sm, func(r *http.Request) (bool, bool) { return true, true }))

	ts := httptest.NewServer(mux)
	defer ts.Close()

	// Build ws URLs from test server URL
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"
	wsAdminURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws-admin"

	dialer := gws.Dialer{}
	// connect non-admin
	conn, resp, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("websocket dial failed: %v (resp=%v)", err, resp)
	}
	defer func() { _ = conn.Close() }()

	// connect admin
	adminConn, resp2, err := dialer.Dial(wsAdminURL, nil)
	if err != nil {
		t.Fatalf("websocket dial to admin failed: %v (resp=%v)", err, resp2)
	}
	defer func() { _ = adminConn.Close() }()

	// Read initial STATUS_UPDATE from both clients
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, nonAdminMsg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read initial message for non-admin: %v", err)
	}
	_ = adminConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, adminMsg, err := adminConn.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read initial message for admin: %v", err)
	}

	// Verify masking for non-admin: the IP should be masked
	var nonEnv map[string]any
	if err := json.Unmarshal(nonAdminMsg, &nonEnv); err != nil {
		t.Fatalf("failed to unmarshal non-admin env: %v", err)
	}
	if mt, _ := nonEnv["messageType"].(string); mt != "STATUS_UPDATE" {
		t.Fatalf("expected STATUS_UPDATE as first non-admin message, got %v", mt)
	}
	data, _ := nonEnv["data"].(map[string]any)
	links, _ := data["links_detailed"].([]any)
	if len(links) == 0 {
		t.Fatalf("expected links_detailed in non-admin snapshot")
	}
	first, _ := links[0].(map[string]any)
	ip, _ := first["ip"].(string)
	if !strings.HasSuffix(ip, ".*.*") {
		t.Fatalf("expected masked IP for non-admin, got %s", ip)
	}

	// Verify admin receives unmasked IP
	var adminEnv map[string]any
	if err := json.Unmarshal(adminMsg, &adminEnv); err != nil {
		t.Fatalf("failed to unmarshal admin env: %v", err)
	}
	if mt, _ := adminEnv["messageType"].(string); mt != "STATUS_UPDATE" {
		t.Fatalf("expected STATUS_UPDATE as first admin message, got %v", mt)
	}
	adata, _ := adminEnv["data"].(map[string]any)
	alinks, _ := adata["links_detailed"].([]any)
	if len(alinks) == 0 {
		t.Fatalf("expected links_detailed in admin snapshot")
	}
	afirst, _ := alinks[0].(map[string]any)
	aip, _ := afirst["ip"].(string)
	if strings.HasSuffix(aip, ".*.*") {
		t.Fatalf("expected unmasked IP for admin, got %s", aip)
	}

	// Prepare broadcast payload including scoreboard
	payload := map[string]any{
		"summary":    map[string]any{"rows_processed": 1},
		"scoreboard": []map[string]any{{"callsign": "ABC", "experience_points": 10}},
	}

	// Broadcast to connected clients
	hub.BroadcastTallyCompleted(payload)

	// Read messages until we find GAMIFICATION_TALLY_COMPLETED or timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	found := false
	for {
		select {
		case <-ctx.Done():
			if !found {
				t.Fatalf("did not receive GAMIFICATION_TALLY_COMPLETED message before timeout")
			}
			return
		default:
			_ = conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
			_, data, err := conn.ReadMessage()
			if err != nil {
				// continue loop until timeout
				continue
			}
			var env map[string]any
			if err := json.Unmarshal(data, &env); err != nil {
				t.Fatalf("failed to unmarshal ws envelope: %v", err)
			}
			mt, _ := env["messageType"].(string)
			if mt == "GAMIFICATION_TALLY_COMPLETED" {
				// verify scoreboard exists
				d, ok := env["data"].(map[string]any)
				if !ok {
					t.Fatalf("unexpected data shape in envelope: %v", env["data"])
				}
				if _, ok := d["scoreboard"]; !ok {
					t.Fatalf("expected scoreboard in payload, got: %v", d)
				}
				return
			}
		}
	}
}
