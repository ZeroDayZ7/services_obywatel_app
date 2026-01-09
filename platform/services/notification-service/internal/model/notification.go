package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/pkg/shared" // import Twojego pakietu
	"gorm.io/gorm"
)

type Notification struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	UserID    uuid.UUID      `gorm:"type:uuid;index" json:"userId"`
	Title     string         `gorm:"type:varchar(255)" json:"title"`
	Content   string         `gorm:"type:text" json:"content"`
	Priority  string         `gorm:"type:varchar(20)" json:"priority"`
	Category  string         `gorm:"type:varchar(50)" json:"category"`
	IsRead    bool           `gorm:"default:false" json:"isRead"`
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
		n.ID = shared.MustGenerateUuidV7()
	}
	now := time.Now().UTC()
	n.CreatedAt = now
	n.UpdatedAt = now
	return nil
}
