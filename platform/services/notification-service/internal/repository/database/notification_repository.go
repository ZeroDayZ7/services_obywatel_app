package database

import (
	"github.com/zerodayz7/platform/services/notification-service/internal/model"
	"gorm.io/gorm"
)

type NotificationRepository struct {
	db *gorm.DB
}

func NewNotificationRepository(db *gorm.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

func (r *NotificationRepository) Create(notification *model.Notification) error {
	return r.db.Create(notification).Error
}

func (r *NotificationRepository) GetByUserID(userID uint) ([]model.Notification, error) {
	var notifications []model.Notification
	// GORM automatycznie zmapuje createdAt desc na created_at desc
	err := r.db.Where("user_id = ?", userID).Order("created_at desc").Find(&notifications).Error
	return notifications, err
}

// Zmieniono id z uint na string, aby pasowało do UUID
func (r *NotificationRepository) MarkAsRead(id string) error {
	// Upewnij się, że nazwa pola to "is_read" (tak GORM mapuje IsRead z modelu)
	return r.db.Model(&model.Notification{}).Where("id = ?", id).Update("is_read", true).Error
}

func (r *NotificationRepository) MarkAllAsRead(userID uint) error {
	// Aktualizujemy tylko powiadomienia należące do zalogowanego UserID
	return r.db.Model(&model.Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Update("is_read", true).Error
}
