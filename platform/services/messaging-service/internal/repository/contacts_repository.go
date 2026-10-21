package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/services/messaging-service/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ContactsRepository interface {
	GetContactsByUserID(ctx context.Context, userID uuid.UUID) ([]model.Contact, error)
	GetContactByOwnerAndTarget(ctx context.Context, ownerID, targetID uuid.UUID) (*model.Contact, error)
	GetContactByID(ctx context.Context, id uuid.UUID) (*model.Contact, error)
	CreateContact(ctx context.Context, contact *model.Contact) error
	UpdateContactStatus(ctx context.Context, contactID uuid.UUID, status model.ContactStatus) error
	CancelOutgoingRequest(ctx context.Context, ownerID, contactID uuid.UUID) error
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
		Where("deleted_at IS NULL AND (owner_id = ? OR contact_id = ?)", userID, userID).
		Order("created_at DESC").
		Find(&contacts).Error
	return contacts, err
}

func (r *contactsRepository) GetContactByOwnerAndTarget(ctx context.Context, ownerID, targetID uuid.UUID) (*model.Contact, error) {
	var contact model.Contact
	err := r.db.WithContext(ctx).
		Where("deleted_at IS NULL AND owner_id = ? AND contact_id = ?", ownerID, targetID).
		First(&contact).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		var reverse model.Contact
		reverseErr := r.db.WithContext(ctx).
			Where("deleted_at IS NULL AND owner_id = ? AND contact_id = ?", targetID, ownerID).
			First(&reverse).Error
		if errors.Is(reverseErr, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		if reverseErr != nil {
			return nil, reverseErr
		}
		return &reverse, nil
	}
	if err != nil {
		return nil, err
	}
	return &contact, nil
}

func (r *contactsRepository) GetContactByID(ctx context.Context, id uuid.UUID) (*model.Contact, error) {
	var contact model.Contact
	err := r.db.WithContext(ctx).
		Where("deleted_at IS NULL AND id = ?", id).
		First(&contact).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &contact, err
}

func (r *contactsRepository) CreateContact(ctx context.Context, contact *model.Contact) error {
	if contact == nil {
		return errors.New("contact is nil")
	}
	if contact.ID == uuid.Nil {
		contact.ID = uuid.New()
	}
	return r.db.WithContext(ctx).Create(contact).Error
}

func (r *contactsRepository) UpdateContactStatus(ctx context.Context, contactID uuid.UUID, status model.ContactStatus) error {
	// Podbijamy wersję (version = version + 1) na potrzeby synchronizacji różnicowej (Delta Sync)
	return r.db.WithContext(ctx).
		Model(&model.Contact{}).
		Where("deleted_at IS NULL AND id = ?", contactID).
		Updates(map[string]interface{}{
			"status":  status,
			"version": gorm.Expr("version + 1"),
		}).Error
}

func (r *contactsRepository) CancelOutgoingRequest(ctx context.Context, ownerID, contactID uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing model.Contact
		err := tx.Model(&model.Contact{}).
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("deleted_at IS NULL AND id = ? AND owner_id = ? AND status = ? AND direction = ?", contactID, ownerID, model.ContactStatusPending, model.ContactDirectionOutgoing).
			First(&existing).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return gorm.ErrRecordNotFound
			}
			return err
		}

		now := time.Now()
		result := tx.Model(&model.Contact{}).
			Where("deleted_at IS NULL AND id = ? AND owner_id = ? AND status = ? AND direction = ?", contactID, ownerID, model.ContactStatusPending, model.ContactDirectionOutgoing).
			Updates(map[string]interface{}{
				"deleted_at": now,
				"status":     model.ContactStatusBlocked,
				"version":    gorm.Expr("version + 1"),
				"sync_state": model.ContactSyncStatePendingDelete,
				"updated_at": now,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
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
				ID:        uuid.New(),
				OwnerID:   ownerID,
				ContactID: targetID,
				Status:    status,
				Direction: model.ContactDirectionIncoming,
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
