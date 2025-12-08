package model

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID               uint           `gorm:"primaryKey;autoIncrement"`
	Username         string         `gorm:"size:30;not null;unique"`
	Email            string         `gorm:"size:100;not null;unique"`
	Password         string         `gorm:"size:128;not null"`
	TwoFactorEnabled bool           `gorm:"not null;default:false"`
	TwoFactorSecret  string         `gorm:"size:64"`
	CreatedAt        time.Time      `gorm:"autoCreateTime"`
	UpdatedAt        time.Time      `gorm:"autoUpdateTime"`
	DeletedAt        gorm.DeletedAt `gorm:"index"`
}
