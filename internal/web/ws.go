package web

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/dbehnke/allstar-nexus/internal/core"
)

// messageEnvelope defines WS protocol envelope.
type messageEnvelope struct {
	MessageType string      `json:"messageType"`
	Data        interface{} `json:"data,omitempty"`
	Timestamp   int64       `json:"timestamp"`
}

// Hub manages websocket clients and broadcasts.
type Hub struct {
	mu      sync.RWMutex
	clients map[*websocket.Conn]struct{}
}

func NewHub() *Hub { return &Hub{clients: map[*websocket.Conn]struct{}{}} }

// HandleWS upgrades and registers a client.
func (h *Hub) HandleWS(sm *core.StateManager, authValidator func(r *http.Request) bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// If this is not a WebSocket upgrade request, return a helpful status.
		if r.Header.Get("Connection") == "" || r.Header.Get("Upgrade") == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUpgradeRequired)
			w.Write([]byte(`{"ok":false,"error":"websocket_upgrade_required"}`))
			return
		}
		if authValidator != nil && !authValidator(r) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			// write minimal error if not already written
			http.Error(w, "websocket_accept_failed", http.StatusInternalServerError)
			return
		}
		h.mu.Lock()
		h.clients[c] = struct{}{}
		h.mu.Unlock()
		go func() {
			defer func() { h.mu.Lock(); delete(h.clients, c); h.mu.Unlock(); c.Close(websocket.StatusNormalClosure, "") }()
			for { // discard inbound
				if _, _, err := c.Read(context.Background()); err != nil {
					return
				}
			}
		}()
		// Immediately send current snapshot
		snap := sm.Snapshot()
		env := messageEnvelope{MessageType: "STATUS_UPDATE", Data: snap, Timestamp: time.Now().UnixMilli()}
		b, _ := json.Marshal(env)
		c.Write(context.Background(), websocket.MessageText, b)

		// Send initial talker log snapshot
		talkerLog := sm.TalkerLogSnapshot()
		talkerEnv := messageEnvelope{MessageType: "TALKER_LOG_SNAPSHOT", Data: talkerLog, Timestamp: time.Now().UnixMilli()}
		talkerB, _ := json.Marshal(talkerEnv)
		c.Write(context.Background(), websocket.MessageText, talkerB)
	}
}

// BroadcastLoop listens for state updates and fans out.
func (h *Hub) BroadcastLoop(updates <-chan core.NodeState) {
	for st := range updates {
		env := messageEnvelope{MessageType: "STATUS_UPDATE", Data: st, Timestamp: time.Now().UnixMilli()}
		payload, _ := json.Marshal(env)
		h.mu.RLock()
		for c := range h.clients {
			go func(conn *websocket.Conn, p []byte) { conn.Write(context.Background(), websocket.MessageText, p) }(c, payload)
		}
		h.mu.RUnlock()
	}
}

// TalkerLoop broadcasts talker events.
func (h *Hub) TalkerLoop(events <-chan core.TalkerEvent) {
	for evt := range events {
		env := messageEnvelope{MessageType: "TALKER_EVENT", Data: evt, Timestamp: time.Now().UnixMilli()}
		payload, _ := json.Marshal(env)
		h.mu.RLock()
		for c := range h.clients {
			go func(conn *websocket.Conn, p []byte) { conn.Write(context.Background(), websocket.MessageText, p) }(c, payload)
		}
		h.mu.RUnlock()
	}
}

// LinkUpdateLoop broadcasts incremental link additions.
func (h *Hub) LinkUpdateLoop(updates <-chan []core.LinkInfo) {
	for added := range updates {
		env := messageEnvelope{MessageType: "LINK_ADDED", Data: added, Timestamp: time.Now().UnixMilli()}
		payload, _ := json.Marshal(env)
		h.mu.RLock()
		for c := range h.clients {
			go func(conn *websocket.Conn, p []byte) { conn.Write(context.Background(), websocket.MessageText, p) }(c, payload)
		}
		h.mu.RUnlock()
	}
}

// LinkRemovalLoop broadcasts link removals.
func (h *Hub) LinkRemovalLoop(removals <-chan []int) {
	for rem := range removals {
		env := messageEnvelope{MessageType: "LINK_REMOVED", Data: rem, Timestamp: time.Now().UnixMilli()}
		payload, _ := json.Marshal(env)
		h.mu.RLock()
		for c := range h.clients {
			go func(conn *websocket.Conn, p []byte) { conn.Write(context.Background(), websocket.MessageText, p) }(c, payload)
		}
		h.mu.RUnlock()
	}
}

// LinkTxLoop broadcasts per-link TX start/stop events.
func (h *Hub) LinkTxLoop(events <-chan core.LinkTxEvent) {
	for evt := range events {
		env := messageEnvelope{MessageType: "LINK_TX", Data: evt, Timestamp: time.Now().UnixMilli()}
		payload, _ := json.Marshal(env)
		h.mu.RLock()
		for c := range h.clients {
			go func(conn *websocket.Conn, p []byte) { conn.Write(context.Background(), websocket.MessageText, p) }(c, payload)
		}
		h.mu.RUnlock()
	}
}

// LinkTxBatchLoop consumes the same events channel and batches edges within a window (e.g., 100ms) into LINK_TX_BATCH messages.
func (h *Hub) LinkTxBatchLoop(events <-chan core.LinkTxEvent, window time.Duration) {
	if window <= 0 {
		window = 100 * time.Millisecond
	}
	buf := make([]core.LinkTxEvent, 0, 16)
	timer := time.NewTimer(window)
	flush := func() {
		if len(buf) == 0 {
			return
		}
		env := messageEnvelope{MessageType: "LINK_TX_BATCH", Data: buf, Timestamp: time.Now().UnixMilli()}
		payload, _ := json.Marshal(env)
		h.mu.RLock()
		for c := range h.clients {
			go func(conn *websocket.Conn, p []byte) { conn.Write(context.Background(), websocket.MessageText, p) }(c, payload)
		}
		h.mu.RUnlock()
		buf = buf[:0]
	}
	for {
		select {
		case evt, ok := <-events:
			if !ok {
				flush()
				return
			}
			buf = append(buf, evt)
			if len(buf) == 1 {
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(window)
			}
		case <-timer.C:
			flush()
			timer.Reset(window)
		}
	}
}

// HeartbeatLoop periodically emits a STATUS_UPDATE snapshot even if no new events have arrived,
// ensuring newly connected clients or quiet systems still receive state.
func (h *Hub) HeartbeatLoop(sm *core.StateManager, interval time.Duration) {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		snap := sm.Snapshot()
		env := messageEnvelope{MessageType: "STATUS_UPDATE", Data: snap, Timestamp: time.Now().UnixMilli()}
		payload, _ := json.Marshal(env)
		h.mu.RLock()
		for c := range h.clients {
			go func(conn *websocket.Conn, p []byte) { conn.Write(context.Background(), websocket.MessageText, p) }(c, payload)
		}
		h.mu.RUnlock()
	}
}

// TalkerLogRefreshLoop periodically sends full talker log snapshot (every 2 minutes)
// This ensures clients stay in sync even if they miss incremental TALKER_EVENT messages
func (h *Hub) TalkerLogRefreshLoop(sm *core.StateManager, interval time.Duration) {
	if interval <= 0 {
		interval = 2 * time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		talkerLog := sm.TalkerLogSnapshot()
		env := messageEnvelope{MessageType: "TALKER_LOG_SNAPSHOT", Data: talkerLog, Timestamp: time.Now().UnixMilli()}
		payload, _ := json.Marshal(env)
		h.mu.RLock()
		for c := range h.clients {
			go func(conn *websocket.Conn, p []byte) { conn.Write(context.Background(), websocket.MessageText, p) }(c, payload)
		}
		h.mu.RUnlock()
	}
}
