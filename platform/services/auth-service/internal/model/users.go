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

type UserStatus string

const (
	StatusActive    UserStatus = "ACTIVE"    // aktywny – użytkownik może normalnie korzystać z systemu
	StatusSuspended UserStatus = "SUSPENDED" // zawieszony – konto tymczasowo zablokowane
	StatusPending   UserStatus = "PENDING"   // oczekujący – np. konto czeka na weryfikację
	StatusBanned    UserStatus = "BANNED"    // zbanowany – konto trwale zablokowane
	StatusLocked    UserStatus = "LOCKED"
)

type Permission string

type UserPermission struct {
	ID         uuid.UUID  `gorm:"type:uuid;primaryKey"`
	UserID     uuid.UUID  `gorm:"type:uuid;index"`
	Permission Permission `gorm:"type:varchar(50);not null"`
	Scope      string     `gorm:"type:text"`
	CreatedAt  time.Time
}

type User struct {
	ID                  uuid.UUID  `gorm:"type:uuid;primaryKey"`
	Username            string     `gorm:"size:30;not null;unique"`
	Email               string     `gorm:"size:100;not null;unique"`
	Password            string     `gorm:"size:128;not null"`
	Role                UserRole   `gorm:"type:varchar(20);not null;default:'user'"`
	Status              UserStatus `gorm:"type:varchar(20);not null;default:'ACTIVE'"`
	FailedLoginAttempts int8       `gorm:"not null;default:0"`
	LockedUntil         *time.Time `gorm:"index"`
	LastLogin           time.Time
	PasswordChangedAt   *time.Time
	LastIP              string           `gorm:"size:45"`
	TwoFactorEnabled    bool             `gorm:"not null;default:false"`
	TwoFactorSecret     string           `gorm:"size:64"`
	Permissions         []UserPermission `gorm:"foreignKey:UserID"`
	CreatedAt           time.Time        `gorm:"autoCreateTime"`
	UpdatedAt           time.Time        `gorm:"autoUpdateTime"`
	DeletedAt           gorm.DeletedAt   `gorm:"index"`
}

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	idStr := shared.GenerateUuidV7()
	u.ID, err = uuid.Parse(idStr)
	return err
}
