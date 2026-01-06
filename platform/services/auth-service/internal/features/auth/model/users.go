package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/pkg/shared"
	"gorm.io/gorm"
)

type UserRole string

const (
	RoleUser  UserRole = "user"
	RoleClerk UserRole = "clerk"
	RoleAdmin UserRole = "admin"
)

type User struct {
	// Użycie UUID zamiast uint zwiększa bezpieczeństwo publicznych identyfikatorów
	ID       uuid.UUID `gorm:"type:uuid;primaryKey"`
	Username string    `gorm:"size:30;not null;unique"`
	Email    string    `gorm:"size:100;not null;unique"`
	Password string    `gorm:"size:128;not null"`

	// RBAC: Kluczowe pole dla Twojego systemu uprawnień
	Role UserRole `gorm:"type:varchar(20);not null;default:'user'"`

	TwoFactorEnabled bool   `gorm:"not null;default:false"`
	TwoFactorSecret  string `gorm:"size:64"`

	CreatedAt time.Time      `gorm:"autoCreateTime"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// TO JEST MIEJSCE, GDZIE DZIEJE SIĘ MAGIA:
func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	// Tutaj wywołujemy Twoją funkcję z shared!
	idStr := shared.GenerateUuidV7()

	// GORM potrzebuje obiektu typu uuid.UUID, więc parsujemy stringa
	u.ID, err = uuid.Parse(idStr)
	return err
}
