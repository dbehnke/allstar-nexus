package models

import "time"

type User struct {
	ID           int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Email        string    `gorm:"unique;not null;size:255" json:"email"`
	PasswordHash string    `gorm:"not null" json:"-"`
	Role         string    `gorm:"not null;default:user;size:50" json:"role"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// TableName overrides the default table name
func (User) TableName() string {
	return "users"
}
