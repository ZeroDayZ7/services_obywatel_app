package model

import "time"

type User struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	Username         string    `gorm:"size:100;not null" json:"username"`
	Email            string    `gorm:"size:255;uniqueIndex;not null" json:"email"`
	Password         string    `gorm:"size:255;not null" json:"-"`
	TwoFactorEnabled bool      `gorm:"default:false" json:"two_factor_enabled"`
	TwoFactorSecret  string    `gorm:"size:255" json:"-"`
	CreatedAt        time.Time `gorm:"autoCreateTime" json:"created_at"`
}
