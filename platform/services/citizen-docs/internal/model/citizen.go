package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type DocumentType string

const (
	DocumentTypeIDCard          DocumentType = "id_card"
	DocumentTypePassport        DocumentType = "passport"
	DocumentTypeDriverLicense   DocumentType = "driver_license"
	DocumentTypeStudentCard     DocumentType = "student_card"
	DocumentTypeLargeFamilyCard DocumentType = "large_family"
	DocumentTypeDisabilityCard  DocumentType = "disability_card"
	DocumentTypeOther           DocumentType = "other"
)

type DocumentStatus string

const (
	DocumentStatusActive   DocumentStatus = "active"
	DocumentStatusInactive DocumentStatus = "inactive"
	DocumentStatusExpired  DocumentStatus = "expired"
	DocumentStatusRevoked  DocumentStatus = "revoked"
	DocumentStatusPending  DocumentStatus = "pending"
)

// CitizenProfile – Główny profil w osobnej bazie mikroserwisu dokumentów
type CitizenProfile struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	UserID        uuid.UUID `gorm:"type:uuid;uniqueIndex;not null"` // Identyfikator logiczny z Auth/User Service
	EncryptedData []byte    `gorm:"type:bytea;not null"`
	PeselHash     string    `gorm:"size:64;uniqueIndex;not null"`

	// Relacja wewnątrz TEJ SAMEJ bazy (w obrębie tego samego mikroserwisu)
	Documents []UserDocument `gorm:"foreignKey:ProfileID" json:"documents,omitempty"`

	CreatedAt time.Time      `gorm:"autoCreateTime"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// UserDocument – Dokumenty tożsamości obywatela
type UserDocument struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	ProfileID uuid.UUID `gorm:"type:uuid;index;not null"`

	Type   DocumentType   `gorm:"type:varchar(50);not null;index"`
	Status DocumentStatus `gorm:"type:varchar(20);not null;default:'active';index"`

	EncryptedMeta  []byte `gorm:"type:bytea;not null"`
	EncryptedFront []byte `gorm:"type:bytea" json:"-"`
	EncryptedBack  []byte `gorm:"type:bytea" json:"-"`

	IssuedAt  *time.Time `gorm:"index"`
	ExpiresAt *time.Time `gorm:"index"`

	CreatedAt time.Time      `gorm:"autoCreateTime"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index"`

	// Relacja wewnątrz bazy mikroserwisu – usunięcie profilu usuwa jego dokumenty
	Profile *CitizenProfile `gorm:"foreignKey:ProfileID;constraint:OnDelete:CASCADE"`
}

type DocumentMeta struct {
	DocumentNumber string         `json:"document_number"`
	Issuer         string         `json:"issuer,omitempty"`
	AdditionalInfo datatypes.JSON `json:"additional_info,omitempty"`
}

type CitizenData struct {
	FirstName   string         `json:"first_name"`
	LastName    string         `json:"last_name"`
	PESEL       string         `json:"pesel"`
	DateOfBirth string         `json:"date_of_birth"`
	Citizenship string         `json:"citizenship"`
	Attributes  datatypes.JSON `json:"attributes,omitempty"`
}
