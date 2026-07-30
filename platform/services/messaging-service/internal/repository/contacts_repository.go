package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/services/messaging-service/internal/model"
	"gorm.io/gorm"
)

type ContactsRepository interface {
	GetContactsByUserID(ctx context.Context, userID uuid.UUID) ([]model.Contact, error)
	GetContactByOwnerAndTarget(ctx context.Context, ownerID, targetID uuid.UUID) (*model.Contact, error)
	GetContactByID(ctx context.Context, id uuid.UUID) (*model.Contact, error)
	CreateContact(ctx context.Context, contact *model.Contact) error
	UpdateContactStatus(ctx context.Context, contactID uuid.UUID, status model.ContactStatus) error
	CreateSymmetricContact(ctx context.Context, ownerID, targetID uuid.UUID, status model.ContactStatus) error
}

type contactsRepository struct {
	db *gorm.DB
}

func NewContactsRepository(db *gorm.DB) ContactsRepository {
	return &contactsRepository{db: db}
}

func (r *contactsRepository) GetContactsByUserID(ctx context.Context, userID uuid.UUID) ([]model.Contact, error) {
	var contacts []model.Contact
	err := r.db.WithContext(ctx).
		Where("owner_id = ?", userID).
		Find(&contacts).Error
	return contacts, err
}

func (r *contactsRepository) GetContactByOwnerAndTarget(ctx context.Context, ownerID, targetID uuid.UUID) (*model.Contact, error) {
	var contact model.Contact
	err := r.db.WithContext(ctx).
		Where("owner_id = ? AND contact_id = ?", ownerID, targetID).
		First(&contact).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &contact, err
}

func (r *contactsRepository) GetContactByID(ctx context.Context, id uuid.UUID) (*model.Contact, error) {
	var contact model.Contact
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&contact).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &contact, err
}

func (r *contactsRepository) CreateContact(ctx context.Context, contact *model.Contact) error {
	return r.db.WithContext(ctx).Create(contact).Error
}

func (r *contactsRepository) UpdateContactStatus(ctx context.Context, contactID uuid.UUID, status model.ContactStatus) error {
	// Podbijamy wersję (version = version + 1) na potrzeby synchronizacji różnicowej (Delta Sync)
	return r.db.WithContext(ctx).
		Model(&model.Contact{}).
		Where("id = ?", contactID).
		Updates(map[string]interface{}{
			"status":  status,
			"version": gorm.Expr("version + 1"),
		}).Error
}

func (r *contactsRepository) CreateSymmetricContact(ctx context.Context, ownerID, targetID uuid.UUID, status model.ContactStatus) error {
	// Tworzy relację powrotną w transakcji, jeśli jeszcze nie istnieje
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&model.Contact{}).
			Where("owner_id = ? AND contact_id = ?", ownerID, targetID).
			Count(&count).Error; err != nil {
			return err
		}

		if count == 0 {
			symmetricContact := model.Contact{
				OwnerID:   ownerID,
				ContactID: targetID,
				Status:    status,
				Version:   1,
			}
			return tx.Create(&symmetricContact).Error
		}

		return tx.Model(&model.Contact{}).
			Where("owner_id = ? AND contact_id = ?", ownerID, targetID).
			Updates(map[string]interface{}{
				"status":  status,
				"version": gorm.Expr("version + 1"),
			}).Error
	})
}
