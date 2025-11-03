package models

import (
	"time"
)

// NodeGroup represents a named collection of nodes for bulk operations
type NodeGroup struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"uniqueIndex;not null" json:"name"`
	Description string    `json:"description"`
	NodeNumbers []int     `gorm:"serializer:json" json:"node_numbers"` // JSON array of node numbers
	CreatedBy   string    `gorm:"not null" json:"created_by"`          // Email of creator
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// NodeGroupConfig represents saved link configuration for a group
type NodeGroupConfig struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	GroupID     uint   `gorm:"not null;index" json:"group_id"`
	NodeGroup   *NodeGroup `gorm:"foreignKey:GroupID;constraint:OnDelete:CASCADE" json:"group,omitempty"`
	Name        string `gorm:"not null" json:"name"`
	Description string `json:"description"`
	// Link configuration (e.g., what nodes to link to, parameters)
	LinkConfig  map[string]interface{} `gorm:"serializer:json" json:"link_config"`
	Permanent   bool                   `json:"permanent"` // Whether links should be permanent
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// TableName overrides the table name for NodeGroup
func (NodeGroup) TableName() string {
	return "node_groups"
}

// TableName overrides the table name for NodeGroupConfig
func (NodeGroupConfig) TableName() string {
	return "node_group_configs"
}
