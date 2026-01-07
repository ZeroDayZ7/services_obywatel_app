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
	// POPRAWKA 1: Sprawdzamy czy UUID jest "zerowy" (uuid.Nil) zamiast ""
	if n.ID == uuid.Nil {
		// POPRAWKA 2: Przypisujemy czysty uuid.UUID zamiast .String()
		n.ID = uuid.New()
	}

	n.IsRead = false
	n.CreatedAt = time.Now()
	n.UpdatedAt = time.Now()

	return s.repo.Create(n)
}

// ListByUser pobiera listę dla konkretnego użytkownika
func (s *NotificationService) ListByUser(userID uuid.UUID) ([]model.Notification, error) {
	return s.repo.GetByUserID(userID)
}

func (s *NotificationService) MarkAllRead(userID uuid.UUID) error {
	return s.repo.MarkAllAsRead(userID)
}

// POPRAWIONE: Teraz przyjmuje userID i przekazuje je do repozytorium
func (s *NotificationService) MarkRead(id uuid.UUID, userID uuid.UUID) error {
	return s.repo.MarkAsRead(id, userID)
}

// MoveToTrash - Soft Delete (przeniesienie do kosza)
func (s *NotificationService) MoveToTrash(id uuid.UUID, userID uuid.UUID) error {
	return s.repo.MoveToTrash(id, userID)
}

// ClearTrash - Hard Delete (opróżnienie kosza)
func (s *NotificationService) ClearTrash(userID uuid.UUID) error {
	return s.repo.HardDeleteTrash(userID)
}

func (s *NotificationService) Restore(id uuid.UUID, userID uuid.UUID) error {
	return s.repo.RestoreFromTrash(id, userID)
}

func (s *NotificationService) DeletePermanently(id uuid.UUID, userID uuid.UUID) error {
	return s.repo.DeletePermanently(id, userID)
}
