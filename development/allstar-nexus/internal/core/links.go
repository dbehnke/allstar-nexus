package core

import "time"

type LinkInfo struct {
	Node           int        `json:"node"`
	ConnectedSince time.Time  `json:"connected_since"`
	LastTxStart    *time.Time `json:"last_tx_start,omitempty"`
	LastTxEnd      *time.Time `json:"last_tx_end,omitempty"`
	LastHeardAt    *time.Time `json:"last_heard_at,omitempty"`
	CurrentTx      bool       `json:"current_tx"`
	TotalTxSeconds int        `json:"total_tx_seconds"`

	// Enhanced AMI fields from XStat/SawStat
	IP              string     `json:"ip,omitempty"`              // IP address (empty for EchoLink)
	IsKeyed         bool       `json:"is_keyed"`                  // Remote node is currently keying
	Direction       string     `json:"direction,omitempty"`       // "IN" or "OUT"
	Elapsed         string     `json:"elapsed,omitempty"`         // Connection elapsed time
	LinkType        string     `json:"link_type,omitempty"`       // "ESTABLISHED", "CONNECTING", etc.
	Mode            string     `json:"mode,omitempty"`            // T=Transceive, R=Receive, C=Connecting, M=Monitor
	LastHeard       string     `json:"last_heard,omitempty"`      // Human-readable last heard time
	SecsSinceKeyed  int        `json:"secs_since_keyed"`          // Seconds since last keyed
	LastKeyedTime   *time.Time `json:"last_keyed_time,omitempty"` // Timestamp of last key
}

func (li *LinkInfo) UpdateTx(active bool, now time.Time) {
	if active && !li.CurrentTx {
		ts := now
		li.LastTxStart = &ts
		li.CurrentTx = true
	} else if !active && li.CurrentTx {
		li.CurrentTx = false
		te := now
		li.LastTxEnd = &te
		if li.LastTxStart != nil {
			li.TotalTxSeconds += int(now.Sub(*li.LastTxStart).Seconds())
		}
	}
}
