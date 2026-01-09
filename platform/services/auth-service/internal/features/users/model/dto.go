// model/user_session_dto.go
package model

import (
	"time"
)

type UserSessionDTO struct {
	SessionID           uint       `json:"id" gorm:"column:session_id"`
	DeviceNameEncrypted string     `json:"device_name_encrypted" gorm:"column:device_name_encrypted"`
	Platform            string     `json:"platform" gorm:"column:platform"`
	CreatedAt           time.Time  `json:"created_at" gorm:"column:created_at"`
	LastUsedAt          *time.Time `json:"last_activity" gorm:"column:last_used_at"`
	Fingerprint         string     `json:"fingerprint" gorm:"column:fingerprint"`
}
