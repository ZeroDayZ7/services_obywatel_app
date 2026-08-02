package repository

import (
	"context"

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
	GetMessagesByConversation(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID, limit, offset int) ([]model.Message, error)
	GetConversations(ctx context.Context, userID uuid.UUID) ([]model.Conversation, error)
	GetConversationByID(ctx context.Context, userID, conversationID uuid.UUID) (*model.Conversation, error)
	CreateConversation(ctx context.Context, conv *model.Conversation) error
	UpdateLastReadSequence(ctx context.Context, userID, conversationID uuid.UUID, sequence uint64) error

	// E2EE Keys & Identity
	GetDeviceIdentity(ctx context.Context, userID uuid.UUID, deviceID string) (*model.UserDeviceIdentity, error)
	SaveDeviceIdentity(ctx context.Context, identity *model.UserDeviceIdentity) error
	SavePreKeys(ctx context.Context, keys []model.UserPreKey) error
	PopPreKey(ctx context.Context, userID uuid.UUID) (*model.UserPreKey, error)

	// Contacts
	UpsertContact(ctx context.Context, contact *model.Contact) error
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

		if err := tx.Create(msg).Error; err != nil {
			return err
		}

		return tx.Model(&model.Conversation{}).
			Where("id = ?", msg.ConversationID).
			Update("last_sequence", msg.Sequence).Error
	})
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

func (r *messagingRepository) CreateConversation(ctx context.Context, conv *model.Conversation) error {
	return r.db.WithContext(ctx).Create(conv).Error
}

func (r *messagingRepository) UpdateLastReadSequence(ctx context.Context, userID, conversationID uuid.UUID, sequence uint64) error {
	return r.db.WithContext(ctx).
		Model(&model.ConversationMember{}).
		Where("user_id = ? AND conversation_id = ?", userID, conversationID).
		Update("last_read_sequence", sequence).Error
}

// #endregion

// #region E2EE
func (r *messagingRepository) GetDeviceIdentity(ctx context.Context, userID uuid.UUID, deviceID string) (*model.UserDeviceIdentity, error) {
	var identity model.UserDeviceIdentity
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND device_id = ?", userID, deviceID).
		First(&identity).Error
	if err != nil {
		return nil, err
	}
	return &identity, nil
}

func (r *messagingRepository) SaveDeviceIdentity(ctx context.Context, identity *model.UserDeviceIdentity) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "device_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"public_key", "signed_pre_key", "signed_pre_key_sig", "signed_pre_key_id", "updated_at"}),
		}).
		Create(identity).Error
}

func (r *messagingRepository) SavePreKeys(ctx context.Context, keys []model.UserPreKey) error {
	if len(keys) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&keys).Error
}

func (r *messagingRepository) PopPreKey(ctx context.Context, deviceID uuid.UUID) (*model.UserPreKey, error) {
	var key model.UserPreKey
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("device_id = ?", deviceID).First(&key).Error; err != nil {
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
