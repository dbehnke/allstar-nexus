package core

import "time"

// LinkTxEvent represents a per-link transmit edge (start or stop).
type LinkTxEvent struct {
	Node           int        `json:"node"`
	Kind           string     `json:"kind"` // START or STOP
	At             time.Time  `json:"at"`
	TotalTxSeconds int        `json:"total_tx_seconds"`
	LastTxStart    *time.Time `json:"last_tx_start,omitempty"`
	LastTxEnd      *time.Time `json:"last_tx_end,omitempty"`
}
