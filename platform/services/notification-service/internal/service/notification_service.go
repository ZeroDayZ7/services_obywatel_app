package service

import (
	"context"

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

func (s *NotificationService) Send(ctx context.Context, n *model.Notification) error {
	// Logika ID i dat została przeniesiona do model.BeforeCreate
	return s.repo.Create(ctx, n)
}

func (s *NotificationService) ListByUser(ctx context.Context, userID uuid.UUID) ([]model.Notification, error) {
	return s.repo.GetByUserID(ctx, userID)
}

func (s *NotificationService) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	return s.repo.MarkAllAsRead(ctx, userID)
}

func (s *NotificationService) MarkRead(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return s.repo.MarkAsRead(ctx, id, userID)
}

func (s *NotificationService) MoveToTrash(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return s.repo.MoveToTrash(ctx, id, userID)
}

func (s *NotificationService) ClearTrash(ctx context.Context, userID uuid.UUID) error {
	return s.repo.HardDeleteTrash(ctx, userID)
}

func (s *NotificationService) Restore(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return s.repo.RestoreFromTrash(ctx, id, userID)
}

func (s *NotificationService) DeletePermanently(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return s.repo.DeletePermanently(ctx, id, userID)
}
