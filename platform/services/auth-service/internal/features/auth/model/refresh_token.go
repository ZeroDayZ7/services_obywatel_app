package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/pkg/shared"
	"gorm.io/gorm"
)

type RefreshToken struct {
	// Zmieniamy na UUID v7 dla spójności
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	// UserID musi być UUID, aby pasował do User.ID
	UserID uuid.UUID `gorm:"type:uuid;not null;index"`

	Token             string    `gorm:"size:512;not null;uniqueIndex"`
	DeviceFingerprint string    `gorm:"size:128;not null"`
	ExpiresAt         time.Time `gorm:"not null"`
	CreatedAt         time.Time `gorm:"autoCreateTime"`
	UpdatedAt         time.Time `gorm:"autoUpdateTime"`
	Revoked           bool      `gorm:"default:false"`
}

// Hook do automatycznego generowania UUID v7
func (rt *RefreshToken) BeforeCreate(tx *gorm.DB) (err error) {
	idStr := shared.GenerateUuidV7()
	rt.ID, err = uuid.Parse(idStr)
	return err
}
