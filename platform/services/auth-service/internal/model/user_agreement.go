// model/user_agreement.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AgreementStatus string

const (
	AgreementStatusPending AgreementStatus = "PENDING" // Oczekuje na aktywację / podpis
	AgreementStatusActive  AgreementStatus = "ACTIVE"  // Aktywna
	AgreementStatusRevoked AgreementStatus = "REVOKED" // Unieważniona
	AgreementStatusExpired AgreementStatus = "EXPIRED" // Wygasła
)

type PukStatus string

const (
	PukStatusActive  PukStatus = "ACTIVE"  // Gotowy do użycia przy resecie / aktywacji
	PukStatusUsed    PukStatus = "USED"    // Wykorzystany
	PukStatusBlocked PukStatus = "BLOCKED" // Zablokowany po zbyt wielu błędnych próbach
)

// region UserAgreement
type UserAgreement struct {
	ID              uuid.UUID       `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	UserID          uuid.UUID       `gorm:"type:uuid;not null;uniqueIndex"`
	AgreementNumber string          `gorm:"size:64;not null;uniqueIndex"`
	PeselEncrypted  string          `gorm:"size:256;not null;index"` // PESEL zaszyfrowany AES-GCM
	VerifiedPhone   string          `gorm:"size:20;not null"`        // Numer telefonu do kodów SMS z umowy
	Status          AgreementStatus `gorm:"type:varchar(20);not null;default:'PENDING'"`
	SignedAt        time.Time       `gorm:"not null"`
	VerifiedAt      *time.Time
	VerifiedVia     string         `gorm:"size:50;not null"` // np. "BRANCH", "MOJE_ID", "COURIER"
	CreatedAt       time.Time      `gorm:"autoCreateTime"`
	UpdatedAt       time.Time      `gorm:"autoUpdateTime"`
	DeletedAt       gorm.DeletedAt `gorm:"index"`

	// Relacja
	PukCode *UserPukCode `gorm:"foreignKey:UserAgreementID"`
}

// region UserPukCode
type UserPukCode struct {
	ID              uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	UserAgreementID uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex"`
	UserID          uuid.UUID  `gorm:"type:uuid;not null;index"`
	PukHash         string     `gorm:"size:128;not null"` // Hash kodu PUK (np. bcrypt/Argon2id)
	Status          PukStatus  `gorm:"type:varchar(20);not null;default:'ACTIVE'"`
	FailedAttempts  int8       `gorm:"not null;default:0"`
	MaxAttempts     int8       `gorm:"not null;default:3"`
	ExpiresAt       *time.Time `gorm:"index"` // Opcjonalnie: PUK ważny np. 30 dni na pierwszą aktywację
	UsedAt          *time.Time
	CreatedAt       time.Time      `gorm:"autoCreateTime"`
	UpdatedAt       time.Time      `gorm:"autoUpdateTime"`
	DeletedAt       gorm.DeletedAt `gorm:"index"`
}
