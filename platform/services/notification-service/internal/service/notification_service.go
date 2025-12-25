package service

import (
	"time"

	"github.com/zerodayz7/platform/services/notification-service/internal/model"
	mysql "github.com/zerodayz7/platform/services/notification-service/internal/repository/database"
)

// NotificationService logika biznesowa powiadomień
type NotificationService struct {
	repo *mysql.NotificationRepository
}

// NewNotificationService tworzy instancję serwisu
func NewNotificationService(repo *mysql.NotificationRepository) *NotificationService {
	return &NotificationService{repo: repo}
}

// Send tworzy powiadomienie
func (s *NotificationService) Send(userID uint, title, message, notifType string) error {
	notification := &model.Notification{
		UserID:    userID,
		Title:     title,
		Message:   message,
		Type:      notifType,
		Read:      false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	return s.repo.Create(notification)
}

// ListByUser pobiera wszystkie powiadomienia użytkownika
func (s *NotificationService) ListByUser(userID uint) ([]model.Notification, error) {
	return s.repo.GetByUserID(userID)
}

// MarkRead oznacza powiadomienie jako przeczytane
func (s *NotificationService) MarkRead(id uint) error {
	return s.repo.MarkAsRead(id)
}
