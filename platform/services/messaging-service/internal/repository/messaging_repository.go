package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/services/messaging-service/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type MessagingRepository interface {
	// Delta Sync (Contacts & Messages)
	GetContactsSinceVersion(ctx context.Context, ownerID uuid.UUID, sinceVersion uint64) ([]model.Contact, error)
	GetMessagesSinceVersion(ctx context.Context, userID uuid.UUID, sinceVersion uint64) ([]model.Message, error)

	// Messaging & Conversations
	CreateMessage(ctx context.Context, msg *model.Message) error
	GetMessageByIdempotencyKey(ctx context.Context, idempotencyKey string) (*model.Message, error)
	GetMessagesByConversation(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID, limit, offset int) ([]model.Message, error)
	GetConversations(ctx context.Context, userID uuid.UUID) ([]model.Conversation, error)
	GetConversationByID(ctx context.Context, userID, conversationID uuid.UUID) (*model.Conversation, error)
	FindDirectConversationByUsers(ctx context.Context, userA, userB uuid.UUID) (*model.Conversation, error)
	CreateConversation(ctx context.Context, conv *model.Conversation) error
	UpdateLastReadSequence(ctx context.Context, userID, conversationID uuid.UUID, sequence uint64) error

	// E2EE Keys & Identity
	GetDeviceIdentity(ctx context.Context, userID uuid.UUID, deviceID string) (*model.UserDeviceIdentity, error)
	GetLatestDeviceIdentityForUser(ctx context.Context, userID uuid.UUID) (*model.UserDeviceIdentity, error)
	SaveDeviceIdentity(ctx context.Context, identity *model.UserDeviceIdentity) error
	SavePreKeys(ctx context.Context, keys []model.UserPreKey) error
	PopPreKey(ctx context.Context, userID uuid.UUID, deviceID string) (*model.UserPreKey, error)

	// Contacts
	UpsertContact(ctx context.Context, contact *model.Contact) error

	// Messaging activation
	GetActivationStatus(ctx context.Context, userID uuid.UUID) (*model.MessagingActivation, error)
	GetOrCreateActivation(ctx context.Context, userID uuid.UUID, deviceID string) (*model.MessagingActivation, error)
	SaveActivation(ctx context.Context, activation *model.MessagingActivation) error
}

type messagingRepository struct {
	db *gorm.DB
}

func NewMessagingRepository(db *gorm.DB) MessagingRepository {
	return &messagingRepository{db: db}
}

// #region DeltaSync
func (r *messagingRepository) GetContactsSinceVersion(ctx context.Context, ownerID uuid.UUID, sinceVersion uint64) ([]model.Contact, error) {
	var contacts []model.Contact
	err := r.db.WithContext(ctx).
		Where("owner_id = ? AND version > ?", ownerID, sinceVersion).
		Order("version ASC").
		Find(&contacts).Error
	return contacts, err
}

func (r *messagingRepository) GetMessagesSinceVersion(ctx context.Context, userID uuid.UUID, sinceVersion uint64) ([]model.Message, error) {
	var messages []model.Message
	err := r.db.WithContext(ctx).
		Select("messages.*").
		Joins("JOIN conversation_members cm ON cm.conversation_id = messages.conversation_id").
		Where("cm.user_id = ? AND messages.version > ?", userID, sinceVersion).
		Order("messages.version ASC").
		Find(&messages).Error
	return messages, err
}

// #endregion

// #region MessagingAndConversations
func (r *messagingRepository) CreateMessage(ctx context.Context, msg *model.Message) error {
	if msg == nil {
		return errors.New("message is nil")
	}
	if strings.TrimSpace(msg.IdempotencyKey) == "" {
		msg.IdempotencyKey = uuid.NewString()
	}

	if existing, err := r.GetMessageByIdempotencyKey(ctx, msg.IdempotencyKey); err != nil {
		return err
	} else if existing != nil {
		return gorm.ErrDuplicatedKey
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var lastSeq uint64
		err := tx.Model(&model.Conversation{}).
			Where("id = ?", msg.ConversationID).
			Select("COALESCE(last_sequence, 0)").
			Scan(&lastSeq).Error
		if err != nil {
			return err
		}

		msg.Sequence = lastSeq + 1

		var lastVersion uint64
		if err := tx.Model(&model.Message{}).
			Select("COALESCE(MAX(version), 0)").
			Scan(&lastVersion).Error; err != nil {
			return err
		}
		msg.Version = lastVersion + 1

		if err := tx.Create(msg).Error; err != nil {
			return err
		}

		return tx.Model(&model.Conversation{}).
			Where("id = ?", msg.ConversationID).
			Update("last_sequence", msg.Sequence).Error
	})
}

func (r *messagingRepository) GetMessageByIdempotencyKey(ctx context.Context, idempotencyKey string) (*model.Message, error) {
	if idempotencyKey == "" {
		return nil, nil
	}

	var msg model.Message
	if err := r.db.WithContext(ctx).Where("idempotency_key = ?", idempotencyKey).First(&msg).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &msg, nil
}

func (r *messagingRepository) GetMessagesByConversation(ctx context.Context, userID, conversationID uuid.UUID, limit, offset int) ([]model.Message, error) {
	var messages []model.Message
	err := r.db.WithContext(ctx).
		Select("messages.*").
		Joins("JOIN conversation_members cm ON cm.conversation_id = messages.conversation_id").
		Where("cm.user_id = ? AND messages.conversation_id = ?", userID, conversationID).
		Order("messages.sequence DESC").
		Limit(limit).
		Offset(offset).
		Find(&messages).Error
	return messages, err
}

func (r *messagingRepository) GetConversations(ctx context.Context, userID uuid.UUID) ([]model.Conversation, error) {
	var conversations []model.Conversation
	err := r.db.WithContext(ctx).
		Joins("JOIN conversation_members cm ON cm.conversation_id = conversations.id").
		Where("cm.user_id = ?", userID).
		Preload("Members").
		Find(&conversations).Error
	return conversations, err
}

func (r *messagingRepository) GetConversationByID(ctx context.Context, userID, conversationID uuid.UUID) (*model.Conversation, error) {
	var conv model.Conversation
	err := r.db.WithContext(ctx).
		Joins("JOIN conversation_members cm ON cm.conversation_id = conversations.id").
		Where("cm.user_id = ? AND conversations.id = ?", userID, conversationID).
		Preload("Members").
		First(&conv).Error
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

func (r *messagingRepository) FindDirectConversationByUsers(ctx context.Context, userA, userB uuid.UUID) (*model.Conversation, error) {
	if userA == uuid.Nil || userB == uuid.Nil || userA == userB {
		return nil, nil
	}

	var conv model.Conversation
	err := r.db.WithContext(ctx).
		Table("conversations").
		Joins("JOIN conversation_members cm_a ON cm_a.conversation_id = conversations.id").
		Joins("JOIN conversation_members cm_b ON cm_b.conversation_id = conversations.id").
		Where("conversations.type = ?", model.ConversationTypeDirect).
		Where("conversations.deleted_at IS NULL AND cm_a.deleted_at IS NULL AND cm_b.deleted_at IS NULL").
		Where("(cm_a.user_id = ? AND cm_b.user_id = ?) OR (cm_a.user_id = ? AND cm_b.user_id = ?)", userA, userB, userB, userA).
		Preload("Members").
		First(&conv).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &conv, nil
}

func (r *messagingRepository) CreateConversation(ctx context.Context, conv *model.Conversation) error {
	return r.db.WithContext(ctx).Create(conv).Error
}

func (r *messagingRepository) UpdateLastReadSequence(ctx context.Context, userID, conversationID uuid.UUID, sequence uint64) error {
	return r.db.WithContext(ctx).
		Model(&model.ConversationMember{}).
		Where("user_id = ? AND conversation_id = ?", userID, conversationID).
		Update("last_read_sequence", sequence).Error
}

func (r *messagingRepository) GetActivationStatus(ctx context.Context, userID uuid.UUID) (*model.MessagingActivation, error) {
	var activation model.MessagingActivation
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC").First(&activation).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &model.MessagingActivation{
				UserID: userID,
				Status: model.ActivationStatusNotStarted,
			}, nil
		}
		return nil, err
	}
	return &activation, nil
}

func (r *messagingRepository) GetOrCreateActivation(ctx context.Context, userID uuid.UUID, deviceID string) (*model.MessagingActivation, error) {
	activation, err := r.GetActivationStatus(ctx, userID)
	if err != nil {
		return nil, err
	}
	if activation.ID == uuid.Nil {
		activation = &model.MessagingActivation{
			UserID:   userID,
			DeviceID: deviceID,
			Status:   model.ActivationStatusNotStarted,
		}
		if err := r.db.WithContext(ctx).Create(activation).Error; err != nil {
			return nil, err
		}
	}
	if deviceID != "" && activation.DeviceID == "" {
		activation.DeviceID = deviceID
		if err := r.SaveActivation(ctx, activation); err != nil {
			return nil, err
		}
	}
	return activation, nil
}

func (r *messagingRepository) SaveActivation(ctx context.Context, activation *model.MessagingActivation) error {
	if activation == nil {
		return nil
	}
	return r.db.WithContext(ctx).Save(activation).Error
}

// #endregion

// #region E2EE
func (r *messagingRepository) GetDeviceIdentity(ctx context.Context, userID uuid.UUID, deviceID string) (*model.UserDeviceIdentity, error) {
	var identity model.UserDeviceIdentity
	query := r.db.WithContext(ctx).Where("user_id = ?", userID)
	if deviceID != "" {
		query = query.Where("device_id = ?", deviceID)
	}

	err := query.Order("created_at DESC").First(&identity).Error
	if err != nil {
		return nil, err
	}
	return &identity, nil
}

func (r *messagingRepository) GetLatestDeviceIdentityForUser(ctx context.Context, userID uuid.UUID) (*model.UserDeviceIdentity, error) {
	return r.GetDeviceIdentity(ctx, userID, "")
}

func (r *messagingRepository) SaveDeviceIdentity(ctx context.Context, identity *model.UserDeviceIdentity) error {
	if identity == nil {
		return nil
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing model.UserDeviceIdentity
		err := tx.Unscoped().
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND device_id = ?", identity.UserID, identity.DeviceID).
			Order("created_at DESC").
			First(&existing).Error
		if err == nil {
			existing.RegistrationID = identity.RegistrationID
			existing.PublicKey = identity.PublicKey
			existing.SignedPreKey = identity.SignedPreKey
			existing.SignedPreKeySig = identity.SignedPreKeySig
			existing.SignedPreKeyID = identity.SignedPreKeyID
			existing.OneTimePreKeysCount = identity.OneTimePreKeysCount
			existing.DeletedAt = gorm.DeletedAt{}
			return tx.Unscoped().Save(&existing).Error
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		if err := tx.Create(identity).Error; err != nil {
			if !isUniqueConstraintError(err) {
				return err
			}

			var competing model.UserDeviceIdentity
			if err2 := tx.Unscoped().
				Where("user_id = ? AND device_id = ?", identity.UserID, identity.DeviceID).
				Order("created_at DESC").
				First(&competing).Error; err2 != nil {
				return err2
			}

			competing.RegistrationID = identity.RegistrationID
			competing.PublicKey = identity.PublicKey
			competing.SignedPreKey = identity.SignedPreKey
			competing.SignedPreKeySig = identity.SignedPreKeySig
			competing.SignedPreKeyID = identity.SignedPreKeyID
			competing.OneTimePreKeysCount = identity.OneTimePreKeysCount
			competing.DeletedAt = gorm.DeletedAt{}
			return tx.Unscoped().Save(&competing).Error
		}
		return nil
	})
}

func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "duplicate key") ||
		strings.Contains(msg, "unique constraint") ||
		strings.Contains(msg, "UNIQUE constraint failed") ||
		strings.Contains(msg, "constraint") && strings.Contains(msg, "already exists")
}

func (r *messagingRepository) SavePreKeys(ctx context.Context, keys []model.UserPreKey) error {
	if len(keys) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&keys).Error
}

func (r *messagingRepository) PopPreKey(ctx context.Context, userID uuid.UUID, deviceID string) (*model.UserPreKey, error) {
	identity, err := r.GetDeviceIdentity(ctx, userID, deviceID)
	if err != nil {
		return nil, err
	}

	var key model.UserPreKey
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("device_id = ?", identity.ID).Order("key_id ASC").First(&key).Error; err != nil {
			return err
		}
		return tx.Delete(&key).Error
	})
	if err != nil {
		return nil, err
	}
	return &key, nil
}

// #endregion

// #region Contacts
func (r *messagingRepository) UpsertContact(ctx context.Context, contact *model.Contact) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "owner_id"}, {Name: "contact_id"}},
			DoUpdates: clause.Assignments(map[string]any{
				"status":          contact.Status,
				"encrypted_alias": contact.EncryptedAlias,
				"version":         gorm.Expr("version + 1"),
				"updated_at":      gorm.Expr("NOW()"),
			}),
		}).
		Create(contact).Error
}

// #endregion
