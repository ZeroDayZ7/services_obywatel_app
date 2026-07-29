package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/messaging-service/config"
	"github.com/zerodayz7/platform/services/messaging-service/internal/model"
	"github.com/zerodayz7/platform/services/messaging-service/internal/repository"
)

type MessagingService interface {
	GetDeltaSync(ctx context.Context, userID uuid.UUID, req model.SyncDeltaRequest) (*model.SyncDeltaResponse, error)
	SendMessage(ctx context.Context, senderID uuid.UUID, msg *model.Message) error
	GetContacts(ctx context.Context, ownerID uuid.UUID, sinceVersion uint64) ([]model.Contact, error)
}

type messagingService struct {
	repo   repository.MessagingRepository
	cfg    *config.Config
	logger *shared.Logger
}

func NewMessagingService(repo repository.MessagingRepository, cfg *config.Config, logger *shared.Logger) MessagingService {
	return &messagingService{
		repo:   repo,
		cfg:    cfg,
		logger: logger,
	}
}

func (s *messagingService) GetDeltaSync(ctx context.Context, userID uuid.UUID, req model.SyncDeltaRequest) (*model.SyncDeltaResponse, error) {
	contacts, err := s.repo.GetContactsSinceVersion(ctx, userID, req.LastKnownContactVersion)
	if err != nil {
		return nil, err
	}

	messages, err := s.repo.GetMessagesSinceVersion(ctx, userID, req.LastKnownMessageVersion)
	if err != nil {
		return nil, err
	}

	return &model.SyncDeltaResponse{
		UpdatedContacts: contacts,
		NewMessages:     messages,
		HasMore:         false,
	}, nil
}

func (s *messagingService) SendMessage(ctx context.Context, senderID uuid.UUID, msg *model.Message) error {
	msg.SenderID = senderID
	return s.repo.CreateMessage(ctx, msg)
}

func (s *messagingService) GetContacts(ctx context.Context, ownerID uuid.UUID, sinceVersion uint64) ([]model.Contact, error) {
	return s.repo.GetContactsSinceVersion(ctx, ownerID, sinceVersion)
}
