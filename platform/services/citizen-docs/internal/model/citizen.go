package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// BaseModel contains common audit timestamps for all database entities.
type BaseModel struct {
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// DocumentStatus represents the lifecycle state of a citizen document.
type DocumentStatus string

const (
	DocumentStatusActive  DocumentStatus = "ACTIVE"
	DocumentStatusPending DocumentStatus = "PENDING"
	DocumentStatusExpired DocumentStatus = "EXPIRED"
	DocumentStatusRevoked DocumentStatus = "REVOKED"
)

// CitizenDocument stores only the document metadata for a user.
type CitizenDocument struct {
	ID                 uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID             uuid.UUID      `gorm:"type:uuid;not null;index" json:"user_id"`
	DocumentType       string         `gorm:"type:varchar(64);not null;index" json:"document_type"`
	DocumentNumber     string         `gorm:"type:varchar(128);not null;index" json:"document_number"`
	DocumentNumberHash string         `gorm:"type:varchar(64);not null;index;unique" json:"document_number_hash"`
	Status             DocumentStatus `gorm:"type:varchar(32);not null;default:'PENDING';index" json:"status"`
	Metadata           datatypes.JSON `gorm:"-" json:"metadata,omitempty"`
	EncryptedMetadata  []byte         `gorm:"column:encrypted_metadata;type:bytea;not null" json:"-"`
	EncryptedDEK       []byte         `gorm:"column:encrypted_dek;type:bytea;not null" json:"-"`
	IssuedAt           *time.Time     `gorm:"index" json:"issued_at,omitempty"`
	ExpiresAt          *time.Time     `gorm:"index" json:"expires_at,omitempty"`

	BaseModel
}

// CreateDocumentPayload is the request contract used when creating a citizen document.
type CreateDocumentPayload struct {
	UserID         uuid.UUID      `json:"user_id"`
	DocumentType   string         `json:"document_type"`
	DocumentNumber string         `json:"document_number,omitempty"`
	Status         DocumentStatus `json:"status,omitempty"`
	Metadata       datatypes.JSON `json:"metadata,omitempty"`
	IssuedAt       *time.Time     `json:"issued_at,omitempty"`
	ExpiresAt      *time.Time     `json:"expires_at,omitempty"`
}
