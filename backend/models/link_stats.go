package models

import "time"

// LinkStat represents per-node transmission statistics
type LinkStat struct {
	Node           int        `gorm:"primaryKey" json:"node"`
	TotalTxSeconds int        `gorm:"not null;default:0" json:"total_tx_seconds"`
	LastTxStart    *time.Time `gorm:"type:timestamp" json:"last_tx_start"`
	LastTxEnd      *time.Time `gorm:"type:timestamp" json:"last_tx_end"`
	ConnectedSince *time.Time `gorm:"type:timestamp" json:"connected_since"`
	UpdatedAt      time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName overrides the default table name
func (LinkStat) TableName() string {
	return "link_stats"
}
