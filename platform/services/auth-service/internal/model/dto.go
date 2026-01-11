// model/user_session_dto.go
package model

import (
	"time"
)

type UserSessionDTO struct {
	SessionID           uint      `gorm:"column:session_id" json:"id"`
	DeviceNameEncrypted string    `gorm:"column:device_name_encrypted" json:"device_name"`
	Platform            string    `json:"platform"`
	CreatedAt           time.Time `json:"created_at"`
	LastUsedAt          time.Time `json:"last_used_at"`
	Fingerprint         string    `gorm:"column:fingerprint" json:"-"` // Ukrywamy w JSON, ale potrzebujemy w Go
	IsCurrent           bool      `json:"is_current"`                  // To ustawimy w serwisie
}
