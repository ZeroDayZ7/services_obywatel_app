package service

import (
	"time"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/services/notification-service/internal/model"
	mysql "github.com/zerodayz7/platform/services/notification-service/internal/repository/database"
)

type NotificationService struct {
	repo *mysql.NotificationRepository
}

func NewNotificationService(repo *mysql.NotificationRepository) *NotificationService {
	return &NotificationService{repo: repo}
}

// Send tworzy powiadomienie z pełnymi danymi spójnymi z Flutterem
func (s *NotificationService) Send(n *model.Notification) error {
	// Generujemy UUID jeśli nie zostało podane
	if n.ID == "" {
		n.ID = uuid.New().String()
	}

	n.IsRead = false
	n.CreatedAt = time.Now()
	n.UpdatedAt = time.Now()

	return s.repo.Create(n)
}

// ListByUser pobiera listę dla konkretnego użytkownika
func (s *NotificationService) ListByUser(userID uint) ([]model.Notification, error) {
	return s.repo.GetByUserID(userID)
}

func (s *NotificationService) MarkAllRead(userID uint) error {
	return s.repo.MarkAllAsRead(userID)
}

// MarkRead zmienia status na przeczytane
func (s *NotificationService) MarkRead(id string) error {
	return s.repo.MarkAsRead(id)
}
