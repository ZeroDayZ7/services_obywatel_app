package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/services/notification-service/internal/model"
)

// #region Interfaces
type NotificationRepository interface {
	Create(ctx context.Context, n *model.Notification) error
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]model.Notification, error)
	MarkAllAsRead(ctx context.Context, userID uuid.UUID) error
	MarkAsRead(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	MoveToTrash(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	HardDeleteTrash(ctx context.Context, userID uuid.UUID) error
	RestoreFromTrash(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	DeletePermanently(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	ProcessSyncBatch(ctx context.Context, userID uuid.UUID, req model.SyncBatchRequest) (processed []string, failed []string, err error)
}

// #endregion

type NotificationService struct {
	repo NotificationRepository
}

// #region NewNotificationService
func NewNotificationService(repo NotificationRepository) *NotificationService {
	return &NotificationService{repo: repo}
}

// #endregion

// #region Send
func (s *NotificationService) Send(ctx context.Context, n *model.Notification) error {
	return s.repo.Create(ctx, n)
}

// #endregion

// #region ListByUser
func (s *NotificationService) ListByUser(ctx context.Context, userID uuid.UUID) ([]model.Notification, error) {
	return s.repo.GetByUserID(ctx, userID)
}

// #endregion

// #region MarkAllRead
func (s *NotificationService) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	return s.repo.MarkAllAsRead(ctx, userID)
}

// #endregion

// #region MarkRead
func (s *NotificationService) MarkRead(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return s.repo.MarkAsRead(ctx, id, userID)
}

// #endregion

// #region MoveToTrash
func (s *NotificationService) MoveToTrash(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return s.repo.MoveToTrash(ctx, id, userID)
}

// #endregion

// #region ClearTrash
func (s *NotificationService) ClearTrash(ctx context.Context, userID uuid.UUID) error {
	return s.repo.HardDeleteTrash(ctx, userID)
}

// #endregion

// #region Restore
func (s *NotificationService) Restore(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return s.repo.RestoreFromTrash(ctx, id, userID)
}

// #endregion

// #region DeletePermanently
func (s *NotificationService) DeletePermanently(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return s.repo.DeletePermanently(ctx, id, userID)
}

// ProcessSyncBatch processes a batch of outbox events for a given user.
func (s *NotificationService) ProcessSyncBatch(ctx context.Context, userID uuid.UUID, req model.SyncBatchRequest) (*model.SyncBatchResponse, error) {
	processed, failed, err := s.repo.ProcessSyncBatch(ctx, userID, req)
	if err != nil {
		return nil, err
	}
	return &model.SyncBatchResponse{
		ProcessedEventIDs: processed,
		FailedEventIDs:    failed,
	}, nil
}

// #endregion
