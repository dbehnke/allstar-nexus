package core

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/dbehnke/allstar-nexus/internal/ami"
)

// PollingService periodically queries AMI for node status to ensure data sync
// This provides a hybrid approach: event-driven updates for real-time changes,
// plus periodic polling for verification and enrichment with data not in events
type PollingService struct {
	connector       *ami.Connector
	stateManager    *StateManager
	interval        time.Duration
	nodes           []int
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
	running         bool
	mu              sync.Mutex
	firstPollDone   bool   // Track if first poll completed successfully
	cleanupCallback func() // Optional callback to trigger database cleanup after first poll
}

// NewPollingService creates a new polling service
func NewPollingService(conn *ami.Connector, sm *StateManager, interval time.Duration, nodes []int) *PollingService {
	if interval <= 0 {
		interval = 60 * time.Second // Default to 1 minute
	}

	return &PollingService{
		connector:     conn,
		stateManager:  sm,
		interval:      interval,
		nodes:         nodes,
		firstPollDone: false,
	}
}

// SetCleanupCallback sets a callback to be called after the first successful poll
// This is useful for cleaning up stale database entries that were seeded at startup
func (ps *PollingService) SetCleanupCallback(callback func()) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.cleanupCallback = callback
}

// Start begins the polling loop
func (ps *PollingService) Start() error {
	ps.mu.Lock()
	if ps.running {
		ps.mu.Unlock()
		return nil
	}
	ps.running = true
	ps.ctx, ps.cancel = context.WithCancel(context.Background())
	ps.mu.Unlock()

	log.Printf("[POLLING] Starting periodic polling service (interval=%s, nodes=%v)", ps.interval, ps.nodes)

	// Start polling goroutine for each node
	for _, nodeID := range ps.nodes {
		ps.wg.Add(1)
		go ps.pollNode(nodeID)
	}

	return nil
}

// Stop gracefully stops the polling service
func (ps *PollingService) Stop() {
	ps.mu.Lock()
	if !ps.running {
		ps.mu.Unlock()
		return
	}
	ps.running = false
	ps.cancel()
	ps.mu.Unlock()

	log.Printf("[POLLING] Stopping polling service...")
	ps.wg.Wait()
	log.Printf("[POLLING] Polling service stopped")
}

// pollNode runs the polling loop for a single node
func (ps *PollingService) pollNode(nodeID int) {
	defer ps.wg.Done()

	ticker := time.NewTicker(ps.interval)
	defer ticker.Stop()

	// Perform initial poll immediately (but wait a bit for AMI to stabilize)
	select {
	case <-time.After(5 * time.Second):
		ps.performPoll(nodeID)
	case <-ps.ctx.Done():
		return
	}

	// Then poll at regular intervals
	for {
		select {
		case <-ticker.C:
			ps.performPoll(nodeID)
		case <-ps.ctx.Done():
			return
		}
	}
}

// performPoll executes a single poll cycle for a node
func (ps *PollingService) performPoll(nodeID int) {
	// Check if AMI is connected before polling
	if !ps.connector.IsConnected() {
		log.Printf("[POLLING] Skipping poll for node %d (AMI not connected, waiting for reconnection...)", nodeID)
		return
	}

	// Create context with timeout for this poll
	ctx, cancel := context.WithTimeout(ps.ctx, 10*time.Second)
	defer cancel()

	log.Printf("[POLLING] Polling node %d for status...", nodeID)

	// Get combined status (XStat + SawStat)
	combined, err := ps.connector.GetCombinedStatus(ctx, nodeID)
	if err != nil {
		log.Printf("[POLLING] Failed to get status for node %d: %v (will retry on next interval)", nodeID, err)
		return
	}

	// Log summary
	log.Printf("[POLLING] Node %d: %d connections, RX=%t, TX=%t",
		nodeID, len(combined.Connections), combined.RxKeyed, combined.TxKeyed)

	// Apply to state manager
	ps.stateManager.ApplyCombinedStatus(combined)

	// Update keying tracker if it exists
	// This enriches the tracker with connection details (Direction, IP, Elapsed, Mode)
	ps.updateKeyingTracker(nodeID, combined)

	// After first successful poll, trigger cleanup callback if set
	ps.mu.Lock()
	if !ps.firstPollDone && ps.cleanupCallback != nil {
		ps.firstPollDone = true
		callback := ps.cleanupCallback
		ps.mu.Unlock()
		log.Printf("[POLLING] First poll completed successfully - triggering database cleanup")
		callback()
	} else {
		ps.mu.Unlock()
	}
}

// updateKeyingTracker enriches the keying tracker with polling data
func (ps *PollingService) updateKeyingTracker(nodeID int, combined *ami.CombinedNodeStatus) {
	ps.stateManager.mu.Lock()
	tracker, exists := ps.stateManager.keyingTrackers[nodeID]
	ps.stateManager.mu.Unlock()

	if !exists {
		return
	}

	// Update each connection in the tracker with enriched data
	for _, conn := range combined.Connections {
		// Parse elapsed time to get connected since
		connectedSince := parseElapsedToConnectedSince(conn.Elapsed)

		// Update tracker with enriched info
		tracker.UpdateLinkInfo(conn.Node, conn.Mode, conn.Direction, conn.IP)
		if !connectedSince.IsZero() {
			tracker.UpdateConnectedSince(conn.Node, connectedSince)
		}
	}
}

// parseElapsedToConnectedSince converts HH:MM:SS elapsed time to a timestamp
func parseElapsedToConnectedSince(elapsed string) time.Time {
	if elapsed == "" {
		return time.Time{}
	}

	// Parse HH:MM:SS format
	var hours, minutes, seconds int
	n, err := fmt.Sscanf(elapsed, "%d:%d:%d", &hours, &minutes, &seconds)
	if err != nil || n != 3 {
		return time.Time{}
	}

	// Calculate connected since by subtracting elapsed from now
	totalSeconds := hours*3600 + minutes*60 + seconds
	return time.Now().Add(-time.Duration(totalSeconds) * time.Second)
}

// TriggerPollOnce requests an immediate poll across all configured nodes.
// This is safe to call from other goroutines and is intended for on-demand
// refreshes (for example, shortly after a new client connects) without
// altering the regular polling loop.
func (ps *PollingService) TriggerPollOnce() {
	ps.mu.Lock()
	nodes := make([]int, len(ps.nodes))
	copy(nodes, ps.nodes)
	ps.mu.Unlock()

	for _, nodeID := range nodes {
		// call performPoll in its own goroutine so a slow node doesn't block others
		go ps.performPoll(nodeID)
	}
}

// TriggerPollNode requests an immediate poll for a specific node.
// The poll runs in its own goroutine.
func (ps *PollingService) TriggerPollNode(nodeID int) {
	if nodeID <= 0 {
		return
	}
	go ps.performPoll(nodeID)
}
