package core

import (
	"sync"
	"time"
)

// KeyingTracker implements jitter-compensated keying tracking with 2-second delay
// Based on the AMI events analysis document specifications
type KeyingTracker struct {
	mu                sync.RWMutex
	localNodeID       int                          // The local source node ID
	adjacentNodes     map[int]*AdjacentNodeStatus  // Map of adjacent node ID -> status
	timerQueue        []UnkeyCheckTimer            // Queue of pending unkey confirmations
	delayMS           int                          // Delay in milliseconds (default 2000)
	onTxStart         func(sourceNode, adjacentNode int, timestamp time.Time)
	onTxEnd           func(sourceNode, adjacentNode int, timestamp time.Time, duration int)
}

// AdjacentNodeStatus tracks the keying state of an adjacent node
type AdjacentNodeStatus struct {
	NodeID          int        `json:"NodeID"`
	IsKeyed         bool       `json:"IsKeyed"`
	KeyedStartTime  *time.Time `json:"KeyedStartTime,omitempty"`
	IsTransmitting  bool       `json:"IsTransmitting"`
	PendingUnkey    bool       `json:"PendingUnkey"` // True when unkey timer is scheduled (during 2s delay)
	TotalTxSeconds  int        `json:"TotalTxSeconds"`
	LastTxEnd       *time.Time `json:"LastTxEnd,omitempty"` // Timestamp of last transmission end

	// Node information for display
	Callsign        string `json:"Callsign,omitempty"`
	Description     string `json:"Description,omitempty"`

	// Link information
	Mode            string    `json:"Mode,omitempty"`
	Direction       string    `json:"Direction,omitempty"`
	IP              string    `json:"IP,omitempty"`
	ConnectedSince  time.Time `json:"ConnectedSince"`
}

// UnkeyCheckTimer represents a scheduled unkey confirmation check
type UnkeyCheckTimer struct {
	Action        string    // "UnkeyCheck"
	SourceNodeID  int       // Local source node
	AdjacentNodeID int      // Adjacent node to check
	ExecutionTime time.Time // When to execute the check
}

// NewKeyingTracker creates a new keying tracker with jitter compensation
func NewKeyingTracker(localNodeID int, delayMS int) *KeyingTracker {
	if delayMS <= 0 {
		delayMS = 2000 // Default 2 second delay
	}

	return &KeyingTracker{
		localNodeID:   localNodeID,
		adjacentNodes: make(map[int]*AdjacentNodeStatus),
		timerQueue:    make([]UnkeyCheckTimer, 0),
		delayMS:       delayMS,
	}
}

// SetCallbacks sets the TX start/end callbacks
// IMPORTANT: Callbacks are invoked while holding kt.mu lock, so they must not call methods that acquire kt.mu
func (kt *KeyingTracker) SetCallbacks(
	onStart func(sourceNode, adjacentNode int, timestamp time.Time),
	onEnd func(sourceNode, adjacentNode int, timestamp time.Time, duration int),
) {
	kt.mu.Lock()
	defer kt.mu.Unlock()
	kt.onTxStart = onStart
	kt.onTxEnd = onEnd
}

// ProcessALinks processes RPT_ALINKS event with keying status
// alinksKeyed is a map of adjacent node ID -> keyed status (true if keyed)
func (kt *KeyingTracker) ProcessALinks(adjacentNodeIDs []int, alinksKeyed map[int]bool, timestamp time.Time) {
	kt.mu.Lock()
	defer kt.mu.Unlock()

	// First, process timer queue for any expired unkey checks
	kt.processTimerQueue(timestamp)

	// Process each adjacent node from RPT_ALINKS
	for _, nodeID := range adjacentNodeIDs {
		linkIsKeyed := alinksKeyed[nodeID]

		// Get or initialize node status
		nodeStatus, exists := kt.adjacentNodes[nodeID]
		if !exists {
			nodeStatus = &AdjacentNodeStatus{
				NodeID:         nodeID,
				IsKeyed:        false,
				IsTransmitting: false,
				ConnectedSince: timestamp,
			}
			kt.adjacentNodes[nodeID] = nodeStatus
		}

		// --- 1. Key-Up (TX START) Detection ---
		if linkIsKeyed && !nodeStatus.IsTransmitting {
			// Start a new tracked session for this node
			nodeStatus.IsKeyed = true
			nodeStatus.IsTransmitting = true
			nodeStatus.PendingUnkey = false
			startTime := timestamp
			nodeStatus.KeyedStartTime = &startTime

			// Clear any pending 'UnkeyCheck' from the timer queue for this node
			kt.removeFromQueue(nodeID)

			// Emit TX_START callback
			if kt.onTxStart != nil {
				kt.onTxStart(kt.localNodeID, nodeID, timestamp)
			}
		} else if !linkIsKeyed && nodeStatus.IsTransmitting {
			// --- 2. Key-Down (Start Delay Timer) ---
			// Node just unkeyed in the event, but we start the jitter compensation timer
			nodeStatus.IsKeyed = false // Temporarily set to FALSE
			nodeStatus.PendingUnkey = true // Mark that we're in the unkey delay period
			executionTime := timestamp.Add(time.Duration(kt.delayMS) * time.Millisecond)

			// Add an action to the queue to check if the unkey is final
			kt.timerQueue = append(kt.timerQueue, UnkeyCheckTimer{
				Action:         "UnkeyCheck",
				SourceNodeID:   kt.localNodeID,
				AdjacentNodeID: nodeID,
				ExecutionTime:  executionTime,
			})
		} else if linkIsKeyed && nodeStatus.IsTransmitting {
			// --- 3. Ongoing Keying (Keep state true and clear timers) ---
			// The node is still keyed, indicating the previous 'UnkeyCheck' was a jitter event.
			nodeStatus.IsKeyed = true // Ensure state is true
			nodeStatus.PendingUnkey = false // Clear pending unkey flag
			kt.removeFromQueue(nodeID)
		}
	}
}

// ProcessTimers processes expired timers and returns true if any timers were processed
// This is a public method that can be called from StateManager to check timers on every event
func (kt *KeyingTracker) ProcessTimers(currentTime time.Time) bool {
	kt.mu.Lock()
	defer kt.mu.Unlock()
	return kt.processTimerQueue(currentTime)
}

// processTimerQueue checks for expired timers and confirms unkey events (must be called with lock held)
// Returns true if any timers were processed
func (kt *KeyingTracker) processTimerQueue(currentTime time.Time) bool {
	newQueue := make([]UnkeyCheckTimer, 0, len(kt.timerQueue))
	processed := false

	for _, timer := range kt.timerQueue {
		if timer.ExecutionTime.After(currentTime) {
			// Timer not yet expired, keep it
			newQueue = append(newQueue, timer)
			continue
		}

		// Timer expired - check if unkey is confirmed
		if timer.Action == "UnkeyCheck" {
			nodeStatus, exists := kt.adjacentNodes[timer.AdjacentNodeID]
			if !exists {
				continue
			}

			// FINAL CONFIRMATION: If no 'K' event was received during the delay
			if !nodeStatus.IsKeyed && nodeStatus.IsTransmitting {
				// Confirmed TX END
				kt.processTxEnd(currentTime, timer.AdjacentNodeID, nodeStatus)
				processed = true
			}
		}
	}

	kt.timerQueue = newQueue
	return processed
}

// processTxEnd handles confirmed TX end for an adjacent node
func (kt *KeyingTracker) processTxEnd(endTime time.Time, adjacentNodeID int, nodeStatus *AdjacentNodeStatus) {
	if nodeStatus.KeyedStartTime == nil {
		return
	}

	// Calculate duration
	duration := int(endTime.Sub(*nodeStatus.KeyedStartTime).Seconds())

	// Update total TX seconds
	nodeStatus.TotalTxSeconds += duration

	// Reset state and record last TX end time
	nodeStatus.IsTransmitting = false
	nodeStatus.PendingUnkey = false
	nodeStatus.KeyedStartTime = nil
	nodeStatus.LastTxEnd = &endTime

	// Emit TX_END callback
	if kt.onTxEnd != nil {
		kt.onTxEnd(kt.localNodeID, adjacentNodeID, endTime, duration)
	}
}

// removeFromQueue removes all UnkeyCheck timers for a specific adjacent node
func (kt *KeyingTracker) removeFromQueue(adjacentNodeID int) {
	newQueue := make([]UnkeyCheckTimer, 0, len(kt.timerQueue))
	for _, timer := range kt.timerQueue {
		if timer.AdjacentNodeID != adjacentNodeID {
			newQueue = append(newQueue, timer)
		}
	}
	kt.timerQueue = newQueue
}

// UpdateNodeInfo updates the display information for an adjacent node
func (kt *KeyingTracker) UpdateNodeInfo(nodeID int, callsign, description string) {
	kt.mu.Lock()
	defer kt.mu.Unlock()

	if nodeStatus, exists := kt.adjacentNodes[nodeID]; exists {
		nodeStatus.Callsign = callsign
		nodeStatus.Description = description
	}
}

// UpdateLinkInfo updates link-specific information for an adjacent node
func (kt *KeyingTracker) UpdateLinkInfo(nodeID int, mode, direction, ip string) {
	kt.mu.Lock()
	defer kt.mu.Unlock()

	if nodeStatus, exists := kt.adjacentNodes[nodeID]; exists {
		nodeStatus.Mode = mode
		nodeStatus.Direction = direction
		nodeStatus.IP = ip
	}
}

// UpdateConnectedSince updates the connection timestamp for an adjacent node
func (kt *KeyingTracker) UpdateConnectedSince(nodeID int, connectedSince time.Time) {
	kt.mu.Lock()
	defer kt.mu.Unlock()

	if nodeStatus, exists := kt.adjacentNodes[nodeID]; exists {
		nodeStatus.ConnectedSince = connectedSince
	}
}

// GetAdjacentNodes returns a snapshot of all adjacent nodes
func (kt *KeyingTracker) GetAdjacentNodes() map[int]AdjacentNodeStatus {
	kt.mu.RLock()
	defer kt.mu.RUnlock()

	snapshot := make(map[int]AdjacentNodeStatus, len(kt.adjacentNodes))
	for id, status := range kt.adjacentNodes {
		snapshot[id] = *status
	}
	return snapshot
}

// GetAdjacentNode returns the status of a specific adjacent node
func (kt *KeyingTracker) GetAdjacentNode(nodeID int) (AdjacentNodeStatus, bool) {
	kt.mu.RLock()
	defer kt.mu.RUnlock()

	if status, exists := kt.adjacentNodes[nodeID]; exists {
		return *status, true
	}
	return AdjacentNodeStatus{}, false
}

// RemoveAdjacentNode removes an adjacent node when it disconnects
func (kt *KeyingTracker) RemoveAdjacentNode(nodeID int) {
	kt.mu.Lock()
	defer kt.mu.Unlock()

	delete(kt.adjacentNodes, nodeID)
	kt.removeFromQueue(nodeID)
}

// GetLocalNodeID returns the local source node ID
func (kt *KeyingTracker) GetLocalNodeID() int {
	kt.mu.RLock()
	defer kt.mu.RUnlock()
	return kt.localNodeID
}
