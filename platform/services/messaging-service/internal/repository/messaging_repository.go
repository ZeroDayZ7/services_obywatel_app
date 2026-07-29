package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/services/messaging-service/internal/model"
	"gorm.io/gorm"
)

type MessagingRepository interface {
	// Delta Sync (Contacts & Messages)
	GetContactsSinceVersion(ctx context.Context, ownerID uuid.UUID, sinceVersion uint64) ([]model.Contact, error)
	GetMessagesSinceVersion(ctx context.Context, userID uuid.UUID, sinceVersion uint64) ([]model.Message, error)

	// Messaging & E2EE
	CreateMessage(ctx context.Context, msg *model.Message) error
	GetDeviceIdentity(ctx context.Context, userID uuid.UUID, deviceID string) (*model.UserDeviceIdentity, error)

	// Contacts
	UpsertContact(ctx context.Context, contact *model.Contact) error
}

type messagingRepository struct {
	db *gorm.DB
}

func NewMessagingRepository(db *gorm.DB) MessagingRepository {
	return &messagingRepository{db: db}
}

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
		Joins("JOIN conversation_members cm ON cm.conversation_id = messages.conversation_id").
		Where("cm.user_id = ? AND messages.version > ?", userID, sinceVersion).
		Order("messages.version ASC").
		Find(&messages).Error
	return messages, err
}

func (r *messagingRepository) CreateMessage(ctx context.Context, msg *model.Message) error {
	return r.db.WithContext(ctx).Create(msg).Error
}

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

func (r *messagingRepository) UpsertContact(ctx context.Context, contact *model.Contact) error {
	return r.db.WithContext(ctx).Save(contact).Error
}
