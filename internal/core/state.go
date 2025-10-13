package core

import (
	"log"
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
	StateVersion  int64      `json:"state_version"`
	SessionStart  time.Time  `json:"session_start"`
	Title         string     `json:"title,omitempty"`
	Subtitle      string     `json:"subtitle,omitempty"`
	NumLinks      int        `json:"num_links"`  // Total links (global)
	NumALinks     int        `json:"num_alinks"` // Adjacent links (local)
}

// SourceNodeKeyingUpdate represents a keying state update for a source node's adjacent links
type SourceNodeKeyingUpdate struct {
	SourceNodeID  int                        `json:"source_node_id"`
	AdjacentNodes map[int]AdjacentNodeStatus `json:"adjacent_nodes"`
	TxKeyed       bool                       `json:"tx_keyed"` // Local TX state for this source node
	RxKeyed       bool                       `json:"rx_keyed"` // Local RX state for this source node
	NumLinks      int                        `json:"num_links,omitempty"`
	NumALinks     int                        `json:"num_alinks,omitempty"`
	Timestamp     time.Time                  `json:"timestamp"`
}

// SourceNodeKeyingEvent represents a TX session edge event (start or end)
type SourceNodeKeyingEvent struct {
	Type         string     `json:"type"` // "TX_START" or "TX_END"
	SourceNodeID int        `json:"source_node_id"`
	NodeID       int        `json:"node_id"` // Adjacent node that started/ended TX
	StartTime    time.Time  `json:"start_time"`
	EndTime      *time.Time `json:"end_time,omitempty"`     // Only set for TX_END
	DurationSec  int        `json:"duration_sec,omitempty"` // Only set for TX_END
	Timestamp    time.Time  `json:"timestamp"`
}

// TransmissionLogRepo defines the interface for transmission log persistence
type TransmissionLogRepo interface {
	LogTransmission(sourceID, adjacentLinkID int, callsign string, startTime, endTime time.Time, durationSeconds int) error
}

// transmissionLogEntry represents a single transmission log entry to be persisted asynchronously
type transmissionLogEntry struct {
	SourceID        int
	AdjacentLinkID  int
	Callsign        string
	TimestampStart  time.Time
	TimestampEnd    time.Time
	DurationSeconds int
}

// StateManager updates and publishes state snapshots.
type StateManager struct {
	mu                    sync.RWMutex
	state                 NodeState
	out                   chan NodeState
	lastTx                bool
	talkerOut             chan TalkerEvent
	log                   *TalkerLog
	linkDiffOut           chan []LinkInfo
	linkRemOut            chan []int
	linkTxOut             chan LinkTxEvent
	persistFn             func(ls []LinkInfo)
	lastTalkerState       map[int]string // Track last TX event kind per node to prevent duplicate talker events
	nodeLookup            *NodeLookupService
	keyingTrackers        map[int]*KeyingTracker      // Per-source-node keying trackers
	keyingOut             chan SourceNodeKeyingUpdate // Channel for source node keying updates
	keyingEventOut        chan SourceNodeKeyingEvent  // Channel for session edge events (TX_START/TX_END)
	numLinks              int                         // Total number of links (global)
	numALinks             int                         // Number of adjacent links (local)
	perSourceNumLinks     map[int]int                 // Per-source total links (server-provided or derived)
	perSourceNumALinks    map[int]int                 // Per-source adjacent links (server-provided or derived)
	lastALinksProcessedAt time.Time                   // Track when we last processed ALINKS to avoid duplicate LINKS processing
	txLogRepo             TransmissionLogRepo         // Repository for logging transmissions
	txLogChan             chan transmissionLogEntry   // Async channel for transmission logging
}

func NewStateManager() *StateManager {
	sm := &StateManager{
		state:              NodeState{UpdatedAt: time.Now(), Version: "0.1.0", SessionStart: time.Now()},
		out:                make(chan NodeState, 8),
		talkerOut:          make(chan TalkerEvent, 16),
		log:                NewTalkerLog(200, 10*time.Minute),
		linkDiffOut:        make(chan []LinkInfo, 8),
		linkRemOut:         make(chan []int, 8),
		linkTxOut:          make(chan LinkTxEvent, 16),
		lastTalkerState:    make(map[int]string),
		keyingTrackers:     make(map[int]*KeyingTracker),
		keyingOut:          make(chan SourceNodeKeyingUpdate, 16),
		keyingEventOut:     make(chan SourceNodeKeyingEvent, 16),
		txLogChan:          make(chan transmissionLogEntry, 32),
		perSourceNumLinks:  make(map[int]int),
		perSourceNumALinks: make(map[int]int),
	}
	// Start async transmission logger
	go sm.transmissionLogWorker()
	return sm
}

// SetNodeLookup configures the node lookup service for enriching link info
func (sm *StateManager) SetNodeLookup(nls *NodeLookupService) {
	sm.nodeLookup = nls
}

// SetTransmissionLogRepo configures the transmission log repository
func (sm *StateManager) SetTransmissionLogRepo(repo TransmissionLogRepo) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.txLogRepo = repo
}

// transmissionLogWorker processes transmission log entries asynchronously
func (sm *StateManager) transmissionLogWorker() {
	for entry := range sm.txLogChan {
		if sm.txLogRepo != nil {
			if err := sm.txLogRepo.LogTransmission(
				entry.SourceID,
				entry.AdjacentLinkID,
				entry.Callsign,
				entry.TimestampStart,
				entry.TimestampEnd,
				entry.DurationSeconds,
			); err != nil {
				log.Printf("[TX LOG ERROR] failed to persist transmission: %v", err)
			}
		}
	}
}

// queueTransmissionLog queues a transmission log entry for async persistence
// This method safely retrieves the callsign by querying the keying tracker
func (sm *StateManager) queueTransmissionLog(sourceID, adjacentID int, startTime, endTime time.Time, durationSec int) {
	// Retrieve callsign from keying tracker (requires lock)
	sm.mu.RLock()
	tracker, exists := sm.keyingTrackers[sourceID]
	sm.mu.RUnlock()

	if !exists {
		return
	}

	// Get adjacent node info (GetAdjacentNode acquires its own lock)
	adjacentNode, found := tracker.GetAdjacentNode(adjacentID)
	callsign := "unknown"
	if found {
		callsign = adjacentNode.Callsign
		if callsign == "" {
			callsign = "unknown"
		}
	}

	select {
	case sm.txLogChan <- transmissionLogEntry{
		SourceID:        sourceID,
		AdjacentLinkID:  adjacentID,
		Callsign:        callsign,
		TimestampStart:  startTime,
		TimestampEnd:    endTime,
		DurationSeconds: durationSec,
	}:
	default:
		log.Printf("[TX LOG WARN] transmission log channel full, dropping entry")
	}
}

func (sm *StateManager) Updates() <-chan NodeState                    { return sm.out }
func (sm *StateManager) TalkerEvents() <-chan TalkerEvent             { return sm.talkerOut }
func (sm *StateManager) TalkerLogSnapshot() any                       { return sm.enrichTalkerSnapshot(sm.log.Snapshot()) }
func (sm *StateManager) LinkUpdates() <-chan []LinkInfo               { return sm.linkDiffOut }
func (sm *StateManager) LinkRemovals() <-chan []int                   { return sm.linkRemOut }
func (sm *StateManager) LinkTxEvents() <-chan LinkTxEvent             { return sm.linkTxOut }
func (sm *StateManager) KeyingUpdates() <-chan SourceNodeKeyingUpdate { return sm.keyingOut }
func (sm *StateManager) KeyingEvents() <-chan SourceNodeKeyingEvent   { return sm.keyingEventOut }

// enrichTalkerSnapshot enriches talker events with current node lookup data
func (sm *StateManager) enrichTalkerSnapshot(events []TalkerEvent) []TalkerEvent {
	if sm.nodeLookup == nil {
		return events
	}

	sm.mu.RLock()
	defer sm.mu.RUnlock()

	enriched := make([]TalkerEvent, len(events))
	for i, evt := range events {
		enriched[i] = evt

		// Skip if already enriched or node is 0
		if evt.Node == 0 || evt.Callsign != "" {
			continue
		}

		// Try to enrich from current LinksDetailed
		found := false
		for j := range sm.state.LinksDetailed {
			if sm.state.LinksDetailed[j].Node == evt.Node {
				enriched[i].Callsign = sm.state.LinksDetailed[j].NodeCallsign
				enriched[i].Description = sm.state.LinksDetailed[j].NodeDescription
				found = true
				break
			}
		}

		// If not found in links, try node lookup service directly
		if !found && evt.Node > 0 {
			if info := sm.nodeLookup.LookupNode(evt.Node); info != nil {
				enriched[i].Callsign = info.Callsign
				enriched[i].Description = info.Description
			}
		} else if !found && evt.Node < 0 {
			// Handle text nodes
			if name, ok := getTextNodeName(evt.Node); ok {
				enriched[i].Callsign = name
				enriched[i].Description = "VOIP Client"
			}
		}
	}

	return enriched
}

// SetPersistHook installs a callback invoked with full LinksDetailed slice after each apply where TX edges occurred.
func (sm *StateManager) SetPersistHook(fn func([]LinkInfo)) { sm.persistFn = fn }

// Run consumes AMI messages and applies them to state.
func (sm *StateManager) Run(msgs <-chan ami.Message) {
	for m := range msgs {
		// Minimal logging for observability: log message type and Event/ActionID headers only.
		if m.Headers != nil {
			if ev, ok := m.Headers["Event"]; ok {
				log.Printf("[AMI EVENT] type=%s event=%s action_id=%s", string(m.Type), ev, m.Headers["ActionID"])
			} else if _, ok := m.Headers["ActionID"]; ok {
				log.Printf("[AMI EVENT] type=%s action_id=%s", string(m.Type), m.Headers["ActionID"])
			} else {
				log.Printf("[AMI EVENT] type=%s (no Event/ActionID)", string(m.Type))
			}
		}
		sm.apply(m)
	}
}

func (sm *StateManager) apply(m ami.Message) {
	if len(m.Headers) == 0 {
		return
	}
	sm.mu.Lock()
	// track if this apply cycle emitted any per-link TX events; if so we should avoid emitting
	// ambiguous global talker events (node==0) for the same activity.
	perLinkEmitted := false
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
		sm.numALinks = len(ids) // Track number of adjacent links

		// Fabricate RPT_LINKS from parsed IDs so downstream link processing populates Links/LinksDetailed
		if _, hasLinks := m.Headers["RPT_LINKS"]; !hasLinks {
			if len(ids) > 0 {
				parts := make([]string, len(ids))
				for i, id := range ids {
					parts[i] = strconv.Itoa(id)
				}
				m.Headers["RPT_LINKS"] = strings.Join(parts, ",")
			}
		}

		// Always log parsed ALINKS for debugging
		prev := make([]int, len(sm.state.Links))
		copy(prev, sm.state.Links)
		log.Printf("[ALINKS DEBUG] parsed ids=%v keyed=%v previous_links=%v", ids, keyedMap, prev)

		// Process keying trackers for each configured source node
		now := time.Now()
		for sourceNodeID, tracker := range sm.keyingTrackers {
			// Process ALINKS for this source node's tracker
			tracker.ProcessALinks(ids, keyedMap, now)

			// Enrich tracker nodes with lookup data
			if sm.nodeLookup != nil {
				for _, nodeID := range ids {
					if nodeID > 0 {
						if info := sm.nodeLookup.LookupNode(nodeID); info != nil {
							tracker.UpdateNodeInfo(nodeID, info.Callsign, info.Description)
						}
					} else if nodeID < 0 {
						// Handle hashed/text nodes by resolving to original name
						if name, ok := getTextNodeName(nodeID); ok {
							tracker.UpdateNodeInfo(nodeID, name, "VOIP Client")
						} else if name, ok := ami.GetTextNodeFromAMI(nodeID); ok {
							tracker.UpdateNodeInfo(nodeID, name, "VOIP Client")
						}
					}
				}
			} else {
				// Even without nodeLookup, attempt to enrich negative nodes from registries
				for _, nodeID := range ids {
					if nodeID < 0 {
						if name, ok := getTextNodeName(nodeID); ok {
							tracker.UpdateNodeInfo(nodeID, name, "VOIP Client")
						} else if name, ok := ami.GetTextNodeFromAMI(nodeID); ok {
							tracker.UpdateNodeInfo(nodeID, name, "VOIP Client")
						}
					}
				}
			}

			// Update per-source counters
			sm.perSourceNumALinks[sourceNodeID] = len(ids)
			// Derive per-source links count from LinksDetailed scoped to this local node if available
			perLinks := 0
			for i := range sm.state.LinksDetailed {
				if sm.state.LinksDetailed[i].LocalNode == sourceNodeID {
					perLinks++
				}
			}
			if perLinks == 0 {
				perLinks = sm.numLinks // fallback to global until LinksDetailed populated
			}
			sm.perSourceNumLinks[sourceNodeID] = perLinks

			// Emit update for this source node (sm.mu is already locked in apply)
			sm.emitKeyingUpdateLocked(sourceNodeID, now)

			// Log keying summary
			keyedCount := 0
			for _, isKeyed := range keyedMap {
				if isKeyed {
					keyedCount++
				}
			}
			log.Printf("[KEYING] Source %d: %d adjacent nodes, %d keyed", sourceNodeID, len(ids), keyedCount)
		}

		// Mark that we just processed ALINKS so we can skip redundant legacy-only RPT_LINKS processing for trackers
		sm.lastALinksProcessedAt = now
	} else if v, ok := m.Headers["RPT_LINKS"]; ok && len(sm.keyingTrackers) > 0 {
		// FALLBACK: If RPT_ALINKS not available, use RPT_LINKS to at least populate the node list
		// (without keying status information)
		// Skip if we just processed ALINKS within the last 500ms to avoid duplicate processing
		now := time.Now()
		shouldSkip := !sm.lastALinksProcessedAt.IsZero() && now.Sub(sm.lastALinksProcessedAt) < 500*time.Millisecond

		if !shouldSkip {
			ids := parseLinkIDs(v)
			emptyKeyedMap := make(map[int]bool) // No keying info available

			log.Printf("[STATE] Processing RPT_LINKS fallback: %d adjacent nodes", len(ids))

			// Process keying trackers with empty keying map
			for sourceNodeID, tracker := range sm.keyingTrackers {
				tracker.ProcessALinks(ids, emptyKeyedMap, now)

				// Enrich tracker nodes with lookup data
				if sm.nodeLookup != nil {
					for _, nodeID := range ids {
						if nodeID > 0 {
							if info := sm.nodeLookup.LookupNode(nodeID); info != nil {
								tracker.UpdateNodeInfo(nodeID, info.Callsign, info.Description)
							}
						} else if nodeID < 0 {
							if name, ok := getTextNodeName(nodeID); ok {
								tracker.UpdateNodeInfo(nodeID, name, "VOIP Client")
							} else if name, ok := ami.GetTextNodeFromAMI(nodeID); ok {
								tracker.UpdateNodeInfo(nodeID, name, "VOIP Client")
							}
						}
					}
				} else {
					// Without lookup, still enrich negative nodes from registries
					for _, nodeID := range ids {
						if nodeID < 0 {
							if name, ok := getTextNodeName(nodeID); ok {
								tracker.UpdateNodeInfo(nodeID, name, "VOIP Client")
							} else if name, ok := ami.GetTextNodeFromAMI(nodeID); ok {
								tracker.UpdateNodeInfo(nodeID, name, "VOIP Client")
							}
						}
					}
				}

				// Update per-source counters
				sm.perSourceNumALinks[sourceNodeID] = len(ids)
				// Derive per-source links count from LinksDetailed scoped to this local node if available
				perLinks := 0
				for i := range sm.state.LinksDetailed {
					if sm.state.LinksDetailed[i].LocalNode == sourceNodeID {
						perLinks++
					}
				}
				if perLinks == 0 {
					perLinks = sm.numLinks // fallback to global until LinksDetailed populated
				}
				sm.perSourceNumLinks[sourceNodeID] = perLinks

				// Emit update for this source node (sm.mu is already locked in apply)
				sm.emitKeyingUpdateLocked(sourceNodeID, now)
			}
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
	// Track number of links (RPT_NUMLINKS = global, RPT_NUMALINKS = adjacent)
	if v, ok := m.Headers["RPT_NUMLINKS"]; ok {
		if n, err := strconv.Atoi(v); err == nil {
			sm.numLinks = n
			sm.state.NumLinks = n
		}
	}
	if v, ok := m.Headers["RPT_NUMALINKS"]; ok {
		if n, err := strconv.Atoi(v); err == nil {
			sm.numALinks = n
			sm.state.NumALinks = n
		}
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
				// Preserve existing link but ensure LocalNode is set for multi-node setups
				if li.LocalNode == 0 && sm.state.NodeID != 0 {
					li.LocalNode = sm.state.NodeID
				}
				newDetails = append(newDetails, *li)
			} else {
				// New link - set LocalNode from primary node ID
				ni := LinkInfo{
					Node:           id,
					LocalNode:      sm.state.NodeID, // Set LocalNode for multi-node compatibility
					ConnectedSince: now,
				}
				// Enrich with node lookup data
				if sm.nodeLookup != nil {
					sm.nodeLookup.EnrichLinkInfo(&ni)
				} else if id < 0 {
					// Enrich text nodes even without nodeLookup service
					if name, found := getTextNodeName(id); found {
						ni.NodeCallsign = name
						ni.NodeDescription = "VOIP Client"
					} else if name, found := ami.GetTextNodeFromAMI(id); found {
						// Fallback to AMI registry
						ni.NodeCallsign = name
						ni.NodeDescription = "VOIP Client"
					}
				}
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
			log.Printf("[STATE] link additions: %v", added)
			select {
			case sm.linkDiffOut <- added:
			default:
			}
		}
		if len(removed) > 0 {
			log.Printf("[STATE] link removals: %v", removed)
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
				nodeID := newDetails[i].Node
				newActive := newDetails[i].CurrentTx

				// Determine the new event kind
				newKind := "TX_STOP"
				if newActive {
					newKind = "TX_START"
				}

				// Check against last known talker state to prevent duplicate events
				lastKind, seen := sm.lastTalkerState[nodeID]

				if !seen || lastKind != newKind {
					// Minimal logging only: indicate node transition without dumping old state
					// State changed or first time seeing this node
					sm.lastTalkerState[nodeID] = newKind

					kind := "STOP"
					if newActive {
						kind = "START"
					}
					evt := LinkTxEvent{Node: newDetails[i].Node, Kind: kind, At: now, TotalTxSeconds: newDetails[i].TotalTxSeconds, LastTxStart: newDetails[i].LastTxStart, LastTxEnd: newDetails[i].LastTxEnd}
					log.Printf("[STATE] link tx event: node=%d kind=%s", evt.Node, evt.Kind)
					select {
					case sm.linkTxOut <- evt:
					default:
					}
					// Emit talker event with node info, passing the LinkInfo for accurate duration
					sm.emitTalkerFromLink("TX_"+kind, &newDetails[i])
					emitted = true
				}
			}
			if emitted {
				perLinkEmitted = true
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

	// Process keying tracker timers on every event to ensure unkey timers expire properly
	// This is critical because timers need to be checked even when no ALINKS events arrive
	now := time.Now()
	for sourceNodeID, tracker := range sm.keyingTrackers {
		// Process the tracker's timer queue
		if tracker.ProcessTimers(now) {
			// Timers were processed, emit update
			sm.emitKeyingUpdateLocked(sourceNodeID, now)
		}
	}

	// Talker edge detection (TX start/stop)
	// Only emit global (node==0) talker events if no per-link TX events were emitted in this apply cycle.
	if !sm.lastTx && sm.state.TxKeyed {
		if !perLinkEmitted {
			sm.emitTalker("TX_START", 0)
		}
	}
	if sm.lastTx && !sm.state.TxKeyed {
		if !perLinkEmitted {
			sm.emitTalker("TX_STOP", 0)
		}
	}
	sm.lastTx = sm.state.TxKeyed
	snap := sm.state
	sm.mu.Unlock()
	select {
	case sm.out <- snap:
	default:
	}
}

func (sm *StateManager) emitTalker(kind string, node int) {
	now := time.Now()
	evt := TalkerEvent{At: now, Kind: kind, Node: node}

	// Enrich with node information if available
	if node != 0 {
		// Look up node in LinksDetailed to get callsign info
		for i := range sm.state.LinksDetailed {
			if sm.state.LinksDetailed[i].Node == node {
				evt.Callsign = sm.state.LinksDetailed[i].NodeCallsign
				evt.Description = sm.state.LinksDetailed[i].NodeDescription

				// For STOP events, calculate duration if we have start time
				if kind == "TX_STOP" && sm.state.LinksDetailed[i].LastTxStart != nil {
					evt.Duration = int(now.Sub(*sm.state.LinksDetailed[i].LastTxStart).Seconds())
				}
				break
			}
		}

		// If no callsign found in LinksDetailed, check if it's a text node
		if evt.Callsign == "" && node < 0 {
			if name, found := getTextNodeName(node); found {
				evt.Callsign = name
				evt.Description = "VOIP Client"
			}
		}
	}

	// Check for duplicate: skip if last state for this node matches current kind
	// This prevents duplicate events from being stored in the ring buffer
	if lastState, exists := sm.lastTalkerState[node]; exists && lastState == kind {
		// log.Printf("DEBUG: Skipping duplicate talker event: node=%d kind=%s", node, kind)
		return // Already in this state, skip duplicate
	}
	sm.lastTalkerState[node] = kind

	// log.Printf("DEBUG: Adding talker event to buffer: node=%d kind=%s callsign=%s", node, kind, evt.Callsign)
	sm.log.Add(evt)
	select {
	case sm.talkerOut <- evt:
	default:
	}
}

// emitTalkerFromLink emits a talker event with data from a LinkInfo struct
// This is used when we have the LinkInfo with accurate timestamps
func (sm *StateManager) emitTalkerFromLink(kind string, link *LinkInfo) {
	if link == nil {
		return
	}

	now := time.Now()
	evt := TalkerEvent{
		At:          now,
		Kind:        kind,
		Node:        link.Node,
		Callsign:    link.NodeCallsign,
		Description: link.NodeDescription,
	}

	// For STOP events, calculate duration from the LinkInfo's timestamps
	if kind == "TX_STOP" && link.LastTxStart != nil && link.LastTxEnd != nil {
		evt.Duration = int(link.LastTxEnd.Sub(*link.LastTxStart).Seconds())
	}

	// If no callsign, check if it's a text node
	if evt.Callsign == "" && link.Node < 0 {
		if name, found := getTextNodeName(link.Node); found {
			evt.Callsign = name
			evt.Description = "VOIP Client"
		}
	}

	// Check for duplicate: skip if last state for this node matches current kind
	// This prevents duplicate events from being stored in the ring buffer
	if lastState, exists := sm.lastTalkerState[link.Node]; exists && lastState == kind {
		// log.Printf("DEBUG: Skipping duplicate talker event (from link): node=%d kind=%s", link.Node, kind)
		return // Already in this state, skip duplicate
	}
	sm.lastTalkerState[link.Node] = kind

	// log.Printf("DEBUG: Adding talker event to buffer (from link): node=%d kind=%s callsign=%s", link.Node, kind, evt.Callsign)
	sm.log.Add(evt)
	select {
	case sm.talkerOut <- evt:
	default:
	}
}

func (sm *StateManager) Snapshot() NodeState { sm.mu.RLock(); defer sm.mu.RUnlock(); return sm.state }

// BumpStateVersion increments the opaque state version counter. This can be used by
// external loops (e.g., heartbeat) to force clients to re-evaluate state when they
// reconnect after a long disconnect. It is safe to call concurrently.
func (sm *StateManager) BumpStateVersion() {
	sm.mu.Lock()
	sm.state.StateVersion++
	sm.mu.Unlock()
}

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

// SetTitle sets the application title to be exposed on STATUS_UPDATE snapshots.
func (sm *StateManager) SetTitle(title string) {
	sm.mu.Lock()
	sm.state.Title = title
	sm.mu.Unlock()
}

// SetSubtitle sets the application subtitle to be exposed on STATUS_UPDATE snapshots.
func (sm *StateManager) SetSubtitle(subtitle string) {
	sm.mu.Lock()
	sm.state.Subtitle = subtitle
	sm.mu.Unlock()
}

// SetNodeID sets the primary node ID to be exposed on STATUS_UPDATE snapshots.
func (sm *StateManager) SetNodeID(nodeID int) {
	sm.mu.Lock()
	sm.state.NodeID = nodeID
	sm.mu.Unlock()
}

// SeedKeyingTrackerFromLinks populates the keying tracker with existing link data
// This is useful on startup when links are loaded from persistence before AMI events arrive
func (sm *StateManager) SeedKeyingTrackerFromLinks(sourceNodeID int) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	tracker, exists := sm.keyingTrackers[sourceNodeID]
	if !exists {
		return
	}

	// Get link IDs from current state
	linkIDs := make([]int, len(sm.state.Links))
	copy(linkIDs, sm.state.Links)

	if len(linkIDs) == 0 {
		return
	}

	// Process with empty keying map (no keying status available yet)
	emptyKeyedMap := make(map[int]bool)
	now := time.Now()
	tracker.ProcessALinks(linkIDs, emptyKeyedMap, now)

	// Enrich with node lookup data and link details
	if sm.nodeLookup != nil {
		for _, nodeID := range linkIDs {
			if info := sm.nodeLookup.LookupNode(nodeID); info != nil {
				tracker.UpdateNodeInfo(nodeID, info.Callsign, info.Description)
			}
		}
	}

	// Update with detailed link info (ConnectedSince, Mode, Direction, IP)
	for _, linkInfo := range sm.state.LinksDetailed {
		tracker.UpdateConnectedSince(linkInfo.Node, linkInfo.ConnectedSince)
		tracker.UpdateLinkInfo(linkInfo.Node, linkInfo.Mode, linkInfo.Direction, linkInfo.IP)
	}

	log.Printf("[KEYING] Seeded tracker for source %d with %d links", sourceNodeID, len(linkIDs))
}

// AddSourceNode adds a source node and creates a keying tracker for it
func (sm *StateManager) AddSourceNode(nodeID int, delayMS int) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.keyingTrackers[nodeID]; exists {
		return // Already exists
	}

	tracker := NewKeyingTracker(nodeID, delayMS)

	// Set up callbacks for TX events
	// IMPORTANT: These callbacks are invoked from ProcessALinks which holds BOTH sm.mu and kt.mu locks
	// Therefore callbacks must NOT call any methods that acquire locks
	tracker.SetCallbacks(
		func(sourceNode, adjacentNode int, timestamp time.Time) {
			// TX START callback - called with locks already held
			// Just emit the event; the keying update will be sent after ProcessALinks completes
			sm.emitKeyingEventLocked(SourceNodeKeyingEvent{
				Type:         "TX_START",
				SourceNodeID: sourceNode,
				NodeID:       adjacentNode,
				StartTime:    timestamp,
				Timestamp:    timestamp,
			})
		},
		func(sourceNode, adjacentNode int, timestamp time.Time, duration int) {
			// TX END callback - called with locks already held
			// We can't call GetAdjacentNode here because it would try to acquire kt.mu again
			// So we pass minimal info and rely on the keying update for full state
			endTime := timestamp
			startTime := timestamp.Add(-time.Duration(duration) * time.Second)
			sm.emitKeyingEventLocked(SourceNodeKeyingEvent{
				Type:         "TX_END",
				SourceNodeID: sourceNode,
				NodeID:       adjacentNode,
				StartTime:    startTime,
				EndTime:      &endTime,
				DurationSec:  duration,
				Timestamp:    timestamp,
			})

			// Queue transmission log entry for async persistence
			// This is safe to call here as queueTransmissionLog doesn't block
			go sm.queueTransmissionLog(sourceNode, adjacentNode, startTime, endTime, duration)
		},
	)

	sm.keyingTrackers[nodeID] = tracker
}

// emitKeyingEventLocked emits a session edge event (must be called with sm.mu already locked)
func (sm *StateManager) emitKeyingEventLocked(event SourceNodeKeyingEvent) {
	log.Printf("[KEYING EVENT] type=%s source=%d node=%d duration=%ds", event.Type, event.SourceNodeID, event.NodeID, event.DurationSec)
	select {
	case sm.keyingEventOut <- event:
	default:
	}
}

// emitKeyingUpdateLocked emits a keying update for a source node (must be called with sm.mu already locked)
func (sm *StateManager) emitKeyingUpdateLocked(sourceNodeID int, timestamp time.Time) {
	tracker, exists := sm.keyingTrackers[sourceNodeID]
	if !exists {
		return
	}

	// Derive per-source counters (fallback to global if missing)
	perLinks := sm.perSourceNumLinks[sourceNodeID]
	perALinks := sm.perSourceNumALinks[sourceNodeID]
	if perLinks == 0 {
		perLinks = sm.numLinks
	}
	if perALinks == 0 {
		perALinks = sm.numALinks
	}
	update := SourceNodeKeyingUpdate{
		SourceNodeID:  sourceNodeID,
		AdjacentNodes: tracker.GetAdjacentNodes(),
		TxKeyed:       sm.state.TxKeyed,
		RxKeyed:       sm.state.RxKeyed,
		NumLinks:      perLinks,
		NumALinks:     perALinks,
		Timestamp:     timestamp,
	}

	// Log the keying update emission for observability (low-noise single-line)
	log.Printf("[KEYING UPDATE] source=%d adjacent_count=%d tx=%t rx=%t timestamp=%s", update.SourceNodeID, len(update.AdjacentNodes), update.TxKeyed, update.RxKeyed, update.Timestamp.Format(time.RFC3339))

	select {
	case sm.keyingOut <- update:
	default:
	}
}

// GetSourceNodes returns a list of configured source node IDs
func (sm *StateManager) GetSourceNodes() []int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	nodes := make([]int, 0, len(sm.keyingTrackers))
	for nodeID := range sm.keyingTrackers {
		nodes = append(nodes, nodeID)
	}
	return nodes
}

// GetSourceNodeSnapshot returns a snapshot of a source node's keying state
func (sm *StateManager) GetSourceNodeSnapshot(nodeID int) (SourceNodeKeyingUpdate, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	tracker, exists := sm.keyingTrackers[nodeID]
	if !exists {
		return SourceNodeKeyingUpdate{}, false
	}

	perLinks := sm.perSourceNumLinks[nodeID]
	perALinks := sm.perSourceNumALinks[nodeID]
	if perLinks == 0 {
		perLinks = sm.numLinks
	}
	if perALinks == 0 {
		perALinks = sm.numALinks
	}
	return SourceNodeKeyingUpdate{
		SourceNodeID:  nodeID,
		AdjacentNodes: tracker.GetAdjacentNodes(),
		TxKeyed:       sm.state.TxKeyed,
		RxKeyed:       sm.state.RxKeyed,
		NumLinks:      perLinks,
		NumALinks:     perALinks,
		Timestamp:     time.Now(),
	}, true
}

// ApplyCombinedStatus updates state from XStat+SawStat combined data
func (sm *StateManager) ApplyCombinedStatus(combined *ami.CombinedNodeStatus) {
	if combined == nil {
		return
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Note: In multi-node setups, multiple pollers update the same StateManager.
	// We track RxKeyed/TxKeyed globally, but it may be overwritten by different nodes.
	// For accurate per-link TX tracking, we use combined.Node to identify which node's data this is.
	sm.state.RxKeyed = combined.RxKeyed
	sm.state.TxKeyed = combined.TxKeyed

	// Build lookup of existing LinkInfo using composite key (LocalNode:RemoteNode)
	// This allows the same remote node to be connected to multiple local nodes
	type linkKey struct {
		localNode  int
		remoteNode int
	}
	existing := map[linkKey]*LinkInfo{}
	for i := range sm.state.LinksDetailed {
		key := linkKey{sm.state.LinksDetailed[i].LocalNode, sm.state.LinksDetailed[i].Node}
		existing[key] = &sm.state.LinksDetailed[i]
	}

	// Track previous keys for THIS local node only
	previousSet := map[linkKey]struct{}{}
	for i := range sm.state.LinksDetailed {
		if sm.state.LinksDetailed[i].LocalNode == combined.Node {
			key := linkKey{sm.state.LinksDetailed[i].LocalNode, sm.state.LinksDetailed[i].Node}
			previousSet[key] = struct{}{}
		}
	}

	now := time.Now()
	newDetails := make([]LinkInfo, 0, len(combined.Connections))
	var added []LinkInfo
	currentSet := map[linkKey]struct{}{}

	// Process each connection from combined status
	for _, conn := range combined.Connections {
		key := linkKey{combined.Node, conn.Node}
		currentSet[key] = struct{}{}

		var li LinkInfo
		// If we already have this link for THIS local node, copy existing data
		if existingLi, ok := existing[key]; ok {
			li = *existingLi
		} else {
			// New connection
			li = LinkInfo{
				Node:           conn.Node,
				LocalNode:      combined.Node, // Track which local node this link belongs to
				ConnectedSince: now,
			}
		}

		// Update which local node this link belongs to (important for multi-node setups)
		li.LocalNode = combined.Node

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

		// Update CurrentTx: Track when the REMOTE node is transmitting (talking)
		// Prefer KeyingInfo.IsKeyed (from SawStat) over conn.IsKeyed (from XStat) for consistency
		// with "Keying" status in Last Heard column
		isCurrentlyKeyed := conn.IsKeyed
		if conn.KeyingInfo != nil {
			isCurrentlyKeyed = conn.KeyingInfo.IsKeyed
		}
		li.UpdateTx(isCurrentlyKeyed, now)

		// Enrich with node lookup data (callsign, description, location)
		if sm.nodeLookup != nil {
			sm.nodeLookup.EnrichLinkInfo(&li)
			log.Printf("[STATE DEBUG] Enriched node %d via nodeLookup: callsign=%s, desc=%s", li.Node, li.NodeCallsign, li.NodeDescription)
		} else if li.Node < 0 {
			// Enrich text nodes even without nodeLookup service
			if name, found := getTextNodeName(li.Node); found {
				li.NodeCallsign = name
				li.NodeDescription = "VOIP Client"
				log.Printf("[STATE DEBUG] Enriched negative node %d from core registry: %s", li.Node, name)
			} else if name, found := ami.GetTextNodeFromAMI(li.Node); found {
				// Fallback to AMI registry
				li.NodeCallsign = name
				li.NodeDescription = "VOIP Client"
				log.Printf("[STATE DEBUG] Enriched negative node %d from AMI registry: %s", li.Node, name)
			} else {
				log.Printf("[STATE DEBUG] Failed to enrich negative node %d - not found in any registry", li.Node)
			}
		}

		newDetails = append(newDetails, li)

		// Track if this is a new connection
		if _, wasPresent := previousSet[key]; !wasPresent {
			added = append(added, li)
		}
	}

	// Compute removals (removed links for THIS local node only)
	var removed []int
	for key := range previousSet {
		if _, still := currentSet[key]; !still {
			removed = append(removed, key.remoteNode)
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
		// Clean up talker state for removed nodes
		for _, nodeID := range removed {
			delete(sm.lastTalkerState, nodeID)
		}
	}

	// Emit per-link TX start/stop events
	var emitted bool
	for i := range newDetails {
		nodeID := newDetails[i].Node
		newActive := newDetails[i].CurrentTx

		// Determine the new event kind
		newKind := "TX_STOP"
		if newActive {
			newKind = "TX_START"
		}

		// Check against last known talker state (not just existing link state)
		// This prevents duplicate events when multiple pollers see the same link
		lastKind, seen := sm.lastTalkerState[nodeID]

		if !seen || lastKind != newKind {
			// State changed or first time seeing this node
			sm.lastTalkerState[nodeID] = newKind

			kind := "STOP"
			if newActive {
				kind = "START"
			}
			evt := LinkTxEvent{
				Node:           nodeID,
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
			// Emit a talker event associated with this node so UI can show per-node duration
			sm.emitTalker("TX_"+kind, nodeID)
			emitted = true
		}
	}

	// Call persist hook if TX edges occurred
	if emitted && sm.persistFn != nil {
		sm.persistFn(newDetails)
	}

	// Update state: In multi-node setups, merge links from this node with links from other nodes
	// Remove old links for this local node, then add new ones
	mergedLinks := make([]LinkInfo, 0, len(sm.state.LinksDetailed))
	for i := range sm.state.LinksDetailed {
		// Keep links that belong to other local nodes (but not LocalNode=0, which are legacy/seeded)
		// LocalNode=0 means unassigned/legacy, so we clear those out and let pollers repopulate
		if sm.state.LinksDetailed[i].LocalNode != 0 && sm.state.LinksDetailed[i].LocalNode != combined.Node {
			mergedLinks = append(mergedLinks, sm.state.LinksDetailed[i])
		}
	}
	// Add new links for this local node
	mergedLinks = append(mergedLinks, newDetails...)

	sm.state.LinksDetailed = mergedLinks
	nodeIDs := make([]int, len(mergedLinks))
	for i := range mergedLinks {
		nodeIDs[i] = mergedLinks[i].Node
	}
	sm.state.Links = nodeIDs
	sm.state.UpdatedAt = now
	sm.state.Heartbeat = now.UnixMilli()

	// Talker edge detection (TX start/stop)
	if !sm.lastTx && sm.state.TxKeyed {
		sm.emitTalker("TX_START", 0)
	}
	if sm.lastTx && !sm.state.TxKeyed {
		sm.emitTalker("TX_STOP", 0)
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
//	Non-numeric: "KF8S", "W1ABC", etc.
//
// Strategy: split by comma, drop leading token if it contains only a count, then
// try to extract digits. If no digits found, convert text identifier to stable int ID.
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
	digitRe := regexp.MustCompile(`(\d{3,})`) // Changed from {3,7} to {3,} for longer node numbers
	for i := start; i < len(tokens); i++ {
		tk := strings.TrimSpace(tokens[i])
		if tk == "" {
			continue
		}

		// First try to parse as a plain integer (handles negative node IDs from internal fabrication)
		// This needs to match node IDs that are at least 3 digits or less than -999
		if n, err := strconv.Atoi(tk); err == nil && (n < -999 || n >= 1000) {
			if _, dup := seen[n]; !dup {
				out = append(out, n)
				seen[n] = struct{}{}
			}
			continue
		}

		// Try to find embedded digits (handles T588841, 588841TU, etc.)
		m := digitRe.FindStringSubmatch(tk)
		if len(m) > 1 {
			if n, err := strconv.Atoi(m[1]); err == nil {
				if _, dup := seen[n]; !dup {
					out = append(out, n)
					seen[n] = struct{}{}
				}
				continue
			}
		}

		// No numeric match - must be a text node ID (callsign, etc.)
		// Strip mode prefix and status suffixes
		cleaned := tk

		// Always remove leading T if present (mode prefix)
		if len(cleaned) > 0 && cleaned[0] == 'T' {
			cleaned = cleaned[1:]
		}

		// Strip status suffixes (TU, TK, TR, TC, TM)
		cleaned = strings.TrimSuffix(cleaned, "TU")
		cleaned = strings.TrimSuffix(cleaned, "TK")
		cleaned = strings.TrimSuffix(cleaned, "TR")
		cleaned = strings.TrimSuffix(cleaned, "TC")
		cleaned = strings.TrimSuffix(cleaned, "TM")

		if cleaned == "" {
			continue
		}

		// Convert to stable integer using hash function
		nodeID := textNodeToInt(cleaned)
		if _, dup := seen[nodeID]; !dup {
			out = append(out, nodeID)
			seen[nodeID] = struct{}{}
			// Store mapping for later lookup
			registerTextNode(nodeID, cleaned)
		}
	}
	return out
}

// textNodeToInt converts a text node identifier to a stable integer ID
// Uses FNV-1a hash mapped to negative range to avoid collision with numeric nodes
func textNodeToInt(s string) int {
	s = strings.ToUpper(s) // Normalize to uppercase
	hash := uint32(2166136261)
	for i := 0; i < len(s); i++ {
		hash ^= uint32(s[i])
		hash *= 16777619
	}
	// Convert to negative number to distinguish from numeric AllStar nodes
	// Use lower 30 bits to keep values reasonable
	return -int(hash & 0x3FFFFFFF)
}

// Global map to store text node ID to original string mapping
var textNodeMap = sync.Map{} // map[int]string

func registerTextNode(nodeID int, text string) {
	textNodeMap.Store(nodeID, strings.ToUpper(text))
}

func getTextNodeName(nodeID int) (string, bool) {
	if val, ok := textNodeMap.Load(nodeID); ok {
		return val.(string), true
	}
	return "", false
}

// GetTextNodeName returns the original text name for a hashed node ID (public API)
func GetTextNodeName(nodeID int) (string, bool) {
	return getTextNodeName(nodeID)
}

// parseALinks parses RPT_ALINKS payload which includes flags appended to node IDs.
// Example: 6,588841TU,590110TU,586671TU,58840TU,550465TK,586081TU
// Also supports: KF8STK, W1ABCTU, etc (text nodes with keying flags)
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
	digitRe := regexp.MustCompile(`(\d{3,})`) // Changed from {3,7} to {3,} for longer node numbers
	seen := map[int]struct{}{}
	for i := start; i < len(parts); i++ {
		p := strings.TrimSpace(parts[i])
		if p == "" {
			continue
		}

		// Check for keying flag before stripping
		isKeyed := strings.Contains(p, "TK") || strings.HasSuffix(p, "K")

		// Try to extract numeric node ID first
		m := digitRe.FindStringSubmatch(p)
		if len(m) > 1 {
			if n, err := strconv.Atoi(m[1]); err == nil {
				if _, dup := seen[n]; !dup {
					ids = append(ids, n)
					seen[n] = struct{}{}
				}
				if isKeyed {
					keyed[n] = true
				}
				continue
			}
		}

		// No numeric match - must be text node (callsign, etc.)
		// Strip mode prefix and status suffixes
		cleaned := p

		// Always remove leading T if present (mode prefix)
		if len(cleaned) > 0 && cleaned[0] == 'T' {
			cleaned = cleaned[1:]
		}

		// Strip status suffixes (TU, TK, TR, TC, TM)
		cleaned = strings.TrimSuffix(cleaned, "TU")
		cleaned = strings.TrimSuffix(cleaned, "TK")
		cleaned = strings.TrimSuffix(cleaned, "TR")
		cleaned = strings.TrimSuffix(cleaned, "TC")
		cleaned = strings.TrimSuffix(cleaned, "TM")

		if cleaned == "" {
			continue
		}

		nodeID := textNodeToInt(cleaned)
		if _, dup := seen[nodeID]; !dup {
			ids = append(ids, nodeID)
			seen[nodeID] = struct{}{}
			registerTextNode(nodeID, cleaned)
		}
		if isKeyed {
			keyed[nodeID] = true
		}
	}
	return ids, keyed
}
