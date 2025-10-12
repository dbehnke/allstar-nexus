package models

import "time"

// TallyState stores global state for the gamification tally service
// A singleton row (ID=1) is used to persist the last tally time across restarts.
type TallyState struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	LastTallyAt time.Time `gorm:"index" json:"last_tally_at"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (TallyState) TableName() string {
	return "tally_state"
}
