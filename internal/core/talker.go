package core

import (
	"sync"
	"time"
)

// TalkerEvent describes a transmit related event with node/callsign info.
type TalkerEvent struct {
	At          time.Time `json:"at"`
	Kind        string    `json:"kind"`     // TX_START / TX_STOP
	Node        int       `json:"node,omitempty"`
	Callsign    string    `json:"callsign,omitempty"`
	Description string    `json:"description,omitempty"`
	Duration    int       `json:"duration,omitempty"` // Duration in seconds (for STOP events)
}

// TalkerLog is a size & time bounded ring buffer.
type TalkerLog struct {
	mu  sync.RWMutex
	buf []TalkerEvent
	max int
	ttl time.Duration
}

func NewTalkerLog(max int, ttl time.Duration) *TalkerLog { return &TalkerLog{max: max, ttl: ttl} }

func (tl *TalkerLog) Add(evt TalkerEvent) {
	tl.mu.Lock()
	tl.pruneLocked()
	tl.buf = append(tl.buf, evt)
	if len(tl.buf) > tl.max {
		tl.buf = tl.buf[len(tl.buf)-tl.max:]
	}
	tl.mu.Unlock()
}

func (tl *TalkerLog) Snapshot() []TalkerEvent {
	tl.mu.RLock()
	defer tl.mu.RUnlock()
	out := make([]TalkerEvent, len(tl.buf))
	copy(out, tl.buf)
	return out
}

func (tl *TalkerLog) pruneLocked() {
	if tl.ttl <= 0 || len(tl.buf) == 0 {
		return
	}
	cutoff := time.Now().Add(-tl.ttl)
	idx := 0
	for i, e := range tl.buf {
		if e.At.After(cutoff) {
			idx = i
			break
		}
	}
	tl.buf = tl.buf[idx:]
}
