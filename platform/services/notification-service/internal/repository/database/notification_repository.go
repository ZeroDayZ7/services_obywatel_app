package database

import (
	"time"

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
	// Pobieramy tylko te, które NIE są w koszu (GORM automatycznie obsłuży deleted_at IS NULL)
	err := r.db.Where("user_id = ?", userID).Order("created_at desc").Find(&notifications).Error
	return notifications, err
}

// POPRAWIONE: Dodano userID dla bezpieczeństwa
func (r *NotificationRepository) MarkAsRead(id string, userID uint) error {
	return r.db.Model(&model.Notification{}).
		Where("id = ? AND user_id = ?", id, userID).
		Update("is_read", true).Error
}

func (r *NotificationRepository) MarkAllAsRead(userID uint) error {
	return r.db.Model(&model.Notification{}).
		Where("user_id = ? AND is_read = ? AND deleted_at IS NULL", userID, false).
		Update("is_read", true).Error
}

func (r *NotificationRepository) MoveToTrash(id string, userID uint) error {
	return r.db.Model(&model.Notification{}).
		Where("id = ? AND user_id = ?", id, userID).
		Update("deleted_at", time.Now()).Error
}

func (r *NotificationRepository) HardDeleteTrash(userID uint) error {
	return r.db.Unscoped().
		Where("user_id = ? AND deleted_at IS NOT NULL", userID).
		Delete(&model.Notification{}).Error
}
