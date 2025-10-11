package models

// LevelConfig stores XP requirements for each level
type LevelConfig struct {
	Level              int    `gorm:"primaryKey" json:"level"`
	RequiredExperience int    `gorm:"not null" json:"required_experience"`
	Name               string `gorm:"size:100" json:"name"` // Future: "Newbie", "Veteran", etc.
}

func (LevelConfig) TableName() string {
	return "level_configs"
}
