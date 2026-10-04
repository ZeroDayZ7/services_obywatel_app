package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/messaging-service/config"
	"github.com/zerodayz7/platform/services/messaging-service/internal/model"
	"github.com/zerodayz7/platform/services/messaging-service/internal/repository"
	"gorm.io/gorm"
)

var (
	ErrInvalidUUID          = errors.New("invalid uuid format")
	ErrConversationNotFound = errors.New("conversation not found")
	ErrDeviceNotFound       = errors.New("device identity not found")
)

type MessagingService interface {
	// Delta & Outbox
	GetDeltaSync(ctx context.Context, userID uuid.UUID, req model.SyncDeltaRequest) (*model.SyncDeltaResponse, error)
	ProcessOutbox(ctx context.Context, userID uuid.UUID, req model.OutboxBatchRequest) (*model.OutboxBatchResponse, error)

	// Messages & Contacts
	SendMessage(ctx context.Context, senderID uuid.UUID, msg *model.Message) error
	GetContacts(ctx context.Context, ownerID uuid.UUID, sinceVersion uint64) ([]model.Contact, error)
	GetMessages(ctx context.Context, userID uuid.UUID, conversationID string, limit, offset int) ([]model.Message, error)

	// Conversations
	GetConversations(ctx context.Context, userID uuid.UUID) ([]model.Conversation, error)
	CreateConversation(ctx context.Context, userID uuid.UUID, req model.CreateConversationRequest) (*model.Conversation, error)
	GetConversationByID(ctx context.Context, userID uuid.UUID, conversationID string) (*model.Conversation, error)
	MarkAsRead(ctx context.Context, userID uuid.UUID, conversationID string) error

	// E2EE Keys
	UploadDeviceKeys(ctx context.Context, userID uuid.UUID, req model.UploadDeviceKeysRequest) error
	GetUserKeyBundle(ctx context.Context, targetUserID string) (*model.PreKeyBundleDto, error)
	GetUserPreKeys(ctx context.Context, targetUserID string) (*model.UserPreKeysResponse, error)
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

// #region DeltaAndOutbox
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

func (s *messagingService) ProcessOutbox(ctx context.Context, userID uuid.UUID, req model.OutboxBatchRequest) (*model.OutboxBatchResponse, error) {
	processed := 0
	for _, evt := range req.Messages {
		if evt.EventType == "SEND_MESSAGE" && evt.ConversationID != nil {
			msg := &model.Message{
				ConversationID:   *evt.ConversationID,
				SenderID:         userID,
				Type:             model.MessageTypeText,
				EncryptedPayload: []byte(evt.Payload),
			}
			if err := s.repo.CreateMessage(ctx, msg); err == nil {
				processed++
			}
		}
	}

	return &model.OutboxBatchResponse{ProcessedCount: processed}, nil
}

// #endregion

// #region MessagesAndContacts
func (s *messagingService) SendMessage(ctx context.Context, senderID uuid.UUID, msg *model.Message) error {
	msg.SenderID = senderID
	return s.repo.CreateMessage(ctx, msg)
}

func (s *messagingService) GetContacts(ctx context.Context, ownerID uuid.UUID, sinceVersion uint64) ([]model.Contact, error) {
	return s.repo.GetContactsSinceVersion(ctx, ownerID, sinceVersion)
}

func (s *messagingService) GetMessages(ctx context.Context, userID uuid.UUID, conversationID string, limit, offset int) ([]model.Message, error) {
	convID, err := uuid.Parse(conversationID)
	if err != nil {
		return nil, ErrInvalidUUID
	}

	if limit <= 0 {
		limit = 20
	}

	return s.repo.GetMessagesByConversation(ctx, userID, convID, limit, offset)
}

// #endregion

// #region Conversations
func (s *messagingService) GetConversations(ctx context.Context, userID uuid.UUID) ([]model.Conversation, error) {
	return s.repo.GetConversations(ctx, userID)
}

func (s *messagingService) CreateConversation(ctx context.Context, userID uuid.UUID, req model.CreateConversationRequest) (*model.Conversation, error) {
	conv := &model.Conversation{
		Type:  req.Type,
		Title: req.Title,
		Members: []model.ConversationMember{
			{
				UserID: userID,
				Role:   "admin",
			},
		},
	}

	for _, recipientID := range req.RecipientIDs {
		if recipientID != userID {
			conv.Members = append(conv.Members, model.ConversationMember{
				UserID: recipientID,
				Role:   "member",
			})
		}
	}

	if err := s.repo.CreateConversation(ctx, conv); err != nil {
		return nil, err
	}

	return conv, nil
}

func (s *messagingService) GetConversationByID(ctx context.Context, userID uuid.UUID, conversationID string) (*model.Conversation, error) {
	convID, err := uuid.Parse(conversationID)
	if err != nil {
		return nil, ErrInvalidUUID
	}

	return s.repo.GetConversationByID(ctx, userID, convID)
}

func (s *messagingService) MarkAsRead(ctx context.Context, userID uuid.UUID, conversationID string) error {
	convID, err := uuid.Parse(conversationID)
	if err != nil {
		return ErrInvalidUUID
	}

	conv, err := s.repo.GetConversationByID(ctx, userID, convID)
	if err != nil {
		return err
	}

	return s.repo.UpdateLastReadSequence(ctx, userID, convID, conv.LastSequence)
}

// #endregion

// #region E2EE
func BuildUserDeviceIdentity(userID uuid.UUID, req model.UploadDeviceKeysRequest) *model.UserDeviceIdentity {
	return &model.UserDeviceIdentity{
		UserID:              userID,
		DeviceID:            req.DeviceID,
		RegistrationID:      req.RegistrationID,
		PublicKey:           req.PublicKey,
		SignedPreKey:        req.SignedPreKey,
		SignedPreKeySig:     req.SignedPreKeySig,
		SignedPreKeyID:      req.SignedPreKeyID,
		OneTimePreKeysCount: len(req.OneTimePreKeys),
	}
}

func BuildPreKeyBundle(identity *model.UserDeviceIdentity, oneTimePreKey *model.UserPreKey) *model.PreKeyBundleDto {
	if identity == nil {
		return nil
	}

	bundle := &model.PreKeyBundleDto{
		RegistrationID:        identity.RegistrationID,
		DeviceID:              identity.DeviceID,
		SignedPreKeyID:        identity.SignedPreKeyID,
		SignedPreKeyPublic:    identity.SignedPreKey,
		SignedPreKeySignature: identity.SignedPreKeySig,
		IdentityKey:           identity.PublicKey,
	}
	if oneTimePreKey != nil {
		preKeyID := oneTimePreKey.KeyID
		bundle.PreKeyID = &preKeyID
		bundle.PreKeyPublic = oneTimePreKey.PublicKey
	}
	return bundle
}

func (s *messagingService) UploadDeviceKeys(ctx context.Context, userID uuid.UUID, req model.UploadDeviceKeysRequest) error {
	req.Normalize()
	if req.DeviceID == "" {
		return errors.New("device_id is required")
	}

	identity := BuildUserDeviceIdentity(userID, req)
	if err := s.repo.SaveDeviceIdentity(ctx, identity); err != nil {
		return err
	}

	storedIdentity, err := s.repo.GetDeviceIdentity(ctx, userID, req.DeviceID)
	if err != nil {
		return err
	}
	identity.ID = storedIdentity.ID

	if len(req.OneTimePreKeys) > 0 {
		preKeys := make([]model.UserPreKey, 0, len(req.OneTimePreKeys))
		for i, keyBytes := range req.OneTimePreKeys {
			preKeys = append(preKeys, model.UserPreKey{
				DeviceID:  identity.ID,
				KeyID:     uint32(i + 1),
				PublicKey: keyBytes,
			})
		}
		if err := s.repo.SavePreKeys(ctx, preKeys); err != nil {
			return err
		}
	}

	return nil
}

func (s *messagingService) GetUserKeyBundle(ctx context.Context, targetUserID string) (*model.PreKeyBundleDto, error) {
	targetID, err := uuid.Parse(targetUserID)
	if err != nil {
		return nil, ErrInvalidUUID
	}

	identity, err := s.repo.GetLatestDeviceIdentityForUser(ctx, targetID)
	if err != nil {
		return nil, ErrDeviceNotFound
	}

	preKey, err := s.repo.PopPreKey(ctx, identity.UserID, identity.DeviceID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if err != nil {
		return BuildPreKeyBundle(identity, nil), nil
	}
	return BuildPreKeyBundle(identity, preKey), nil
}

func (s *messagingService) GetUserPreKeys(ctx context.Context, targetUserID string) (*model.UserPreKeysResponse, error) {
	bundle, err := s.GetUserKeyBundle(ctx, targetUserID)
	if err != nil {
		return nil, err
	}

	targetID, err := uuid.Parse(targetUserID)
	if err != nil {
		return nil, ErrInvalidUUID
	}

	identity, err := s.repo.GetLatestDeviceIdentityForUser(ctx, targetID)
	if err != nil {
		return nil, ErrDeviceNotFound
	}

	res := &model.UserPreKeysResponse{
		UserID:          identity.UserID,
		DeviceID:        identity.DeviceID,
		IdentityKey:     identity.PublicKey,
		SignedPreKey:    identity.SignedPreKey,
		SignedPreKeySig: identity.SignedPreKeySig,
		SignedPreKeyID:  identity.SignedPreKeyID,
	}
	if bundle != nil {
		res.OneTimePreKey = bundle.PreKeyPublic
		if bundle.PreKeyID != nil {
			res.OneTimePreKeyID = *bundle.PreKeyID
		}
	}
	return res, nil
}

// #endregion
