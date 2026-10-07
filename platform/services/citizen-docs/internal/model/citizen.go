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

// CitizenDocument stores the encrypted document metadata for a user.
type CitizenDocument struct {
	ID                 uuid.UUID      `gorm:"type:uuid;primaryKey;default:uuidv7()" json:"id"`
	UserID             uuid.UUID      `gorm:"type:uuid;not null;index" json:"user_id"`
	DocumentType       string         `gorm:"type:varchar(64);not null;index" json:"document_type"`
	Status             DocumentStatus `gorm:"type:varchar(32);not null;default:'PENDING';index" json:"status"`
	IssuedAt           *time.Time     `gorm:"index" json:"issued_at,omitempty"`
	ExpiresAt          *time.Time     `gorm:"index" json:"expires_at,omitempty"`
	DocumentNumberHash string         `gorm:"type:varchar(64);not null;index;unique" json:"document_number_hash"`
	Version            uint64         `gorm:"not null;default:1;index" json:"version,omitempty"`
	EncryptedMetadata  []byte         `gorm:"column:encrypted_metadata;type:bytea;not null" json:"-"`
	EncryptedDEK       []byte         `gorm:"column:encrypted_dek;type:bytea;not null" json:"-"`
	BaseModel
}

// UserDocumentState stores the aggregated document snapshot for a user's delta sync cursor.
type UserDocumentState struct {
	UserID        uuid.UUID      `gorm:"type:uuid;primaryKey" json:"user_id"`
	StateVersion  uint64         `gorm:"not null;default:0;index" json:"state_version"`
	AggregateHash string         `gorm:"type:varchar(128);not null;default:'';index" json:"aggregate_hash"`
	DocumentCount int            `gorm:"not null;default:0" json:"document_count"`
	CreatedAt     time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
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
