package model

import (
	"time"
)

type RefreshToken struct {
	ID                uint      `gorm:"primaryKey"`
	UserID            uint      `gorm:"not null;index"`
	Token             string    `gorm:"size:512;not null;uniqueIndex"`
	DeviceFingerprint string    `gorm:"size:128;not null"`
	ExpiresAt         time.Time `gorm:"not null"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
	Revoked           bool `gorm:"default:false"`
}
