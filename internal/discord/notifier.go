package discord

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/dbehnke/allstar-nexus/internal/core"
)

// ActivityState represents the current state of a node
type ActivityState int

const (
	StateIdle ActivityState = iota
	StateActive
	StateQSO
)

func (s ActivityState) String() string {
	switch s {
	case StateIdle:
		return "idle"
	case StateActive:
		return "active"
	case StateQSO:
		return "qso"
	default:
		return "unknown"
	}
}

// TalkerInfo tracks information about an individual talker
type TalkerInfo struct {
	NodeID      int
	Callsign    string
	Description string
	StartTime   time.Time
	LastSeen    time.Time
}

// Config holds Discord notifier configuration
type Config struct {
	WebhookURL            string
	QSOInactiveSeconds    int
	NodeIdleSeconds       int
	MinTalkersForQSO      int
	NotifyIndividualTalks bool
	CheckInterval         time.Duration // Internal check interval (default 5s, can be lower for testing)
}

// Notifier handles Discord webhook notifications for node activity
type Notifier struct {
	config        Config
	webhookURL    string
	client        *http.Client
	mu            sync.RWMutex
	currentState  ActivityState
	activeTalkers map[int]*TalkerInfo // map of node ID to talker info
	lastActivity  time.Time
	nodeID        int // The node ID being monitored
	nodeName      string
	stopCh        chan struct{}
	wg            sync.WaitGroup
	nodeLookup    NodeLookupService // Interface for looking up node info
}

// NodeLookupService interface for looking up node information
type NodeLookupService interface {
	LookupNode(nodeID int) *core.NodeInfo
}

// NewNotifier creates a new Discord notifier
func NewNotifier(config Config, nodeID int, nodeName string) *Notifier {
	if nodeName == "" {
		nodeName = fmt.Sprintf("%d", nodeID)
	}
	if config.CheckInterval == 0 {
		config.CheckInterval = 5 * time.Second // Default check interval
	}
	return &Notifier{
		config:        config,
		webhookURL:    config.WebhookURL,
		client:        &http.Client{Timeout: 10 * time.Second},
		currentState:  StateIdle,
		activeTalkers: make(map[int]*TalkerInfo),
		nodeID:        nodeID,
		nodeName:      nodeName,
		stopCh:        make(chan struct{}),
	}
}

// SetNodeLookup sets the node lookup service for enriching node information
func (n *Notifier) SetNodeLookup(lookup NodeLookupService) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.nodeLookup = lookup
}

// Start begins monitoring for state transitions
func (n *Notifier) Start() {
	n.wg.Add(1)
	go n.monitorStateTransitions()
}

// Stop halts the notifier
func (n *Notifier) Stop() {
	close(n.stopCh)
	n.wg.Wait()
}

// ProcessTalkerEvent processes a talker event from the state manager
func (n *Notifier) ProcessTalkerEvent(evt core.TalkerEvent) {
	n.mu.Lock()
	defer n.mu.Unlock()

	// Debug logging to track all events received
	log.Printf("[DISCORD DEBUG] Received TalkerEvent: kind=%s node=%d callsign=%s", evt.Kind, evt.Node, evt.Callsign)

	now := time.Now()
	n.lastActivity = now

	// Skip node==0 events (these are global/unspecific events)
	if evt.Node == 0 {
		log.Printf("[DISCORD DEBUG] Skipping node==0 event - use LinkTxEvents for actual node tracking")
		return
	}

	switch evt.Kind {
	case "TX_START":
		// New talker started
		if _, exists := n.activeTalkers[evt.Node]; !exists {
			// First time seeing this talker in current session
			n.activeTalkers[evt.Node] = &TalkerInfo{
				NodeID:      evt.Node,
				Callsign:    evt.Callsign,
				Description: evt.Description,
				StartTime:   now,
				LastSeen:    now,
			}

			log.Printf("[DISCORD DEBUG] New talker added: node=%d callsign=%s, total_talkers=%d", evt.Node, evt.Callsign, len(n.activeTalkers))

			// Notify about individual talker if enabled
			if n.config.NotifyIndividualTalks {
				callsignInfo := fmt.Sprintf("%s (%d)", evt.Callsign, evt.Node)
				if evt.Callsign == "" {
					callsignInfo = fmt.Sprintf("Node %d", evt.Node)
				}
				n.sendNotification(fmt.Sprintf("%s is now talking.", callsignInfo))
			}

			// Check for state transitions
			n.checkStateTransition()
		} else {
			// Update last seen time
			n.activeTalkers[evt.Node].LastSeen = now
			log.Printf("[DISCORD DEBUG] Updated existing talker: node=%d", evt.Node)
		}

	case "TX_STOP":
		// Talker stopped - keep them in activeTalkers for now, cleanup happens in monitor
		if talker, exists := n.activeTalkers[evt.Node]; exists {
			talker.LastSeen = now
			log.Printf("[DISCORD DEBUG] Talker stopped: node=%d", evt.Node)
		}
	}
}

// ProcessLinkTxEvent processes a per-link transmit event
func (n *Notifier) ProcessLinkTxEvent(evt core.LinkTxEvent) {
	n.mu.Lock()
	defer n.mu.Unlock()

	// Debug logging to track all link events received
	log.Printf("[DISCORD DEBUG] Received LinkTxEvent: kind=%s node=%d", evt.Kind, evt.Node)

	now := time.Now()
	n.lastActivity = now

	// Skip invalid nodes
	if evt.Node == 0 {
		log.Printf("[DISCORD DEBUG] Skipping node==0 link event")
		return
	}

	// Look up callsign information
	var callsign, description string
	if n.nodeLookup != nil {
		if info := n.nodeLookup.LookupNode(evt.Node); info != nil {
			callsign = info.Callsign
			description = info.Description
			log.Printf("[DISCORD DEBUG] Node lookup successful: node=%d callsign=%s", evt.Node, callsign)
		} else {
			log.Printf("[DISCORD DEBUG] Node lookup returned nil for node=%d", evt.Node)
		}
	} else {
		log.Printf("[DISCORD DEBUG] No node lookup service available")
	}

	switch evt.Kind {
	case "START":
		// New talker started
		if _, exists := n.activeTalkers[evt.Node]; !exists {
			// First time seeing this talker in current session
			n.activeTalkers[evt.Node] = &TalkerInfo{
				NodeID:      evt.Node,
				Callsign:    callsign,
				Description: description,
				StartTime:   now,
				LastSeen:    now,
			}

			log.Printf("[DISCORD DEBUG] New talker added from link event: node=%d callsign=%s, total_talkers=%d", evt.Node, callsign, len(n.activeTalkers))

			// Notify about individual talker if enabled
			if n.config.NotifyIndividualTalks {
				// Format: Callsign (NodeID) or just Node NodeID if no callsign
				var talkerInfo string
				if callsign != "" {
					talkerInfo = fmt.Sprintf("%s (%d)", callsign, evt.Node)
				} else {
					talkerInfo = fmt.Sprintf("Node %d", evt.Node)
				}
				n.sendNotification(fmt.Sprintf("📻 %s is now talking.", talkerInfo))
			}

			// Check for state transitions
			n.checkStateTransition()
		} else {
			// Update last seen time and enrich with callsign if we didn't have it before
			talker := n.activeTalkers[evt.Node]
			talker.LastSeen = now
			if talker.Callsign == "" && callsign != "" {
				talker.Callsign = callsign
				talker.Description = description
			}
			log.Printf("[DISCORD DEBUG] Updated existing talker from link event: node=%d", evt.Node)
		}

	case "STOP":
		// Talker stopped - keep them in activeTalkers for now, cleanup happens in monitor
		if talker, exists := n.activeTalkers[evt.Node]; exists {
			talker.LastSeen = now
			log.Printf("[DISCORD DEBUG] Talker stopped from link event: node=%d", evt.Node)
		}
	}
}

// checkStateTransition evaluates current state and triggers notifications if state changes
func (n *Notifier) checkStateTransition() {
	oldState := n.currentState
	newState := n.evaluateState()

	if oldState != newState {
		n.currentState = newState
		n.notifyStateChange(oldState, newState)
	}
}

// evaluateState determines what state the node should be in based on active talkers
func (n *Notifier) evaluateState() ActivityState {
	talkerCount := len(n.activeTalkers)

	if talkerCount == 0 {
		return StateIdle
	} else if talkerCount >= n.config.MinTalkersForQSO {
		return StateQSO
	} else {
		return StateActive
	}
}

// notifyStateChange sends a notification about a state transition
func (n *Notifier) notifyStateChange(oldState, newState ActivityState) {
	switch newState {
	case StateActive:
		if oldState == StateIdle {
			n.sendNotification(fmt.Sprintf("✨ Node %s has activity!", n.nodeName))
		}
	case StateQSO:
		if oldState != StateQSO {
			n.sendNotification("🎙️ A QSO has started!")
		}
	case StateIdle:
		if oldState == StateQSO {
			// QSO ended first, then idle
			n.sendNotification("👋 A QSO has ended!")
			// Add a small delay before sending the second notification to ensure they appear
			// as separate messages with distinct timestamps in Discord
			time.Sleep(2 * time.Second)
			n.sendNotification(fmt.Sprintf("💤 Node %s is now idle.", n.nodeName))
		} else if oldState == StateActive {
			// Just went from active to idle (no QSO)
			n.sendNotification(fmt.Sprintf("💤 Node %s is now idle.", n.nodeName))
		}
	}
}

// monitorStateTransitions periodically checks for timeouts and state changes
func (n *Notifier) monitorStateTransitions() {
	defer n.wg.Done()

	ticker := time.NewTicker(n.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-n.stopCh:
			return
		case <-ticker.C:
			n.checkTimeouts()
		}
	}
}

// checkTimeouts removes inactive talkers and checks for state transitions
func (n *Notifier) checkTimeouts() {
	n.mu.Lock()
	defer n.mu.Unlock()

	now := time.Now()
	qsoTimeout := time.Duration(n.config.QSOInactiveSeconds) * time.Second
	idleTimeout := time.Duration(n.config.NodeIdleSeconds) * time.Second

	// Track if we removed any talkers
	talkersRemoved := false

	// Remove talkers who haven't been seen recently
	for nodeID, talker := range n.activeTalkers {
		if now.Sub(talker.LastSeen) > qsoTimeout {
			delete(n.activeTalkers, nodeID)
			talkersRemoved = true
		}
	}

	// If talkers were removed, check for state transition
	if talkersRemoved {
		n.checkStateTransition()
	}

	// Check if we should transition to idle due to prolonged inactivity
	if n.currentState == StateQSO || n.currentState == StateActive {
		if now.Sub(n.lastActivity) > idleTimeout && len(n.activeTalkers) == 0 {
			// Transition to idle
			oldState := n.currentState
			n.currentState = StateIdle
			n.notifyStateChange(oldState, StateIdle)
		}
	}
}

// sendNotification sends a message to the Discord webhook
func (n *Notifier) sendNotification(message string) {
	if n.webhookURL == "" {
		log.Printf("[DISCORD] webhook URL not configured, skipping: %s", message)
		return
	}

	payload := map[string]interface{}{
		"content": message,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[DISCORD] failed to marshal payload: %v", err)
		return
	}

	go func() {
		req, err := http.NewRequest("POST", n.webhookURL, bytes.NewBuffer(data))
		if err != nil {
			log.Printf("[DISCORD] failed to create request: %v", err)
			return
		}

		req.Header.Set("Content-Type", "application/json")

		resp, err := n.client.Do(req)
		if err != nil {
			log.Printf("[DISCORD] failed to send notification: %v", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			log.Printf("[DISCORD] webhook returned status %d", resp.StatusCode)
		} else {
			log.Printf("[DISCORD] notification sent: %s", message)
		}
	}()
}

// GetCurrentState returns the current activity state (for testing/debugging)
func (n *Notifier) GetCurrentState() ActivityState {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.currentState
}

// GetActiveTalkerCount returns the number of currently active talkers
func (n *Notifier) GetActiveTalkerCount() int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return len(n.activeTalkers)
}
