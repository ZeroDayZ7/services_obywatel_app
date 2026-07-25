// model/user_session_dto.go
package model

import (
	"time"

	"github.com/google/uuid"
)

type UserSessionDTO struct {
	SessionID           uuid.UUID `gorm:"column:session_id" json:"id"`
	DeviceNameEncrypted string    `gorm:"column:device_name_encrypted" json:"device_name"`
	Platform            string    `gorm:"column:platform" json:"platform"`
	CreatedAt           time.Time `gorm:"column:created_at" json:"created_at"`
	LastUsedAt          time.Time `gorm:"column:last_used_at" json:"last_used_at"`
	Fingerprint         string    `gorm:"column:fingerprint" json:"fingerprint"`
	IsCurrent           bool      `json:"is_current"`
}
