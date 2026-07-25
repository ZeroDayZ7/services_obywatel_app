// model/user_session_dto.go
package model

import (
	"time"

	"github.com/google/uuid"
)

type UserSessionDTO struct {
	SessionID           uuid.UUID `gorm:"column:session_id" json:"id"`
	DeviceNameEncrypted string    `gorm:"column:device_name_encrypted" json:"device_name"`
	Platform            string    `json:"platform"`
	CreatedAt           time.Time `json:"created_at"`
	LastUsedAt          time.Time `json:"last_used_at"`
	Fingerprint         string    `gorm:"column:fingerprint" json:"-"`
	IsCurrent           bool      `json:"is_current"`
}
