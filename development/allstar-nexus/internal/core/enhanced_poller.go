package core

import (
	"context"
	"time"

	"github.com/dbehnke/allstar-nexus/internal/ami"
	"go.uber.org/zap"
)

// EnhancedPoller polls XStat and SawStat to provide detailed connection info
type EnhancedPoller struct {
	Conn     *ami.Connector
	State    *StateManager
	Node     int           // Local node number to query
	Interval time.Duration // Poll interval
	Logger   *zap.Logger
	stop     chan struct{}
}

func NewEnhancedPoller(conn *ami.Connector, sm *StateManager, node int, interval time.Duration, logger *zap.Logger) *EnhancedPoller {
	if interval <= 0 {
		interval = 5 * time.Second // More frequent than legacy poller for real-time updates
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &EnhancedPoller{
		Conn:     conn,
		State:    sm,
		Node:     node,
		Interval: interval,
		Logger:   logger,
		stop:     make(chan struct{}),
	}
}

func (ep *EnhancedPoller) Start(ctx context.Context) {
	ticker := time.NewTicker(ep.Interval)
	go func() {
		defer ticker.Stop()
		// Poll immediately on start
		ep.poll(ctx)
		for {
			select {
			case <-ticker.C:
				ep.poll(ctx)
			case <-ctx.Done():
				return
			case <-ep.stop:
				return
			}
		}
	}()
}

func (ep *EnhancedPoller) Stop() {
	close(ep.stop)
}

func (ep *EnhancedPoller) poll(ctx context.Context) {
	pollCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	// Get combined XStat + SawStat data
	combined, err := ep.Conn.GetCombinedStatus(pollCtx, ep.Node)
	if err != nil {
		ep.Logger.Debug("failed to get combined status", zap.Error(err))
		return
	}

	// Apply the combined status to StateManager
	ep.State.ApplyCombinedStatus(combined)
}
