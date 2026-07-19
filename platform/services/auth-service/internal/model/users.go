package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type UserRole string

const (
	RoleUser  UserRole = "operator"
	RoleAdmin UserRole = "admin"
	RoleRoot  UserRole = "root"
)

type UserStatus string

const (
	StatusActive    UserStatus = "ACTIVE"
	StatusSuspended UserStatus = "SUSPENDED"
	StatusPending   UserStatus = "PENDING"
	StatusBanned    UserStatus = "BANNED"
	StatusLocked    UserStatus = "LOCKED"
)

type User struct {
	ID                  uuid.UUID      `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	Username            string         `gorm:"size:30;not null;unique"`
	Email               string         `gorm:"size:100;not null;unique"`
	Password            string         `gorm:"size:128;not null"`
	Role                UserRole       `gorm:"type:varchar(20);not null;default:'operator'"`
	Departments         pq.StringArray `gorm:"type:text[]"`
	Permissions         pq.StringArray `gorm:"type:text[]"`
	Status              UserStatus     `gorm:"type:varchar(20);not null;default:'ACTIVE'"`
	FailedLoginAttempts int8           `gorm:"not null;default:0"`
	LockedUntil         *time.Time     `gorm:"index"`
	LastLogin           time.Time
	PasswordChangedAt   *time.Time
	LastIP              string         `gorm:"size:45"`
	TwoFactorEnabled    bool           `gorm:"not null;default:false"`
	TwoFactorSecret     string         `gorm:"size:64"`
	CreatedAt           time.Time      `gorm:"autoCreateTime"`
	UpdatedAt           time.Time      `gorm:"autoUpdateTime"`
	DeletedAt           gorm.DeletedAt `gorm:"index"`

	// Relacje
	Devices       []UserDevice   `gorm:"foreignKey:UserID"`
	RefreshTokens []RefreshToken `gorm:"foreignKey:UserID"`
}

type AvailablePermission struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:uuidv7()" json:"id"`
	Key         string    `gorm:"size:100;not null;uniqueIndex" json:"key"`
	Department  string    `gorm:"size:50;not null;index" json:"department"`
	Description string    `gorm:"size:255;not null" json:"description"`
	IsSpecial   bool      `gorm:"not null;default:false" json:"is_special"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
}
