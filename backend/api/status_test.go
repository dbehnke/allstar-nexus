package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dbehnke/allstar-nexus/internal/core"
)

// fakeStateManager implements StateManagerInterface for testing
type fakeStateManager struct {
	snap core.NodeState
}

func (f *fakeStateManager) TalkerLogSnapshot() any   { return nil }
func (f *fakeStateManager) Snapshot() core.NodeState { return f.snap }

func TestStatus_IncludesNodeCallsignForHashedNodes(t *testing.T) {
	// Prepare a snapshot with a negative hashed node and callsign filled in
	li := core.LinkInfo{Node: -209395397, NodeCallsign: "KF8S", NodeDescription: "VOIP Client"}
	st := core.NodeState{LinksDetailed: []core.LinkInfo{li}}

	api := &API{StateManager: &fakeStateManager{snap: st}}

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	w := httptest.NewRecorder()

	api.Status(w, req)

	res := w.Result()
	defer func() {
		if err := res.Body.Close(); err != nil {
			t.Logf("Failed to close response body: %v", err)
		}
	}()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", res.StatusCode)
	}

	var env struct {
		OK   bool `json:"ok"`
		Data struct {
			State struct {
				LinksDetailed []map[string]interface{} `json:"links_detailed"`
			} `json:"state"`
		} `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&env); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(env.Data.State.LinksDetailed) == 0 {
		t.Fatalf("expected links_detailed in response")
	}
	// Find the negative node entry and ensure node_callsign exists
	found := false
	for _, entry := range env.Data.State.LinksDetailed {
		if n, ok := entry["node"].(float64); ok && int(n) < 0 {
			found = true
			if cs, ok := entry["node_callsign"].(string); !ok || cs == "" {
				t.Fatalf("expected node_callsign for negative node, got %#v", entry["node_callsign"])
			}
		}
	}
	if !found {
		t.Fatalf("did not find negative node in links_detailed")
	}
}
