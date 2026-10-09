package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	apperr "github.com/zerodayz7/platform/pkg/errors"
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
	ErrInvalidSession       = errors.New("invalid or untrusted device session")
	ErrDeviceMismatch       = errors.New("sender device is not bound to the authenticated user")
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

	// Messaging activation and onboarding
	GetActivationStatus(ctx context.Context, userID uuid.UUID) (*model.MessagingActivationStatusResponse, error)
	GetCurrentTerms(ctx context.Context) (*model.MessagingTermsDocument, error)
	ActivateMessaging(ctx context.Context, userID uuid.UUID, req model.ActivateMessagingRequest) (*model.MessagingActivationStatusResponse, error)
	AcceptTerms(ctx context.Context, userID uuid.UUID, req model.AcceptTermsRequest) (*model.MessagingActivationStatusResponse, error)

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
			idempotencyKey := strings.TrimSpace(evt.IdempotencyKey)
			if idempotencyKey == "" && evt.EventID != uuid.Nil {
				idempotencyKey = evt.EventID.String()
			}
			if idempotencyKey != "" {
				if existing, err := s.repo.GetMessageByIdempotencyKey(ctx, idempotencyKey); err == nil && existing != nil {
					processed++
					continue
				} else if err != nil {
					return nil, err
				}
			}

			msg := &model.Message{
				ConversationID:   *evt.ConversationID,
				SenderID:         userID,
				Type:             model.MessageTypeText,
				EncryptedPayload: []byte(evt.Payload),
				IdempotencyKey:   idempotencyKey,
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
	if msg == nil {
		return ErrInvalidSession
	}

	s.logger.Info("[CONVERSATION_DB_LOOKUP] resolving conversation before message insert",
		"user_id", senderID.String(),
		"conversation_id", msg.ConversationID.String(),
		"sender_device_id", msg.SenderDeviceID,
	)

	if err := s.ValidateSenderDeviceOwnership(ctx, senderID, msg.SenderDeviceID); err != nil {
		return err
	}
	msg.SenderID = senderID

	s.logger.Info("[MESSAGE_DB_INSERT] creating message record",
		"user_id", senderID.String(),
		"conversation_id", msg.ConversationID.String(),
		"ciphertext_len", len(msg.EncryptedPayload),
	)

	if err := s.repo.CreateMessage(ctx, msg); err != nil {
		s.logger.Error("[MESSAGE_DB_INSERT_FAILED] failed to persist message",
			"user_id", senderID.String(),
			"conversation_id", msg.ConversationID.String(),
			"error", err.Error())
		return err
	}

	s.logger.Info("[MESSAGE_DB_INSERT_OK] message persisted",
		"user_id", senderID.String(),
		"conversation_id", msg.ConversationID.String(),
		"message_id", msg.ID.String(),
	)
	return nil
}

func (s *messagingService) GetContacts(ctx context.Context, ownerID uuid.UUID, sinceVersion uint64) ([]model.Contact, error) {
	s.logger.Info("[CONTACTS_FETCH_START] fetching contacts for owner",
		"owner_id", ownerID.String(),
		"since_version", sinceVersion,
	)

	contacts, err := s.repo.GetContactsSinceVersion(ctx, ownerID, sinceVersion)
	if err != nil {
		s.logger.Error("[CONTACTS_FETCH_FAILED] failed to fetch contacts",
			"owner_id", ownerID.String(),
			"since_version", sinceVersion,
			"error", err.Error(),
		)
		return nil, err
	}

	s.logger.Info("[CONTACTS_FETCH_OK] contacts loaded",
		"owner_id", ownerID.String(),
		"count", len(contacts),
		"since_version", sinceVersion,
	)
	return contacts, nil
}

func (s *messagingService) GetMessages(ctx context.Context, userID uuid.UUID, conversationID string, limit, offset int) ([]model.Message, error) {
	convID, err := uuid.Parse(conversationID)
	if err != nil {
		s.logger.Error("[MESSAGE_HISTORY_PARSE_FAILED] invalid conversation UUID",
			"user_id", userID.String(),
			"conversation_id", conversationID,
			"error", err.Error(),
		)
		return nil, ErrInvalidUUID
	}

	if limit <= 0 {
		limit = 20
	}

	s.logger.Info("[MESSAGE_HISTORY_FETCH_START] loading message history",
		"user_id", userID.String(),
		"conversation_id", convID.String(),
		"limit", limit,
		"offset", offset,
	)

	messages, err := s.repo.GetMessagesByConversation(ctx, userID, convID, limit, offset)
	if err != nil {
		s.logger.Error("[MESSAGE_HISTORY_FETCH_FAILED] failed to load message history",
			"user_id", userID.String(),
			"conversation_id", convID.String(),
			"error", err.Error(),
		)
		return nil, err
	}

	s.logger.Info("[MESSAGE_HISTORY_FETCH_OK] message history loaded",
		"user_id", userID.String(),
		"conversation_id", convID.String(),
		"count", len(messages),
	)
	return messages, nil
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

// #region MessagingActivation
func (s *messagingService) GetActivationStatus(ctx context.Context, userID uuid.UUID) (*model.MessagingActivationStatusResponse, error) {
	activation, err := s.repo.GetActivationStatus(ctx, userID)
	if err != nil {
		return nil, err
	}
	if activation == nil {
		activation = &model.MessagingActivation{
			UserID: userID,
			Status: model.ActivationStatusNotStarted,
		}
	}
	return activation.ToResponse(), nil
}

func (s *messagingService) GetCurrentTerms(ctx context.Context) (*model.MessagingTermsDocument, error) {
	_ = ctx
	terms := model.CurrentMessagingTerms()
	return &terms, nil
}

func (s *messagingService) ActivateMessaging(ctx context.Context, userID uuid.UUID, req model.ActivateMessagingRequest) (*model.MessagingActivationStatusResponse, error) {
	currentTerms := model.CurrentMessagingTerms()
	if !req.Consent {
		return nil, errors.New("explicit consent is required to activate messaging")
	}
	if strings.TrimSpace(req.TermsVersion) == "" {
		return nil, errors.New("terms_version is required for activation")
	}
	if req.TermsVersion != currentTerms.Version {
		return nil, fmt.Errorf("terms version %q is not the current version %q", req.TermsVersion, currentTerms.Version)
	}

	activation, err := s.repo.GetOrCreateActivation(ctx, userID, req.DeviceID)
	if err != nil {
		return nil, err
	}
	if req.DeviceID != "" {
		activation.DeviceID = req.DeviceID
	}
	activation.TermsVersion = req.TermsVersion
	activation.ConsentAccepted = true
	activation.Status = model.ActivationStatusActive
	now := time.Now()
	if activation.ActivatedAt == nil || activation.ActivatedAt.IsZero() {
		activation.ActivatedAt = &now
	}
	activation.LastSeenAt = &now
	if err := s.repo.SaveActivation(ctx, activation); err != nil {
		return nil, err
	}
	return activation.ToResponse(), nil
}

func (s *messagingService) AcceptTerms(ctx context.Context, userID uuid.UUID, req model.AcceptTermsRequest) (*model.MessagingActivationStatusResponse, error) {
	currentTerms := model.CurrentMessagingTerms()
	if strings.TrimSpace(req.TermsVersion) == "" {
		return nil, errors.New("terms_version is required")
	}
	if req.TermsVersion != currentTerms.Version {
		return nil, fmt.Errorf("terms version %q is not the current version %q", req.TermsVersion, currentTerms.Version)
	}

	activation, err := s.repo.GetOrCreateActivation(ctx, userID, req.DeviceID)
	if err != nil {
		return nil, err
	}
	if req.DeviceID != "" {
		activation.DeviceID = req.DeviceID
	}
	activation.TermsVersion = req.TermsVersion
	activation.ConsentAccepted = true
	if activation.Status == model.ActivationStatusNotStarted {
		activation.Status = model.ActivationStatusActive
	}
	now := time.Now()
	if activation.ActivatedAt == nil || activation.ActivatedAt.IsZero() {
		activation.ActivatedAt = &now
	}
	activation.LastSeenAt = &now
	if err := s.repo.SaveActivation(ctx, activation); err != nil {
		return nil, err
	}
	return activation.ToResponse(), nil
}

// #endregion

// #region E2EE
func validateSignalKeyMaterial(name string, key []byte) error {
	if len(key) == 0 {
		return fmt.Errorf("%s is required", name)
	}
	if len(key) != 33 && len(key) != 65 {
		return fmt.Errorf("%s must be a 33-byte compressed or 65-byte uncompressed EC public key, got %d bytes", name, len(key))
	}
	if bytes.Equal(bytes.TrimSpace(key), bytes.Repeat([]byte{0x00}, len(key))) {
		return fmt.Errorf("%s cannot be all-zero", name)
	}
	if bytes.IndexFunc(key, func(r rune) bool { return r < 0x20 || r > 0x7e }) == -1 && (bytes.Contains(key, []byte("PUBKEY")) || bytes.Contains(key, []byte("SIGNED_PREKEY")) || bytes.Contains(key, []byte("OTK_")) || bytes.Contains(key, []byte("SIG_"))) {
		return fmt.Errorf("%s contains fake/mock Signal key material", name)
	}
	return nil
}

func validateDeviceKeyUpload(req model.UploadDeviceKeysRequest) error {
	if strings.TrimSpace(req.DeviceID) == "" {
		return errors.New("device_id is required")
	}
	if req.RegistrationID == 0 {
		return errors.New("registration_id is required")
	}
	if err := validateSignalKeyMaterial("identity_public_key", req.IdentityPublicKey); err != nil {
		return err
	}
	if err := validateSignalKeyMaterial("signed_pre_key", req.SignedPreKey); err != nil {
		return err
	}
	if len(req.SignedPreKeySig) != 64 {
		return fmt.Errorf("signed_pre_key_sig must be 64 bytes, got %d", len(req.SignedPreKeySig))
	}
	if req.SignedPreKeyID == 0 {
		return errors.New("signed_pre_key_id is required")
	}
	seen := make(map[string]struct{}, len(req.OneTimePreKeys))
	for i, key := range req.OneTimePreKeys {
		if err := validateSignalKeyMaterial(fmt.Sprintf("one_time_pre_key[%d]", i), key); err != nil {
			return err
		}
		if _, exists := seen[string(key)]; exists {
			return fmt.Errorf("duplicate one_time_pre_key[%d] detected", i)
		}
		seen[string(key)] = struct{}{}
	}
	return nil
}

func BuildUserDeviceIdentity(userID uuid.UUID, req model.UploadDeviceKeysRequest) *model.UserDeviceIdentity {
	return &model.UserDeviceIdentity{
		UserID:              userID,
		DeviceID:            req.DeviceID,
		RegistrationID:      req.RegistrationID,
		PublicKey:           req.IdentityPublicKey,
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

func ValidateSenderDeviceBinding(userID uuid.UUID, deviceID string, trustedDevices []string) error {
	if userID == uuid.Nil {
		return ErrInvalidSession
	}
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return ErrInvalidSession
	}
	for _, trustedDeviceID := range trustedDevices {
		if strings.EqualFold(strings.TrimSpace(trustedDeviceID), deviceID) {
			return nil
		}
	}
	return ErrDeviceMismatch
}

func (s *messagingService) ValidateSenderDeviceOwnership(ctx context.Context, userID uuid.UUID, deviceID string) error {
	if userID == uuid.Nil || strings.TrimSpace(deviceID) == "" {
		s.logger.Error("[DEVICE_OWNERSHIP_INVALID] missing user or device id",
			"user_id", userID.String(),
			"device_id", deviceID,
		)
		return ErrInvalidSession
	}
	identity, err := s.repo.GetDeviceIdentity(ctx, userID, deviceID)
	if err != nil || identity == nil {
		s.logger.Error("[DEVICE_OWNERSHIP_MISMATCH] sender device not bound to authenticated user",
			"user_id", userID.String(),
			"device_id", deviceID,
			"error", err,
		)
		return ErrInvalidSession
	}
	return nil
}

func (s *messagingService) UploadDeviceKeys(ctx context.Context, userID uuid.UUID, req model.UploadDeviceKeysRequest) error {
	req.Normalize()
	s.logger.Info("[DEVICE_KEYS_UPLOAD_START] validating and persisting E2EE identity bundle",
		"user_id", userID.String(),
		"device_id", req.DeviceID,
		"pre_key_count", len(req.OneTimePreKeys),
	)
	if err := validateDeviceKeyUpload(req); err != nil {
		s.logger.Error("[DEVICE_KEYS_UPLOAD_VALIDATION_FAILED] rejected malformed device keys",
			"user_id", userID.String(),
			"device_id", req.DeviceID,
			"error", err.Error(),
		)
		return apperr.ErrValidationFailed.WithMeta("detail", err.Error())
	}

	identity := BuildUserDeviceIdentity(userID, req)
	if err := s.repo.SaveDeviceIdentity(ctx, identity); err != nil {
		s.logger.Error("[DEVICE_KEYS_SAVE_FAILED] unable to persist device identity",
			"user_id", userID.String(),
			"device_id", req.DeviceID,
			"error", err.Error(),
		)
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
