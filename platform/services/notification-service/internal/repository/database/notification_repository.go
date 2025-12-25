package database

import (
	"github.com/zerodayz7/platform/services/notification-service/internal/model"
	"gorm.io/gorm"
)

// NotificationRepository obsługuje bazę powiadomień
type NotificationRepository struct {
	db *gorm.DB
}

// NewNotificationRepository tworzy instancję repozytorium
func NewNotificationRepository(db *gorm.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

// Create zapisuje nowe powiadomienie
func (r *NotificationRepository) Create(notification *model.Notification) error {
	return r.db.Create(notification).Error
}

// GetByUserID pobiera powiadomienia dla użytkownika
func (r *NotificationRepository) GetByUserID(userID uint) ([]model.Notification, error) {
	var notifications []model.Notification
	err := r.db.Where("user_id = ?", userID).Order("created_at desc").Find(&notifications).Error
	return notifications, err
}

// MarkAsRead oznacza powiadomienie jako przeczytane
func (r *NotificationRepository) MarkAsRead(id uint) error {
	return r.db.Model(&model.Notification{}).Where("id = ?", id).Update("read", true).Error
}
