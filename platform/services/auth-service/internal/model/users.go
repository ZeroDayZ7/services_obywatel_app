package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type UserRole string

const (
	RoleOperator UserRole = "operator"
	RoleUser     UserRole = "user"
	RoleAdmin    UserRole = "admin"
	RoleRoot     UserRole = "root"
)

const (
	RoleCitizen         UserRole = "CITIZEN"
	RoleOfficer         UserRole = "OFFICER"
	RoleSupervisor      UserRole = "SUPERVISOR"
	RoleDepartmentAdmin UserRole = "DEPARTMENT_ADMIN"
	RoleSysAdmin        UserRole = "SYS_ADMIN"
	RoleAuditor         UserRole = "AUDITOR"
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
	ID                  uuid.UUID                   `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	Username            string                      `gorm:"size:30;not null;unique"`
	Email               string                      `gorm:"size:100;not null;unique"`
	Password            string                      `gorm:"size:128;not null"`
	Role                UserRole                    `gorm:"type:varchar(20);not null;default:'user'"`
	Departments         datatypes.JSONSlice[string] `gorm:"type:jsonb"`
	Permissions         datatypes.JSONSlice[string] `gorm:"type:jsonb"`
	Status              UserStatus                  `gorm:"type:varchar(20);not null;default:'ACTIVE'"`
	FailedLoginAttempts int8                        `gorm:"not null;default:0"`
	LockedUntil         *time.Time                  `gorm:"index"`
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

// region AvailablePermission
type AvailablePermission struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:uuidv7()" json:"id"`
	Key         string    `gorm:"size:100;not null;uniqueIndex" json:"key"`
	Department  string    `gorm:"size:50;not null;index" json:"department"`
	Description string    `gorm:"size:255;not null" json:"description"`
	IsSpecial   bool      `gorm:"not null;default:false" json:"is_special"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
}
