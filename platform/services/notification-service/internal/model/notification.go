package model

import (
	"time"

	"gorm.io/gorm"
)

// Notification reprezentuje powiadomienie w systemie
type Notification struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	UserID    uint           `gorm:"index" json:"user_id"`
	Title     string         `gorm:"type:varchar(255)" json:"title"`
	Message   string         `gorm:"type:text" json:"message"`
	Type      string         `gorm:"type:varchar(50)" json:"type"`
	Read      bool           `gorm:"default:false" json:"read"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
