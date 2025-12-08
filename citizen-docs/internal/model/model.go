package model

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// UserDocument przechowuje dokument użytkownika wraz z plikami jako bytea (Base64)
type UserDocument struct {
	ID             string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID         string         `gorm:"type:uuid;index;not null" json:"user_id"`
	Type           string         `gorm:"type:varchar(50);index;not null" json:"type"`
	DocumentNumber string         `gorm:"type:varchar(100)" json:"document_number"`
	IssuedAt       *time.Time     `json:"issued_at"`
	ExpiresAt      *time.Time     `json:"expires_at"`
	Status         string         `gorm:"type:varchar(30);default:'active'" json:"status"`
	Data           datatypes.JSON `gorm:"type:jsonb" json:"data"`

	// pliki dokumentu w Base64
	FileFront []byte `gorm:"type:bytea" json:"file_front"`
	FileBack  []byte `gorm:"type:bytea" json:"file_back"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
