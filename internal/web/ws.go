package web

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
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
	clients map[*websocket.Conn]clientInfo
	// Optional hook to trigger a server-side poll when new clients connect.
	// This will be set by main when a PollingService is in-use.
	triggerPoll func()
	pollMu      sync.Mutex
	pollTimer   *time.Timer
}

type clientInfo struct {
	isAdmin bool
}

func NewHub() *Hub { return &Hub{clients: map[*websocket.Conn]clientInfo{}} }

// SetTriggerPoll sets an optional function that will be invoked (debounced)
// shortly after new clients connect. Debouncing avoids immediate repeated
// polls if many clients connect at once.
func (h *Hub) SetTriggerPoll(fn func()) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.triggerPoll = fn
}

// TriggerPollDebounced requests a poll after a short delay (2s). Subsequent
// calls within the debounce window reset the timer.
func (h *Hub) TriggerPollDebounced() {
	h.pollMu.Lock()
	defer h.pollMu.Unlock()
	if h.triggerPoll == nil {
		return
	}
	if h.pollTimer != nil {
		if !h.pollTimer.Stop() {
			select {
			case <-h.pollTimer.C:
			default:
			}
		}
	}
	h.pollTimer = time.AfterFunc(2*time.Second, func() {
		h.pollMu.Lock()
		defer h.pollMu.Unlock()
		if h.triggerPoll != nil {
			h.triggerPoll()
		}
		h.pollTimer = nil
	})
}

// HandleWS upgrades and registers a client.
func (h *Hub) HandleWS(sm *core.StateManager, authValidator func(r *http.Request) (allowed bool, isAdmin bool)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// If this is not a WebSocket upgrade request, return a helpful status.
		if r.Header.Get("Connection") == "" || r.Header.Get("Upgrade") == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUpgradeRequired)
			w.Write([]byte(`{"ok":false,"error":"websocket_upgrade_required"}`))
			return
		}
		var allowed, isAdmin bool
		if authValidator != nil {
			allowed, isAdmin = authValidator(r)
		} else {
			allowed, isAdmin = true, false
		}
		if !allowed {
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
		h.clients[c] = clientInfo{isAdmin: isAdmin}
		clientCount := len(h.clients)
		h.mu.Unlock()
		log.Printf("[WS] client connected (total=%d)", clientCount)
		go func() {
			defer func() { h.mu.Lock(); delete(h.clients, c); h.mu.Unlock(); c.Close(websocket.StatusNormalClosure, "") }()
			for { // discard inbound
				if _, _, err := c.Read(context.Background()); err != nil {
					return
				}
			}
		}()
		// Immediately send current snapshot (apply masking for non-admins)
		snap := sm.Snapshot()
		if !isAdmin {
			maskNodeStateIPs(&snap)
		}
		env := messageEnvelope{MessageType: "STATUS_UPDATE", Data: snap, Timestamp: time.Now().UnixMilli()}
		b, _ := json.Marshal(env)
		if err := c.Write(context.Background(), websocket.MessageText, b); err != nil {
			log.Printf("[WS] write STATUS_UPDATE failed: %v", err)
		}

		// Send initial talker log snapshot
		talkerLog := sm.TalkerLogSnapshot()
		talkerEnv := messageEnvelope{MessageType: "TALKER_LOG_SNAPSHOT", Data: talkerLog, Timestamp: time.Now().UnixMilli()}
		talkerB, _ := json.Marshal(talkerEnv)
		if err := c.Write(context.Background(), websocket.MessageText, talkerB); err != nil {
			log.Printf("[WS] write TALKER_LOG_SNAPSHOT failed: %v", err)
		}

		// Send initial source node keying snapshots (apply masking for non-admins)
		for _, sourceNodeID := range sm.GetSourceNodes() {
			if snapshot, ok := sm.GetSourceNodeSnapshot(sourceNodeID); ok {
				if !isAdmin {
					maskSourceNodeKeyingUpdateIPs(&snapshot)
				}
				snEnv := messageEnvelope{MessageType: "SOURCE_NODE_KEYING", Data: snapshot, Timestamp: time.Now().UnixMilli()}
				snB, _ := json.Marshal(snEnv)
				if err := c.Write(context.Background(), websocket.MessageText, snB); err != nil {
					log.Printf("[WS] write SOURCE_NODE_KEYING failed: %v", err)
				}
			}
		}
		// Log that initial snapshot sequence was sent
		log.Printf("[WS] initial snapshot sent to client")

		// Trigger a debounced poll to enrich state shortly after a new client connects.
		// This ensures freshly connected clients don't see empty or partial state
		// during quiet periods when the periodic poller hasn't run yet.
		h.TriggerPollDebounced()
	}
}

// BroadcastLoop listens for state updates and fans out.
func (h *Hub) BroadcastLoop(updates <-chan core.NodeState) {
	for st := range updates {
		// Build both admin and masked payloads once
		adminEnv := messageEnvelope{MessageType: "STATUS_UPDATE", Data: st, Timestamp: time.Now().UnixMilli()}
		adminPayload, _ := json.Marshal(adminEnv)
		masked := st
		maskNodeStateIPs(&masked)
		maskedEnv := messageEnvelope{MessageType: "STATUS_UPDATE", Data: masked, Timestamp: time.Now().UnixMilli()}
		maskedPayload, _ := json.Marshal(maskedEnv)
		h.mu.RLock()
		for c, info := range h.clients {
			if info.isAdmin {
				go func(conn *websocket.Conn, p []byte) { conn.Write(context.Background(), websocket.MessageText, p) }(c, adminPayload)
			} else {
				go func(conn *websocket.Conn, p []byte) { conn.Write(context.Background(), websocket.MessageText, p) }(c, maskedPayload)
			}
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
		// Prepare admin and masked variants
		adminEnv := messageEnvelope{MessageType: "LINK_ADDED", Data: added, Timestamp: time.Now().UnixMilli()}
		adminPayload, _ := json.Marshal(adminEnv)
		maskedSlice := make([]core.LinkInfo, len(added))
		copy(maskedSlice, added)
		for i := range maskedSlice {
			maskedSlice[i].IP = maskIP(maskedSlice[i].IP)
		}
		maskedEnv := messageEnvelope{MessageType: "LINK_ADDED", Data: maskedSlice, Timestamp: time.Now().UnixMilli()}
		maskedPayload, _ := json.Marshal(maskedEnv)
		h.mu.RLock()
		for c, info := range h.clients {
			if info.isAdmin {
				go func(conn *websocket.Conn, p []byte) { conn.Write(context.Background(), websocket.MessageText, p) }(c, adminPayload)
			} else {
				go func(conn *websocket.Conn, p []byte) { conn.Write(context.Background(), websocket.MessageText, p) }(c, maskedPayload)
			}
		}
		h.mu.RUnlock()
		// Trigger a debounced poll after link additions to enrich state (e.g., elapsed, IP)
		h.TriggerPollDebounced()
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
		// Trigger a debounced poll after link removals to confirm state and enrich details.
		h.TriggerPollDebounced()
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
	// We'll bump a state version periodically to force clients to re-evaluate
	// state after long disconnects. Count ticks and bump every 12 ticks (approx 60s).
	tickCount := 0
	for range ticker.C {
		tickCount++
		if tickCount >= 12 {
			// bump state version to indicate a long-running heartbeat tick
			sm.BumpStateVersion()
			tickCount = 0
		}
		snap := sm.Snapshot()
		// Build both admin and masked payloads
		adminEnv := messageEnvelope{MessageType: "STATUS_UPDATE", Data: snap, Timestamp: time.Now().UnixMilli()}
		adminPayload, _ := json.Marshal(adminEnv)
		masked := snap
		maskNodeStateIPs(&masked)
		maskedEnv := messageEnvelope{MessageType: "STATUS_UPDATE", Data: masked, Timestamp: time.Now().UnixMilli()}
		maskedPayload, _ := json.Marshal(maskedEnv)
		h.mu.RLock()
		for c, info := range h.clients {
			if info.isAdmin {
				go func(conn *websocket.Conn, p []byte) { conn.Write(context.Background(), websocket.MessageText, p) }(c, adminPayload)
			} else {
				go func(conn *websocket.Conn, p []byte) { conn.Write(context.Background(), websocket.MessageText, p) }(c, maskedPayload)
			}
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

// SourceNodeKeyingLoop broadcasts source node keying state updates
func (h *Hub) SourceNodeKeyingLoop(updates <-chan core.SourceNodeKeyingUpdate) {
	for update := range updates {
		// Build both admin and masked payloads
		adminEnv := messageEnvelope{MessageType: "SOURCE_NODE_KEYING", Data: update, Timestamp: time.Now().UnixMilli()}
		adminPayload, _ := json.Marshal(adminEnv)
		maskedUpdate := update
		maskSourceNodeKeyingUpdateIPs(&maskedUpdate)
		maskedEnv := messageEnvelope{MessageType: "SOURCE_NODE_KEYING", Data: maskedUpdate, Timestamp: time.Now().UnixMilli()}
		maskedPayload, _ := json.Marshal(maskedEnv)
		h.mu.RLock()
		for c, info := range h.clients {
			if info.isAdmin {
				go func(conn *websocket.Conn, p []byte) { conn.Write(context.Background(), websocket.MessageText, p) }(c, adminPayload)
			} else {
				go func(conn *websocket.Conn, p []byte) { conn.Write(context.Background(), websocket.MessageText, p) }(c, maskedPayload)
			}
		}
		h.mu.RUnlock()
	}
}

// SourceNodeKeyingEventLoop broadcasts session edge events (TX_START/TX_END)
func (h *Hub) SourceNodeKeyingEventLoop(events <-chan core.SourceNodeKeyingEvent) {
	for event := range events {
		env := messageEnvelope{MessageType: "SOURCE_NODE_KEYING_EVENT", Data: event, Timestamp: time.Now().UnixMilli()}
		payload, _ := json.Marshal(env)
		h.mu.RLock()
		for c := range h.clients {
			go func(conn *websocket.Conn, p []byte) { conn.Write(context.Background(), websocket.MessageText, p) }(c, payload)
		}
		h.mu.RUnlock()
	}
}

// BroadcastTallyCompleted emits a GAMIFICATION_TALLY_COMPLETED event with an optional summary payload
func (h *Hub) BroadcastTallyCompleted(summary interface{}) {
	env := messageEnvelope{MessageType: "GAMIFICATION_TALLY_COMPLETED", Data: summary, Timestamp: time.Now().UnixMilli()}
	payload, _ := json.Marshal(env)
	h.mu.RLock()
	for c := range h.clients {
		go func(conn *websocket.Conn, p []byte) { conn.Write(context.Background(), websocket.MessageText, p) }(c, payload)
	}
	h.mu.RUnlock()
}

// maskIP masks the last two octets of an IPv4 address, leaving others unchanged
func maskIP(ip string) string {
	if ip == "" {
		return ip
	}
	parts := strings.Split(ip, ".")
	if len(parts) == 4 {
		return parts[0] + "." + parts[1] + ".*.*"
	}
	// Non-IPv4: leave as-is (could extend to IPv6 masking later)
	return ip
}

// maskNodeStateIPs mutates the provided NodeState to mask IPs in LinksDetailed
func maskNodeStateIPs(st *core.NodeState) {
	if st == nil {
		return
	}
	// Make a defensive copy of the LinksDetailed slice so masking doesn't
	// mutate the shared backing array that may be referenced elsewhere.
	if len(st.LinksDetailed) == 0 {
		return
	}
	copied := make([]core.LinkInfo, len(st.LinksDetailed))
	copy(copied, st.LinksDetailed)
	for i := range copied {
		copied[i].IP = maskIP(copied[i].IP)
	}
	st.LinksDetailed = copied
}

// maskSourceNodeKeyingUpdateIPs mutates the update to mask IPs in AdjacentNodes
func maskSourceNodeKeyingUpdateIPs(upd *core.SourceNodeKeyingUpdate) {
	if upd == nil || len(upd.AdjacentNodes) == 0 {
		return
	}
	masked := make(map[int]core.AdjacentNodeStatus, len(upd.AdjacentNodes))
	for k, v := range upd.AdjacentNodes {
		v.IP = maskIP(v.IP)
		masked[k] = v
	}
	upd.AdjacentNodes = masked
}
