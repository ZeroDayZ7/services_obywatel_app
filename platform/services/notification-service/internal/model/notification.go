package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Notification struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	UserID    uuid.UUID      `gorm:"type:uuid;index" json:"userId"`
	Title     string         `gorm:"type:varchar(255)" json:"title"`
	Content   string         `gorm:"type:text" json:"content"`         // Zmienione z Message
	Priority  string         `gorm:"type:varchar(20)" json:"priority"` // info, success, warning, error
	Category  string         `gorm:"type:varchar(50)" json:"category"` // payments, security, itp.
	IsRead    bool           `gorm:"default:false" json:"isRead"`      // camelCase dla Fluttera
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
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

func (n *Notification) BeforeCreate(tx *gorm.DB) (err error) {
	if n.ID == uuid.Nil {
		n.ID = uuid.New()
	}
	return
}
