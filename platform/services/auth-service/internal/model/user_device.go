package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserDevice struct {
	ID                  uuid.UUID      `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	UserID              uuid.UUID      `gorm:"type:uuid;not null;uniqueIndex:idx_user_device"`
	DeviceFingerprint   string         `gorm:"size:128;not null;uniqueIndex:idx_user_device"`
	PublicKey           string         `gorm:"type:text;not null"`
	DeviceNameEncrypted string         `gorm:"size:256"`
	Platform            string         `gorm:"size:30"`
	IsActive            bool           `gorm:"not null;default:true"`
	IsVerified          bool           `gorm:"not null;default:false"`
	LastIP              string         `gorm:"size:45"`
	LastUsedAt          time.Time      `gorm:"autoUpdateTime"`
	CreatedAt           time.Time      `gorm:"autoCreateTime"`
	DeletedAt           gorm.DeletedAt `gorm:"index"`

	// Relacja
	RefreshTokens []RefreshToken `gorm:"foreignKey:DeviceID"`
}
