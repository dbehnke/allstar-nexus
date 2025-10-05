package core

import (
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dbehnke/allstar-nexus/internal/ami"
)

// NodeState represents current (placeholder) node metrics.
type NodeState struct {
	NodeID        int        `json:"node_id"`
	RxKeyed       bool       `json:"rx_keyed"`
	TxKeyed       bool       `json:"tx_keyed"`
	Links         []int      `json:"links"`
	LinksDetailed []LinkInfo `json:"links_detailed,omitempty"`
	UptimeSec     int        `json:"uptime_sec"`
	LastReloadSec int        `json:"last_reload_sec"`
	BootedAt      *time.Time `json:"booted_at,omitempty"`
	BuildTime     string     `json:"build_time,omitempty"`
	UpdatedAt     time.Time  `json:"updated_at"`
	Version       string     `json:"version"`
	Heartbeat     int64      `json:"heartbeat"`
	SessionStart  time.Time  `json:"session_start"`
}

// StateManager updates and publishes state snapshots.
type StateManager struct {
	mu          sync.RWMutex
	state       NodeState
	out         chan NodeState
	lastTx      bool
	talkerOut   chan TalkerEvent
	log         *TalkerLog
	linkDiffOut chan []LinkInfo
	linkRemOut  chan []int
	linkTxOut   chan LinkTxEvent
	persistFn   func(ls []LinkInfo)
}

func NewStateManager() *StateManager {
	return &StateManager{
		state:       NodeState{UpdatedAt: time.Now(), Version: "0.1.0", SessionStart: time.Now()},
		out:         make(chan NodeState, 8),
		talkerOut:   make(chan TalkerEvent, 16),
		log:         NewTalkerLog(200, 10*time.Minute),
		linkDiffOut: make(chan []LinkInfo, 8),
		linkRemOut:  make(chan []int, 8),
		linkTxOut:   make(chan LinkTxEvent, 16),
	}
}

func (sm *StateManager) Updates() <-chan NodeState        { return sm.out }
func (sm *StateManager) TalkerEvents() <-chan TalkerEvent { return sm.talkerOut }
func (sm *StateManager) TalkerLogSnapshot() []TalkerEvent { return sm.log.Snapshot() }
func (sm *StateManager) LinkUpdates() <-chan []LinkInfo   { return sm.linkDiffOut }
func (sm *StateManager) LinkRemovals() <-chan []int       { return sm.linkRemOut }
func (sm *StateManager) LinkTxEvents() <-chan LinkTxEvent { return sm.linkTxOut }

// SetPersistHook installs a callback invoked with full LinksDetailed slice after each apply where TX edges occurred.
func (sm *StateManager) SetPersistHook(fn func([]LinkInfo)) { sm.persistFn = fn }

// Run consumes AMI messages and applies them to state.
func (sm *StateManager) Run(msgs <-chan ami.Message) {
	for m := range msgs {
		sm.apply(m)
	}
}

func (sm *StateManager) apply(m ami.Message) {
	if len(m.Headers) == 0 {
		return
	}
	sm.mu.Lock()
	// We'll capture keyed status map if RPT_ALINKS present to apply after link detail rebuild.
	var alinksKeyed map[int]bool
	// Derive synthetic legacy headers from Event/VarSet frames to unify downstream logic.
	if ev, ok := m.Headers["Event"]; ok {
		switch ev {
		case "RPT_LINKS":
			if v, ok := m.Headers["EventValue"]; ok {
				m.Headers["RPT_LINKS"] = v
			}
		case "RPT_TXKEYED":
			if v, ok := m.Headers["EventValue"]; ok {
				m.Headers["RPT_TXKEYED"] = v
			}
		case "RPT_RXKEYED":
			if v, ok := m.Headers["EventValue"]; ok {
				m.Headers["RPT_RXKEYED"] = v
			}
		case "RPT_ALINKS":
			if v, ok := m.Headers["EventValue"]; ok {
				m.Headers["RPT_ALINKS"] = v
			}
		}
	}
	if ev, ok := m.Headers["Event"]; ok && ev == "VarSet" { // Variable based update
		if variable, ok := m.Headers["Variable"]; ok {
			if value, ok2 := m.Headers["Value"]; ok2 {
				switch variable {
				case "RPT_LINKS":
					m.Headers["RPT_LINKS"] = value
				case "RPT_ALINKS":
					m.Headers["RPT_ALINKS"] = value
				case "RPT_TXKEYED":
					m.Headers["RPT_TXKEYED"] = value
				case "RPT_RXKEYED":
					m.Headers["RPT_RXKEYED"] = value
				}
			}
		}
	}
	// If we received ALINKS but not standard LINKS, fabricate RPT_LINKS header and capture keyed states.
	if v, ok := m.Headers["RPT_ALINKS"]; ok {
		ids, keyedMap := parseALinks(v)
		alinksKeyed = keyedMap
		if _, has := m.Headers["RPT_LINKS"]; !has && len(ids) > 0 {
			b := strings.Builder{}
			for i, id := range ids {
				if i > 0 {
					b.WriteByte(',')
				}
				b.WriteString(strconv.Itoa(id))
			}
			m.Headers["RPT_LINKS"] = b.String()
		}
	}
	// Detect reconnect via banner (Asterisk Call Manager) to reset uptime-related fields.
	if _, ok := m.Headers["Asterisk Call Manager/Version"]; ok {
		sm.state.UptimeSec = 0
		sm.state.LastReloadSec = 0
		sm.state.BootedAt = nil
	}
	// Example mapping heuristics (placeholder until real Allstar vars wired):
	if v, ok := m.Headers["RPT_TXKEYED"]; ok {
		sm.state.TxKeyed = v == "1"
	}
	if v, ok := m.Headers["RPT_RXKEYED"]; ok {
		sm.state.RxKeyed = v == "1"
	}
	if v, ok := m.Headers["RPT_LINKS"]; ok {
		links := parseLinkIDs(v)
		previousSet := map[int]struct{}{}
		for _, id := range sm.state.Links {
			previousSet[id] = struct{}{}
		}
		sm.state.Links = links
		// maintain detailed link info + capture additions/removals
		existing := map[int]*LinkInfo{}
		for i := range sm.state.LinksDetailed {
			existing[sm.state.LinksDetailed[i].Node] = &sm.state.LinksDetailed[i]
		}
		now := time.Now()
		newDetails := make([]LinkInfo, 0, len(links))
		var added []LinkInfo
		currentSet := map[int]struct{}{}
		for _, id := range links {
			currentSet[id] = struct{}{}
			if li, ok := existing[id]; ok {
				newDetails = append(newDetails, *li)
			} else {
				ni := LinkInfo{Node: id, ConnectedSince: now}
				newDetails = append(newDetails, ni)
				added = append(added, ni)
			}
		}
		// compute removals
		var removed []int
		for id := range previousSet {
			if _, still := currentSet[id]; !still {
				removed = append(removed, id)
			}
		}
		if len(added) > 0 {
			select {
			case sm.linkDiffOut <- added:
			default:
			}
		}
		if len(removed) > 0 {
			select {
			case sm.linkRemOut <- removed:
			default:
			}
		}
		// Apply keyed status if we parsed ALINKS keyed map.
		if alinksKeyed != nil {
			for i := range newDetails {
				if active, ok := alinksKeyed[newDetails[i].Node]; ok {
					newDetails[i].UpdateTx(active, now)
				} else {
					// Was previously active? Need to ensure we record stop edge.
					if existingLi, ok := existing[newDetails[i].Node]; ok && existingLi.CurrentTx {
						newDetails[i].UpdateTx(false, now)
					}
				}
			}
			// Emit per-link TX start/stop events by comparing existing vs newDetails.
			var emitted bool
			for i := range newDetails {
				oldActive := false
				if old, ok := existing[newDetails[i].Node]; ok && old.CurrentTx {
					oldActive = true
				}
				newActive := newDetails[i].CurrentTx
				if oldActive != newActive { // edge
					kind := "STOP"
					if newActive {
						kind = "START"
					}
					evt := LinkTxEvent{Node: newDetails[i].Node, Kind: kind, At: now, TotalTxSeconds: newDetails[i].TotalTxSeconds, LastTxStart: newDetails[i].LastTxStart, LastTxEnd: newDetails[i].LastTxEnd}
					select {
					case sm.linkTxOut <- evt:
					default:
					}
					emitted = true
				}
			}
			if emitted && sm.persistFn != nil {
				sm.persistFn(newDetails)
			}
		}
		sm.state.LinksDetailed = newDetails
	}
	// Uptime parsing from FullyBooted (or other) events: keys 'Uptime' and 'LastReload' observed in capture.
	if v, ok := m.Headers["Uptime"]; ok {
		if n, err := strconv.Atoi(v); err == nil {
			sm.state.UptimeSec = n
		}
	}
	if v, ok := m.Headers["LastReload"]; ok {
		if n, err := strconv.Atoi(v); err == nil {
			sm.state.LastReloadSec = n
		}
	}
	if ev, ok := m.Headers["Event"]; ok && ev == "FullyBooted" && sm.state.BootedAt == nil {
		now := time.Now()
		sm.state.BootedAt = &now
	}
	sm.state.UpdatedAt = time.Now()
	sm.state.Heartbeat = time.Now().UnixMilli()
	// Talker edge detection (TX start/stop)
	if !sm.lastTx && sm.state.TxKeyed {
		sm.emitTalker("TX_START")
	}
	if sm.lastTx && !sm.state.TxKeyed {
		sm.emitTalker("TX_STOP")
	}
	sm.lastTx = sm.state.TxKeyed
	snap := sm.state
	sm.mu.Unlock()
	select {
	case sm.out <- snap:
	default:
	}
}

func (sm *StateManager) emitTalker(kind string) {
	evt := TalkerEvent{At: time.Now(), Kind: kind}
	sm.log.Add(evt)
	select {
	case sm.talkerOut <- evt:
	default:
	}
}

func (sm *StateManager) Snapshot() NodeState { sm.mu.RLock(); defer sm.mu.RUnlock(); return sm.state }

// SeedLinkStats seeds LinksDetailed (used on startup from persisted stats) without emitting diff events.
func (sm *StateManager) SeedLinkStats(list []LinkInfo) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.state.LinksDetailed = append([]LinkInfo(nil), list...)
	ids := make([]int, 0, len(list))
	for _, l := range list {
		ids = append(ids, l.Node)
	}
	sm.state.Links = ids
	sm.state.UpdatedAt = time.Now()
}

// SetVersion updates the version string that will be reported in STATUS_UPDATE snapshots.
func (sm *StateManager) SetVersion(v string) {
	sm.mu.Lock()
	sm.state.Version = v
	sm.mu.Unlock()
}

// SetBuildTime sets the build timestamp to be exposed on STATUS_UPDATE snapshots.
func (sm *StateManager) SetBuildTime(t string) {
	sm.mu.Lock()
	sm.state.BuildTime = t
	sm.mu.Unlock()
}

// ApplyCombinedStatus updates state from XStat+SawStat combined data
func (sm *StateManager) ApplyCombinedStatus(combined *ami.CombinedNodeStatus) {
	if combined == nil {
		return
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Update RX/TX keyed state
	sm.state.RxKeyed = combined.RxKeyed
	sm.state.TxKeyed = combined.TxKeyed

	// Build lookup of existing LinkInfo
	existing := map[int]*LinkInfo{}
	for i := range sm.state.LinksDetailed {
		existing[sm.state.LinksDetailed[i].Node] = &sm.state.LinksDetailed[i]
	}

	// Track previous node set for diff detection
	previousSet := map[int]struct{}{}
	for _, id := range sm.state.Links {
		previousSet[id] = struct{}{}
	}

	now := time.Now()
	newDetails := make([]LinkInfo, 0, len(combined.Connections))
	var added []LinkInfo
	currentSet := map[int]struct{}{}

	// Process each connection from combined status
	for _, conn := range combined.Connections {
		currentSet[conn.Node] = struct{}{}

		var li LinkInfo
		// If we already have this link, copy existing data
		if existingLi, ok := existing[conn.Node]; ok {
			li = *existingLi
		} else {
			// New connection
			li = LinkInfo{
				Node:           conn.Node,
				ConnectedSince: now,
			}
		}

		// Update fields from XStat
		li.IP = conn.IP
		li.IsKeyed = conn.IsKeyed
		li.Direction = conn.Direction
		li.Elapsed = conn.Elapsed
		li.LinkType = conn.LinkType
		li.Mode = conn.Mode
		li.LastHeard = conn.LastHeard

		// Update fields from SawStat (KeyingInfo)
		if conn.KeyingInfo != nil {
			li.SecsSinceKeyed = conn.KeyingInfo.SecsSinceKeyed
			li.LastKeyedTime = &conn.KeyingInfo.LastKeyedTime
			if conn.KeyingInfo.IsKeyed {
				li.LastHeardAt = &now
			}
		}

		// Update CurrentTx based on IsKeyed
		li.UpdateTx(conn.IsKeyed, now)

		newDetails = append(newDetails, li)

		// Track if this is a new connection
		if _, wasPresent := previousSet[conn.Node]; !wasPresent {
			added = append(added, li)
		}
	}

	// Compute removals
	var removed []int
	for id := range previousSet {
		if _, still := currentSet[id]; !still {
			removed = append(removed, id)
		}
	}

	// Emit link addition events
	if len(added) > 0 {
		select {
		case sm.linkDiffOut <- added:
		default:
		}
	}

	// Emit link removal events
	if len(removed) > 0 {
		select {
		case sm.linkRemOut <- removed:
		default:
		}
	}

	// Emit per-link TX start/stop events
	var emitted bool
	for i := range newDetails {
		oldActive := false
		if old, ok := existing[newDetails[i].Node]; ok && old.CurrentTx {
			oldActive = true
		}
		newActive := newDetails[i].CurrentTx
		if oldActive != newActive {
			kind := "STOP"
			if newActive {
				kind = "START"
			}
			evt := LinkTxEvent{
				Node:           newDetails[i].Node,
				Kind:           kind,
				At:             now,
				TotalTxSeconds: newDetails[i].TotalTxSeconds,
				LastTxStart:    newDetails[i].LastTxStart,
				LastTxEnd:      newDetails[i].LastTxEnd,
			}
			select {
			case sm.linkTxOut <- evt:
			default:
			}
			emitted = true
		}
	}

	// Call persist hook if TX edges occurred
	if emitted && sm.persistFn != nil {
		sm.persistFn(newDetails)
	}

	// Update state
	sm.state.LinksDetailed = newDetails
	nodeIDs := make([]int, len(newDetails))
	for i := range newDetails {
		nodeIDs[i] = newDetails[i].Node
	}
	sm.state.Links = nodeIDs
	sm.state.UpdatedAt = now
	sm.state.Heartbeat = now.UnixMilli()

	// Talker edge detection (TX start/stop)
	if !sm.lastTx && sm.state.TxKeyed {
		sm.emitTalker("TX_START")
	}
	if sm.lastTx && !sm.state.TxKeyed {
		sm.emitTalker("TX_STOP")
	}
	sm.lastTx = sm.state.TxKeyed

	// Emit state snapshot
	snap := sm.state
	select {
	case sm.out <- snap:
	default:
	}
}

// parseLinkIDs extracts AllStar node IDs from RPT_LINKS style payloads.
// Formats observed:
//
//	"6,T588841,T590110,T586671,T58840,T550465,T586081" (leading count then tokens)
//	"6,588841TU,590110TU,..." (ALINKS variant with suffix flags)
//	Previously polled plain CSV "2000,3000".
//
// Strategy: split by comma, drop leading token if it contains only a count, then
// strip non-digit prefix/suffix from each token and parse digits.
func parseLinkIDs(payload string) []int {
	if payload == "" {
		return nil
	}
	tokens := strings.Split(payload, ",")
	out := make([]int, 0, len(tokens))
	start := 0
	if len(tokens) > 0 {
		if _, err := strconv.Atoi(tokens[0]); err == nil && len(tokens[0]) < 3 { // count is usually small
			start = 1
		}
	}
	seen := map[int]struct{}{}
	digitRe := regexp.MustCompile(`(\d{3,7})`)
	for i := start; i < len(tokens); i++ {
		tk := strings.TrimSpace(tokens[i])
		if tk == "" {
			continue
		}
		// Find first 3-7 digit run.
		m := digitRe.FindStringSubmatch(tk)
		if len(m) > 1 {
			if n, err := strconv.Atoi(m[1]); err == nil {
				if _, dup := seen[n]; !dup {
					out = append(out, n)
					seen[n] = struct{}{}
				}
			}
		}
	}
	return out
}

// parseALinks parses RPT_ALINKS payload which includes flags appended to node IDs.
// Example: 6,588841TU,590110TU,586671TU,58840TU,550465TK,586081TU
// Flags: T indicates currently transmitting? U/K seen in samples (TK vs TU) - we treat presence of trailing 'K' as keyed, 'TU' as idle or unknown.
// We conservatively mark nodes with trailing 'K' OR payload containing 'TK' as active.
func parseALinks(payload string) (ids []int, keyed map[int]bool) {
	keyed = map[int]bool{}
	if payload == "" {
		return nil, keyed
	}
	parts := strings.Split(payload, ",")
	start := 0
	if len(parts) > 0 {
		if _, err := strconv.Atoi(parts[0]); err == nil && len(parts[0]) < 3 { // leading count
			start = 1
		}
	}
	digitRe := regexp.MustCompile(`(\d{3,7})`)
	seen := map[int]struct{}{}
	for i := start; i < len(parts); i++ {
		p := strings.TrimSpace(parts[i])
		if p == "" {
			continue
		}
		m := digitRe.FindStringSubmatch(p)
		if len(m) > 1 {
			if n, err := strconv.Atoi(m[1]); err == nil {
				if _, dup := seen[n]; !dup {
					ids = append(ids, n)
					seen[n] = struct{}{}
				}
				// Mark keyed if token contains 'TK' (observed sample) or ends with 'K'.
				if strings.Contains(p, "TK") || strings.HasSuffix(p, "K") {
					keyed[n] = true
				}
			}
		}
	}
	return ids, keyed
}
