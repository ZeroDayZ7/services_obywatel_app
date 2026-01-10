package model

import (
	"time"

	"github.com/google/uuid"
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

// CitizenProfile - Główny kontener profilu
type CitizenProfile struct {
	ID            uint      `gorm:"primaryKey"`
	UserID        uuid.UUID `gorm:"type:uuid;uniqueIndex;not null"`
	EncryptedData []byte    `gorm:"type:bytea;not null"` // Tu siedzi CitizenData
	PeselHash     string    `gorm:"size:64;uniqueIndex"`

	// Relacja: Jeden obywatel może mieć wiele dokumentów
	Documents []UserDocument `gorm:"foreignKey:ProfileID" json:"documents"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

// UserDocument - Zaszyfrowane metadane i pliki
type UserDocument struct {
	ID        uint `gorm:"primaryKey"`
	ProfileID uint `gorm:"index;not null"` // Klucz obcy do CitizenProfile

	// TYPY zostawiamy jawne dla indeksowania (np. szukanie wszystkich paszportów)
	Type   DocumentType   `gorm:"type:varchar(50);index" json:"type"`
	Status DocumentStatus `gorm:"type:varchar(20);default:'active'" json:"status"`

	// SZYFROWANE BLOBY
	// Numer dokumentu, daty i JSON siedzą w EncryptedMeta
	EncryptedMeta []byte `gorm:"type:bytea;not null"`

	// Same pliki są szyfrowane osobno (bo są duże)
	EncryptedFront []byte `gorm:"type:bytea" json:"-"`
	EncryptedBack  []byte `gorm:"type:bytea" json:"-"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// Struktura pomocnicza do meta-danych dokumentu (odszyfrowana)
type DocumentMeta struct {
	DocumentNumber string         `json:"document_number"`
	IssuedAt       *time.Time     `json:"issued_at"`
	ExpiresAt      *time.Time     `json:"expires_at"`
	Data           datatypes.JSON `json:"data"`
}

type CitizenData struct {
	FirstName string         `json:"first_name"`
	LastName  string         `json:"last_name"`
	PESEL     string         `json:"pesel"`
	Data      datatypes.JSON `json:"data"`
}
