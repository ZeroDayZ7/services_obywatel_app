package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// region EmployeeCredential
type EmployeeCredentialStatus string

const (
	EmployeeCredentialActive  EmployeeCredentialStatus = "ACTIVE"
	EmployeeCredentialRevoked EmployeeCredentialStatus = "REVOKED"
	EmployeeCredentialExpired EmployeeCredentialStatus = "EXPIRED"
)

type EmployeeCredential struct {
	ID               uuid.UUID                `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	UserID           uuid.UUID                `gorm:"type:uuid;not null;index"`
	CardSerialNumber string                   `gorm:"size:128;not null;uniqueIndex"`
	PublicKey        string                   `gorm:"type:text;not null"`
	KeyAlgorithm     string                   `gorm:"size:30;not null;default:'ED25519'"`
	Status           EmployeeCredentialStatus `gorm:"size:20;not null;default:'ACTIVE'"`
	IssuedBy         uuid.UUID                `gorm:"type:uuid;not null"`
	ExpiresAt        *time.Time               `gorm:"index"`
	LastUsedAt       *time.Time
	CreatedAt        time.Time      `gorm:"autoCreateTime"`
	UpdatedAt        time.Time      `gorm:"autoUpdateTime"`
	DeletedAt        gorm.DeletedAt `gorm:"index"`
}

// endregion
