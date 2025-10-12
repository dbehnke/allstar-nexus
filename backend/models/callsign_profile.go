package models

import "time"

// CallsignProfile tracks experience, level, and renown for each callsign
type CallsignProfile struct {
	ID                 uint      `gorm:"primaryKey" json:"id"`
	Callsign           string    `gorm:"uniqueIndex;size:20;not null" json:"callsign"`
	Level              int       `gorm:"default:1;index" json:"level"`
	ExperiencePoints   int       `gorm:"default:0" json:"experience_points"`
	RenownLevel        int       `gorm:"default:0;index" json:"renown_level"`
	LastTallyAt        time.Time `gorm:"index" json:"last_tally_at"`
	LastTransmissionAt time.Time `gorm:"index" json:"last_transmission_at"`
	RestedBonusSeconds int       `gorm:"default:0" json:"rested_bonus_seconds"`
	DailyXP            int       `gorm:"default:0" json:"daily_xp"`
	WeeklyXP           int       `gorm:"default:0" json:"weekly_xp"`
	CreatedAt          time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (CallsignProfile) TableName() string {
	return "callsign_profiles"
}
