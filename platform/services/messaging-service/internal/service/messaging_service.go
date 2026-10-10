package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
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
	s.logger.Info("[SYNC-DELTA-1] starting delta sync",
		"user_id", userID.String(),
		"last_known_contact_version", req.LastKnownContactVersion,
		"last_known_message_version", req.LastKnownMessageVersion,
	)

	contacts, err := s.repo.GetContactsSinceVersion(ctx, userID, req.LastKnownContactVersion)
	if err != nil {
		s.logger.Error("[SYNC-DELTA-1.1] failed to fetch contacts for delta sync",
			"user_id", userID.String(),
			"error", err.Error(),
		)
		return nil, err
	}

	messages, err := s.repo.GetMessagesSinceVersion(ctx, userID, req.LastKnownMessageVersion)
	if err != nil {
		s.logger.Error("[SYNC-DELTA-1.2] failed to fetch messages for delta sync",
			"user_id", userID.String(),
			"error", err.Error(),
		)
		return nil, err
	}

	resp := &model.SyncDeltaResponse{
		UpdatedContacts: contacts,
		NewMessages:     messages,
		HasMore:         false,
	}

	s.logger.Info("[SYNC-DELTA-1.3] delta sync completed",
		"user_id", userID.String(),
		"contacts_count", len(contacts),
		"messages_count", len(messages),
	)
	return resp, nil
}

func (s *messagingService) ProcessOutbox(ctx context.Context, userID uuid.UUID, req model.OutboxBatchRequest) (*model.OutboxBatchResponse, error) {
	processed := 0
	s.logger.Info("[OUTBOX-1] processing outbox batch",
		"user_id", userID.String(),
		"message_count", len(req.Messages),
	)

	for _, evt := range req.Messages {
		s.logger.Info("[OUTBOX-1.1] handling outbox event",
			"user_id", userID.String(),
			"event_id", evt.EventID,
			"event_type", evt.EventType,
			"conversation_id", evt.ConversationID,
			"sender_device_id", evt.SenderDeviceID,
		)
		msg, ok, err := buildMessageFromOutboxEvent(userID, evt)
		if err != nil {
			s.logger.Error("[OUTBOX-1.2] outbox event conversion failed",
				"user_id", userID.String(),
				"event_id", evt.EventID,
				"error", err.Error(),
			)
			return nil, err
		}
		if !ok {
			s.logger.Warn("[OUTBOX-1.3] outbox event skipped",
				"user_id", userID.String(),
				"event_id", evt.EventID,
				"event_type", evt.EventType,
			)
			continue
		}

		if msg.IdempotencyKey != "" {
			if existing, err := s.repo.GetMessageByIdempotencyKey(ctx, msg.IdempotencyKey); err == nil && existing != nil {
				s.logger.Info("[OUTBOX-1.4] duplicate message found by idempotency key",
					"user_id", userID.String(),
					"idempotency_key", msg.IdempotencyKey,
				)
				processed++
				continue
			} else if err != nil {
				s.logger.Error("[OUTBOX-1.5] failed to check idempotency key",
					"user_id", userID.String(),
					"idempotency_key", msg.IdempotencyKey,
					"error", err.Error(),
				)
				return nil, err
			}
		}

		if err := s.repo.CreateMessage(ctx, msg); err == nil {
			processed++
			s.logger.Info("[OUTBOX-1.6] message persisted from outbox",
				"user_id", userID.String(),
				"conversation_id", msg.ConversationID.String(),
				"idempotency_key", msg.IdempotencyKey,
			)
		} else if !errors.Is(err, gorm.ErrDuplicatedKey) {
			s.logger.Error("[OUTBOX-1.7] message persistence failed",
				"user_id", userID.String(),
				"conversation_id", msg.ConversationID.String(),
				"idempotency_key", msg.IdempotencyKey,
				"error", err.Error(),
			)
			return nil, err
		}
	}

	s.logger.Info("[OUTBOX-1.8] outbox batch complete",
		"user_id", userID.String(),
		"processed_count", processed,
	)
	return &model.OutboxBatchResponse{ProcessedCount: processed}, nil
}

func buildMessageFromOutboxEvent(userID uuid.UUID, evt model.OutboxEventPayload) (*model.Message, bool, error) {
	shared.GetLogger().Info("[OUTBOX-BUILD-1] normalizing outbox event",
		"user_id", userID.String(),
		"event_id", evt.EventID,
		"event_type", evt.EventType,
	)

	if evt.EventType != "" && evt.EventType != "SEND_MESSAGE" {
		shared.GetLogger().Warn("[OUTBOX-BUILD-1.1] unsupported event type skipped",
			"user_id", userID.String(),
			"event_type", evt.EventType,
		)
		return nil, false, nil
	}

	evt.Normalize()
	conversationID := evt.ConversationID
	ciphertext := strings.TrimSpace(evt.Ciphertext)
	idempotencyKey := strings.TrimSpace(evt.IdempotencyKey)
	resolvedSignalType := evt.Type
	if resolvedSignalType == 0 {
		resolvedSignalType = evt.TypeAlt
	}
	messageType := model.MessageTypeText
	if resolvedSignalType != 0 {
		messageType = model.MessageType(strconv.Itoa(int(resolvedSignalType)))
	}

	if conversationID == nil && len(evt.Payload) > 0 {
		var payload map[string]any
		if err := json.Unmarshal(evt.Payload, &payload); err == nil {
			if rawConversationID, ok := payload["conversation_id"].(string); ok && rawConversationID != "" {
				parsed, err := uuid.Parse(rawConversationID)
				if err == nil {
					conversationID = &parsed
				}
			}
			if conversationID == nil {
				if rawConversationID, ok := payload["conversationId"].(string); ok && rawConversationID != "" {
					parsed, err := uuid.Parse(rawConversationID)
					if err == nil {
						conversationID = &parsed
					}
				}
			}
			if ciphertext == "" {
				if rawCiphertext, ok := payload["ciphertext"].(string); ok {
					ciphertext = strings.TrimSpace(rawCiphertext)
				}
			}
			if idempotencyKey == "" {
				if rawKey, ok := payload["idempotency_key"].(string); ok {
					idempotencyKey = strings.TrimSpace(rawKey)
				}
			}
		}
	}

	if evt.EventType == "" && ciphertext != "" {
		evt.EventType = "SEND_MESSAGE"
	}
	if evt.EventType == "" && evt.EventID != uuid.Nil {
		evt.EventType = "SEND_MESSAGE"
	}
	if evt.EventType == "" {
		shared.GetLogger().Warn("[OUTBOX-BUILD-1.2] missing event type, skipping",
			"user_id", userID.String(),
			"event_id", evt.EventID,
		)
		return nil, false, nil
	}
	if conversationID == nil || *conversationID == uuid.Nil {
		shared.GetLogger().Warn("[OUTBOX-BUILD-1.3] missing conversation id, skipping",
			"user_id", userID.String(),
			"event_id", evt.EventID,
		)
		return nil, false, nil
	}
	if strings.TrimSpace(evt.Content) != "" {
		shared.GetLogger().Warn("[OUTBOX-BUILD-1.4] plaintext content rejected as ciphertext",
			"user_id", userID.String(),
			"event_id", evt.EventID,
		)
		return nil, false, apperr.ErrValidationFailed.WithMeta("detail", "ciphertext is required; plaintext content is rejected")
	}
	if ciphertext == "" {
		shared.GetLogger().Warn("[OUTBOX-BUILD-1.5] empty ciphertext rejected",
			"user_id", userID.String(),
			"event_id", evt.EventID,
		)
		return nil, false, apperr.ErrValidationFailed.WithMeta("detail", "ciphertext is required")
	}
	if idempotencyKey == "" && evt.EventID != uuid.Nil {
		idempotencyKey = evt.EventID.String()
	}
	if idempotencyKey == "" {
		idempotencyKey = uuid.NewString()
	}
	if evt.SenderDeviceID == "" {
		evt.SenderDeviceID = evt.SenderDeviceIDAlt
	}
	if evt.SenderDeviceID == "" {
		evt.SenderDeviceID = "unknown-device"
	}

	msg := &model.Message{
		ConversationID:   *conversationID,
		SenderID:         userID,
		SenderDeviceID:   evt.SenderDeviceID,
		Type:             messageType,
		EncryptedPayload: []byte(ciphertext),
		IdempotencyKey:   idempotencyKey,
	}
	shared.GetLogger().Info("[OUTBOX-BUILD-1.6] message built from outbox event",
		"user_id", userID.String(),
		"conversation_id", msg.ConversationID.String(),
		"signal_type", string(msg.Type),
		"ciphertext_len", len(ciphertext),
		"idempotency_key_present", idempotencyKey != "",
	)
	return msg, true, nil
}

// #endregion

// #region MessagesAndContacts
func (s *messagingService) SendMessage(ctx context.Context, senderID uuid.UUID, msg *model.Message) error {
	if msg == nil {
		return ErrInvalidSession
	}

	s.logger.Info("[MESSAGE-SEND-1] starting message persistence flow",
		"user_id", senderID.String(),
		"conversation_id", msg.ConversationID.String(),
		"sender_device_id", msg.SenderDeviceID,
		"ciphertext_len", len(msg.EncryptedPayload),
	)
	if strings.TrimSpace(msg.IdempotencyKey) == "" {
		msg.IdempotencyKey = uuid.NewString()
	}
	if existing, err := s.repo.GetMessageByIdempotencyKey(ctx, msg.IdempotencyKey); err != nil {
		return err
	} else if existing != nil {
		s.logger.Info("[MESSAGE-SEND-1.1] duplicate by idempotency key; skipping",
			"user_id", senderID.String(),
			"idempotency_key", msg.IdempotencyKey,
		)
		return nil
	}

	s.logger.Info("[MESSAGE-SEND-1.2] validating sender device ownership",
		"user_id", senderID.String(),
		"conversation_id", msg.ConversationID.String(),
		"sender_device_id", msg.SenderDeviceID,
		"idempotency_key", msg.IdempotencyKey,
	)

	if err := s.ValidateSenderDeviceOwnership(ctx, senderID, msg.SenderDeviceID); err != nil {
		return err
	}
	msg.SenderID = senderID

	s.logger.Info("[MESSAGE-SEND-1.3] persisting message record",
		"user_id", senderID.String(),
		"conversation_id", msg.ConversationID.String(),
		"ciphertext_len", len(msg.EncryptedPayload),
		"idempotency_key", msg.IdempotencyKey,
	)

	if err := s.repo.CreateMessage(ctx, msg); err != nil {
		if existing, lookupErr := s.repo.GetMessageByIdempotencyKey(ctx, msg.IdempotencyKey); lookupErr == nil && existing != nil {
			s.logger.Info("[MESSAGE-SEND-1.4] message already persisted during race",
				"user_id", senderID.String(),
				"idempotency_key", msg.IdempotencyKey,
			)
			return nil
		}
		s.logger.Error("[MESSAGE-SEND-1.5] failed to persist message",
			"user_id", senderID.String(),
			"conversation_id", msg.ConversationID.String(),
			"error", err.Error())
		return err
	}

	s.logger.Info("[MESSAGE-SEND-1.6] message persisted successfully",
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
	if req.Type == model.ConversationTypeDirect && len(req.RecipientIDs) > 0 {
		recipientID := req.RecipientIDs[0]
		if existing, err := s.repo.FindDirectConversationByUsers(ctx, userID, recipientID); err != nil {
			return nil, err
		} else if existing != nil {
			return existing, nil
		}
	}

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
		if key.KeyID == 0 {
			return fmt.Errorf("one_time_pre_key[%d] is missing key_id", i)
		}
		if err := validateSignalKeyMaterial(fmt.Sprintf("one_time_pre_key[%d]", i), key.PublicKey); err != nil {
			return err
		}
		if _, exists := seen[string(key.PublicKey)]; exists {
			return fmt.Errorf("duplicate one_time_pre_key[%d] detected", i)
		}
		seen[string(key.PublicKey)] = struct{}{}
	}
	return nil
}

func BuildUserDeviceIdentity(userID uuid.UUID, req model.UploadDeviceKeysRequest) *model.UserDeviceIdentity {
	return &model.UserDeviceIdentity{
		UserID:              userID,
		DeviceID:            req.DeviceID,
		SignalDeviceID:      req.SignalDeviceID,
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
		DeviceID:              "",
		SignedPreKeyID:        identity.SignedPreKeyID,
		SignedPreKeyPublic:    identity.SignedPreKey,
		SignedPreKeySignature: identity.SignedPreKeySig,
		IdentityKey:           identity.PublicKey,
	}
	if identity.SignalDeviceID > 0 {
		bundle.DeviceID = strconv.FormatUint(uint64(identity.SignalDeviceID), 10)
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
	s.logger.Info("[DEVICE-OWNERSHIP-1] validating sender device binding",
		"user_id", userID.String(),
		"device_id", deviceID,
	)
	if userID == uuid.Nil || strings.TrimSpace(deviceID) == "" {
		s.logger.Error("[DEVICE-OWNERSHIP-1.1] missing user or device id",
			"user_id", userID.String(),
			"device_id", deviceID,
		)
		return ErrInvalidSession
	}
	identity, err := s.repo.GetDeviceIdentity(ctx, userID, deviceID)
	if err != nil || identity == nil {
		s.logger.Error("[DEVICE-OWNERSHIP-1.2] sender device not bound to authenticated user",
			"user_id", userID.String(),
			"device_id", deviceID,
			"error", err,
		)
		return ErrInvalidSession
	}
	s.logger.Info("[DEVICE-OWNERSHIP-1.3] sender device ownership verified",
		"user_id", userID.String(),
		"device_id", deviceID,
	)
	return nil
}

func (s *messagingService) UploadDeviceKeys(ctx context.Context, userID uuid.UUID, req model.UploadDeviceKeysRequest) error {
	req.Normalize()
	s.logger.Info("[DEVICE-KEYS-1] validating and persisting E2EE identity bundle",
		"user_id", userID.String(),
		"device_id", req.DeviceID,
		"pre_key_count", len(req.OneTimePreKeys),
	)
	if err := validateDeviceKeyUpload(req); err != nil {
		s.logger.Error("[DEVICE-KEYS-1.1] rejected malformed device keys",
			"user_id", userID.String(),
			"device_id", req.DeviceID,
			"error", err.Error(),
		)
		return apperr.ErrValidationFailed.WithMeta("detail", err.Error())
	}

	identity := BuildUserDeviceIdentity(userID, req)
	if err := s.repo.SaveDeviceIdentity(ctx, identity); err != nil {
		s.logger.Error("[DEVICE-KEYS-1.2] unable to persist device identity",
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
		for i, key := range req.OneTimePreKeys {
			keyID := key.KeyID
			if keyID == 0 {
				keyID = uint32(i + 1)
			}
			preKeys = append(preKeys, model.UserPreKey{
				DeviceID:  identity.ID,
				KeyID:     keyID,
				PublicKey: key.PublicKey,
			})
		}
		if err := s.repo.SavePreKeys(ctx, preKeys); err != nil {
			return err
		}
	}

	s.logger.Info("[DEVICE-KEYS-1.3] device identity bundle saved successfully",
		"user_id", userID.String(),
		"device_id", req.DeviceID,
		"one_time_pre_keys_stored", len(req.OneTimePreKeys),
	)
	return nil
}

func (s *messagingService) GetUserKeyBundle(ctx context.Context, targetUserID string) (*model.PreKeyBundleDto, error) {
	s.logger.Info("[KEY-BUNDLE-1] fetching pre-key bundle",
		"target_user_id", targetUserID,
	)
	targetID, err := uuid.Parse(targetUserID)
	if err != nil {
		s.logger.Error("[KEY-BUNDLE-1.1] invalid target user id",
			"target_user_id", targetUserID,
			"error", err.Error(),
		)
		return nil, ErrInvalidUUID
	}

	identity, err := s.repo.GetLatestDeviceIdentityForUser(ctx, targetID)
	if err != nil {
		s.logger.Error("[KEY-BUNDLE-1.2] no device identity found for target user",
			"target_user_id", targetUserID,
			"error", err.Error(),
		)
		return nil, ErrDeviceNotFound
	}

	preKey, err := s.repo.PopPreKey(ctx, identity.UserID, identity.DeviceID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		s.logger.Error("[KEY-BUNDLE-1.3] failed to consume pre-key",
			"target_user_id", targetUserID,
			"device_id", identity.DeviceID,
			"error", err.Error(),
		)
		return nil, err
	}
	if err != nil {
		s.logger.Info("[KEY-BUNDLE-1.4] no remaining one-time pre-key; using identity/signed key only",
			"target_user_id", targetUserID,
			"device_id", identity.DeviceID,
		)
		return BuildPreKeyBundle(identity, nil), nil
	}
	bundle := BuildPreKeyBundle(identity, preKey)
	s.logger.Info("[KEY-BUNDLE-1.5] pre-key bundle resolved",
		"target_user_id", targetUserID,
		"device_id", identity.DeviceID,
		"has_pre_key", bundle != nil && bundle.PreKeyID != nil,
	)
	return bundle, nil
}

func (s *messagingService) GetUserPreKeys(ctx context.Context, targetUserID string) (*model.UserPreKeysResponse, error) {
	s.logger.Info("[KEY-PREKEYS-1] resolving user pre-keys",
		"target_user_id", targetUserID,
	)
	bundle, err := s.GetUserKeyBundle(ctx, targetUserID)
	if err != nil {
		return nil, err
	}

	targetID, err := uuid.Parse(targetUserID)
	if err != nil {
		s.logger.Error("[KEY-PREKEYS-1.1] invalid target user id during pre-key response",
			"target_user_id", targetUserID,
			"error", err.Error(),
		)
		return nil, ErrInvalidUUID
	}

	identity, err := s.repo.GetLatestDeviceIdentityForUser(ctx, targetID)
	if err != nil {
		s.logger.Error("[KEY-PREKEYS-1.2] latest device identity missing",
			"target_user_id", targetUserID,
			"error", err.Error(),
		)
		return nil, ErrDeviceNotFound
	}

	res := &model.UserPreKeysResponse{
		UserID:          identity.UserID,
		DeviceID:        "",
		IdentityKey:     identity.PublicKey,
		SignedPreKey:    identity.SignedPreKey,
		SignedPreKeySig: identity.SignedPreKeySig,
		SignedPreKeyID:  identity.SignedPreKeyID,
	}
	if identity.SignalDeviceID > 0 {
		res.DeviceID = strconv.FormatUint(uint64(identity.SignalDeviceID), 10)
	}
	if bundle != nil {
		res.OneTimePreKey = bundle.PreKeyPublic
		if bundle.PreKeyID != nil {
			res.OneTimePreKeyID = *bundle.PreKeyID
		}
	}
	s.logger.Info("[KEY-PREKEYS-1.3] user pre-key response built",
		"target_user_id", targetUserID,
		"device_id", identity.DeviceID,
		"has_otk", res.OneTimePreKey != nil,
	)
	return res, nil
}

// #endregion
