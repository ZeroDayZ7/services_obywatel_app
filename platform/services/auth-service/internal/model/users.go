package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type UserRole string

const (
	RoleOperator UserRole = "operator"
	RoleUser     UserRole = "user"
	RoleAdmin    UserRole = "admin"
	RoleRoot     UserRole = "root"
)

// Domyślne uprawnienia systemowe w formacie dot-notation
const (
	PermSystemAdmin  = "system.admin"
	PermSystemManage = "system.manage"

	PermUsersRead   = "users.read"
	PermUsersWrite  = "users.write"
	PermUsersDelete = "users.delete"

	PermReportsView   = "reports.view"
	PermReportsExport = "reports.export"

	// Uprawnienia dla wiadomości i WebSocket
	PermMessagesRead    = "messages.read"
	PermMessagesWrite   = "messages.write"
	PermMessagingAccess = "messaging.access"

	// Uprawnienia dla dokumentów
	PermDocumentsRead  = "documents.read"
	PermDocumentsWrite = "documents.write"
)

type UserStatus string

const (
	StatusActive    UserStatus = "ACTIVE"
	StatusSuspended UserStatus = "SUSPENDED"
	StatusPending   UserStatus = "PENDING"
	StatusBanned    UserStatus = "BANNED"
	StatusLocked    UserStatus = "LOCKED"
)

// region User
type User struct {
	ID                  uuid.UUID      `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	Username            string         `gorm:"size:30;not null;unique"`
	Email               string         `gorm:"size:100;not null;unique"`
	Password            string         `gorm:"size:128;not null"`
	Role                UserRole       `gorm:"type:varchar(20);not null;default:'user'"`
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
	Agreement     *UserAgreement `gorm:"foreignKey:UserID"`
}

// region AvailablePermission
type AvailablePermission struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:uuidv7()" json:"id"`
	Key         string    `gorm:"size:100;not null;uniqueIndex" json:"key"`
	Department  string    `gorm:"size:50;not null;index" json:"department"`
	Description string    `gorm:"size:255;not null" json:"description"`
	IsSpecial   bool      `gorm:"not null;default:false" json:"is_special"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
}
