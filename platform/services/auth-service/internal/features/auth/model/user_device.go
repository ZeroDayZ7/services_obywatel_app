package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/zerodayz7/platform/pkg/shared"
)

type UserDevice struct {
	ID                uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID            uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_user_device"`
	DeviceFingerprint string    `gorm:"size:128;not null;uniqueIndex:idx_user_device"`

	PublicKey           string `gorm:"type:text;not null"`
	DeviceNameEncrypted string `gorm:"size:256"`
	Platform            string `gorm:"size:30"`
	IsActive            bool   `gorm:"default:true"`
	IsVerified          bool   `gorm:"default:false"`

	LastUsedAt time.Time      `gorm:"autoUpdateTime"`
	CreatedAt  time.Time      `gorm:"autoCreateTime"`
	DeletedAt  gorm.DeletedAt `gorm:"index"`
	LastIp     string         `gorm:"size:45"`
}

func (ud *UserDevice) BeforeCreate(tx *gorm.DB) (err error) {
	idStr := shared.GenerateUuidV7()
	ud.ID, err = uuid.Parse(idStr)
	return err
}
