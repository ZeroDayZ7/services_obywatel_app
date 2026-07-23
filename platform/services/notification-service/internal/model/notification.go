package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Notification struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:uuidv7()" json:"id"`
	UserID    uuid.UUID      `gorm:"type:uuid;not null;index:idx_user_notifications" json:"userId"`
	Title     string         `gorm:"type:varchar(255);not null" json:"title"`
	Content   string         `gorm:"type:text;not null" json:"content"`
	Priority  string         `gorm:"type:varchar(20);not null;default:'normal'" json:"priority"`
	Category  string         `gorm:"type:varchar(50);not null;default:'system'" json:"category"`
	IsRead    bool           `gorm:"not null;default:false;index:idx_user_notifications" json:"isRead"`
	CreatedAt time.Time      `gorm:"autoCreateTime;index:idx_user_notifications" json:"createdAt"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type NotificationEvent struct {
	UserID   uuid.UUID      `json:"user_id"`
	Title    string         `json:"title"`
	Content  string         `json:"content"`
	Priority string         `json:"priority"`
	Category string         `json:"category"`
	Metadata map[string]any `json:"metadata"`
}
