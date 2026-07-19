package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RefreshToken struct {
	ID                uuid.UUID      `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	UserID            uuid.UUID      `gorm:"type:uuid;not null;index"`
	DeviceID          *uuid.UUID     `gorm:"type:uuid;index"`
	Token             string         `gorm:"size:64;not null;uniqueIndex"`
	DeviceFingerprint string         `gorm:"size:128;not null"`
	ExpiresAt         time.Time      `gorm:"not null;index"`
	CreatedAt         time.Time      `gorm:"autoCreateTime"`
	UpdatedAt         time.Time      `gorm:"autoUpdateTime"`
	DeletedAt         gorm.DeletedAt `gorm:"index"`
	Revoked           bool           `gorm:"default:false;index"`
}
