package models

import "time"

// XPActivityLog tracks XP awards with all multipliers for transparency and anti-cheating
type XPActivityLog struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	Callsign         string    `gorm:"index;size:20;not null" json:"callsign"`
	HourBucket       time.Time `gorm:"index;not null" json:"hour_bucket"`     // Truncated to hour for efficient queries
	RawXP            int       `gorm:"not null" json:"raw_xp"`                // Before multipliers
	AwardedXP        int       `gorm:"not null" json:"awarded_xp"`            // After all multipliers and caps
	RestedMultiplier float64   `gorm:"default:1.0" json:"rested_multiplier"`  // Rested bonus multiplier
	DRMultiplier     float64   `gorm:"default:1.0" json:"dr_multiplier"`      // Diminishing returns multiplier
	KerchunkPenalty  float64   `gorm:"default:1.0" json:"kerchunk_penalty"`   // Kerchunk detection penalty
	CreatedAt        time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (XPActivityLog) TableName() string {
	return "xp_activity_logs"
}
