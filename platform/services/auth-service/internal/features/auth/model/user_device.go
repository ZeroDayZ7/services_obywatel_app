package model

import (
	"time"
)

type UserDevice struct {
	ID uint `gorm:"primaryKey;autoIncrement"`
	// Dodajemy ten sam tag uniqueIndex dla obu kolumn:
	UserID            uint   `gorm:"not null;uniqueIndex:idx_user_device"`
	DeviceFingerprint string `gorm:"size:128;not null;uniqueIndex:idx_user_device"`

	PublicKey           string    `gorm:"size:512;not null"`
	DeviceNameEncrypted string    `gorm:"size:256"`
	Platform            string    `gorm:"size:30"`
	LastUsedAt          time.Time `gorm:"autoUpdateTime"`
	CreatedAt           time.Time `gorm:"autoCreateTime"`
	IsActive            bool      `gorm:"default:true"`
	IsVerified          bool      `gorm:"default:false"`
	LastIp              string    `gorm:"size:45"`
}
