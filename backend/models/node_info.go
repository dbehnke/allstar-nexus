package models

import (
	"time"
)

// NodeInfo represents AllStar node information from astdb
type NodeInfo struct {
	NodeID      int       `gorm:"primaryKey;column:node_id;index:idx_node_id" json:"node_id"`
	Callsign    string    `gorm:"column:callsign;size:20;index:idx_callsign" json:"callsign"`
	Description string    `gorm:"column:description;size:255" json:"description"`
	Location    string    `gorm:"column:location;size:255;index:idx_location" json:"location"`
	LastSeen    time.Time `gorm:"column:last_seen;index:idx_last_seen" json:"last_seen"` // Track when node was last in astdb
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

// TableName overrides the table name
func (NodeInfo) TableName() string {
	return "node_info"
}
