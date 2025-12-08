package model

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// DocumentType definiuje typ dokumentu
type DocumentType string

const (
	DocumentTypePassport      DocumentType = "passport"
	DocumentTypeIDCard        DocumentType = "id_card"
	DocumentTypeDriverLicense DocumentType = "driver_license"
	DocumentTypeOther         DocumentType = "other"
)

// DocumentStatus definiuje status dokumentu
type DocumentStatus string

const (
	DocumentStatusActive   DocumentStatus = "active"
	DocumentStatusInactive DocumentStatus = "inactive"
	DocumentStatusExpired  DocumentStatus = "expired"
	DocumentStatusRevoked  DocumentStatus = "revoked"
)

// UserDocument przechowuje dokument użytkownika wraz z plikami w Base64
type UserDocument struct {
	ID             uint           `gorm:"primaryKey;autoIncrement;type:int" json:"id"`
	UserID         uint           `gorm:"type:int;index;not null" json:"user_id"`
	Type           DocumentType   `gorm:"type:document_type;not null;index" json:"type"`
	DocumentNumber string         `gorm:"type:varchar(100)" json:"document_number"`
	IssuedAt       *time.Time     `json:"issued_at"`
	ExpiresAt      *time.Time     `json:"expires_at"`
	Status         DocumentStatus `gorm:"type:document_status;default:'active'" json:"status"`
	Data           datatypes.JSON `gorm:"type:jsonb" json:"data"`

	// Pliki dokumentu w Base64
	FileFront []byte `gorm:"type:bytea" json:"file_front"`
	FileBack  []byte `gorm:"type:bytea" json:"file_back"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
