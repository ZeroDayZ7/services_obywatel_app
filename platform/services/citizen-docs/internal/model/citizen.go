package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

//#region BaseModel
type BaseModel struct {
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

//#endregion

//#region DocumentStatus
type DocumentStatus string

const (
	DocumentStatusActive    DocumentStatus = "active"
	DocumentStatusInactive  DocumentStatus = "inactive"
	DocumentStatusExpired   DocumentStatus = "expired"
	DocumentStatusRevoked   DocumentStatus = "revoked"
	DocumentStatusPending   DocumentStatus = "pending"
	DocumentStatusSuspended DocumentStatus = "suspended"
)

//#endregion

//#region CitizenProfile
type CitizenProfile struct {
	ID            uuid.UUID      `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	UserID        uuid.UUID      `gorm:"type:uuid;uniqueIndex;not null"`
	EncryptedData []byte         `gorm:"type:bytea;not null"`
	EncryptedDEK  []byte         `gorm:"type:bytea;not null"`
	PeselHash     string         `gorm:"size:64;uniqueIndex;not null"`
	Version       uint64         `gorm:"not null;default:1;index"`
	Documents     []UserDocument `gorm:"foreignKey:ProfileID" json:"documents,omitempty"`

	BaseModel
}

//#endregion

//#region UserDocument
type UserDocument struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	ProfileID uuid.UUID      `gorm:"type:uuid;index;not null"`
	TypeCode  string         `gorm:"type:varchar(64);not null;index"`
	Status    DocumentStatus `gorm:"type:varchar(20);not null;default:'active';index"`

	EncryptedMeta    []byte `gorm:"type:bytea;not null"`
	EncryptedMetaDEK []byte `gorm:"type:bytea;not null"`

	EncryptedFront    []byte `gorm:"type:bytea" json:"-"`
	EncryptedFrontDEK []byte `gorm:"type:bytea" json:"-"`

	EncryptedBack    []byte `gorm:"type:bytea" json:"-"`
	EncryptedBackDEK []byte `gorm:"type:bytea" json:"-"`

	IssuerSignature  []byte `gorm:"type:bytea;not null"`
	SigningKeyID     string `gorm:"type:varchar(64);not null;index"`
	RevocationSerial string `gorm:"type:varchar(128);uniqueIndex;not null"`

	IssuedAt  *time.Time `gorm:"index"`
	ExpiresAt *time.Time `gorm:"index"`

	Version uint64          `gorm:"not null;default:1;index"`
	Profile *CitizenProfile `gorm:"foreignKey:ProfileID;constraint:OnDelete:CASCADE"`

	BaseModel
}

//#endregion

//#region CitizenData
type CitizenData struct {
	FirstName   string         `json:"first_name"`
	LastName    string         `json:"last_name"`
	PESEL       string         `json:"pesel"`
	DateOfBirth string         `json:"date_of_birth"`
	Citizenship string         `json:"citizenship"`
	Attributes  datatypes.JSON `json:"attributes,omitempty"`
}

//#endregion

//#region DocumentMeta
type DocumentMeta struct {
	DocumentNumber   string         `json:"document_number"`
	Title            string         `json:"title,omitempty"`
	Issuer           string         `json:"issuer,omitempty"`
	Category         string         `json:"category,omitempty"`
	AllowedScopes    []string       `json:"allowed_scopes,omitempty"`
	CustomAttributes datatypes.JSON `json:"custom_attributes,omitempty"`
}

//#endregion

//#region DocumentVerificationPayload
type DocumentVerificationPayload struct {
	DocumentID       uuid.UUID      `json:"doc_id"`
	ProfileID        uuid.UUID      `json:"profile_id"`
	RevocationSerial string         `json:"revocation_serial"`
	Scope            string         `json:"scope"`
	Claims           datatypes.JSON `json:"claims"`
	IssuedNonce      int64          `json:"nonce"`
	IssuerSignature  []byte         `json:"issuer_sig"`
	DeviceSignature  []byte         `json:"device_sig"`
}

//#endregion
