package models

import (
	"time"
)

// TransmissionLog records each transmission event on the network
type TransmissionLog struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	SourceID        int       `gorm:"index;not null" json:"source_id"`                                                                 // Local source node ID
	AdjacentLinkID  int       `gorm:"index;not null" json:"adjacent_link_id"`                                                          // Remote/adjacent node ID that transmitted
	Callsign        string    `gorm:"index:idx_callsign_start,priority:1;index;size:20" json:"callsign"`                               // Callsign of the transmitting node
	TimestampStart  time.Time `gorm:"index;not null" json:"timestamp_start"`                                                           // UTC timestamp when TX started
	TimestampEnd    time.Time `gorm:"index;not null" json:"timestamp_end"`                                                             // UTC timestamp when TX ended
	StartUnix       int64     `gorm:"index:idx_callsign_start,priority:2;index:idx_start_callsign,priority:1;index" json:"start_unix"` // Unix seconds for start (index-friendly)
	EndUnix         int64     `gorm:"index" json:"end_unix"`                                                                           // Unix seconds for end (index-friendly)
	DurationSeconds int       `gorm:"not null" json:"duration_seconds"`                                                                // Duration in seconds
	CreatedAt       time.Time `gorm:"autoCreateTime" json:"created_at"`                                                                // Record creation timestamp
}

// TableName overrides the default table name
func (TransmissionLog) TableName() string {
	return "transmission_logs"
}
