package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/services/messaging-service/internal/model"
	"gorm.io/gorm"
)

type PersonalSpaceRepository interface {
	GetOrCreateSpace(ctx context.Context, ownerID uuid.UUID) (*model.PersonalSpace, error)
	GetSpace(ctx context.Context, ownerID uuid.UUID) (*model.PersonalSpace, error)
	ListItems(ctx context.Context, ownerID uuid.UUID, sinceVersion uint64) ([]model.PersonalSpaceItem, error)
	GetItemByID(ctx context.Context, ownerID, itemID uuid.UUID) (*model.PersonalSpaceItem, error)
	CreateItem(ctx context.Context, item *model.PersonalSpaceItem) error
	UpdateItem(ctx context.Context, item *model.PersonalSpaceItem) error
	DeleteItem(ctx context.Context, ownerID, itemID uuid.UUID) error
}

type personalSpaceRepository struct {
	db *gorm.DB
}

func NewPersonalSpaceRepository(db *gorm.DB) PersonalSpaceRepository {
	return &personalSpaceRepository{db: db}
}

func (r *personalSpaceRepository) GetOrCreateSpace(ctx context.Context, ownerID uuid.UUID) (*model.PersonalSpace, error) {
	if ownerID == uuid.Nil {
		return nil, errors.New("owner id is required")
	}

	var space model.PersonalSpace
	if err := r.db.WithContext(ctx).Where("owner_id = ?", ownerID).First(&space).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		space = model.PersonalSpace{
			ID:      uuid.New(),
			OwnerID: ownerID,
			Title:   "Notatki dla siebie",
		}
		if err := r.db.WithContext(ctx).Create(&space).Error; err != nil {
			return nil, err
		}
	}
	return &space, nil
}

func (r *personalSpaceRepository) GetSpace(ctx context.Context, ownerID uuid.UUID) (*model.PersonalSpace, error) {
	if ownerID == uuid.Nil {
		return nil, errors.New("owner id is required")
	}
	var space model.PersonalSpace
	if err := r.db.WithContext(ctx).Where("owner_id = ?", ownerID).First(&space).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &space, nil
}

func (r *personalSpaceRepository) ListItems(ctx context.Context, ownerID uuid.UUID, sinceVersion uint64) ([]model.PersonalSpaceItem, error) {
	var items []model.PersonalSpaceItem
	if err := r.db.WithContext(ctx).
		Where("owner_id = ? AND sync_version > ?", ownerID, sinceVersion).
		Order("sync_version ASC, updated_at ASC").
		Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *personalSpaceRepository) GetItemByID(ctx context.Context, ownerID, itemID uuid.UUID) (*model.PersonalSpaceItem, error) {
	var item model.PersonalSpaceItem
	if err := r.db.WithContext(ctx).Where("owner_id = ? AND id = ?", ownerID, itemID).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (r *personalSpaceRepository) CreateItem(ctx context.Context, item *model.PersonalSpaceItem) error {
	if item == nil {
		return errors.New("item is nil")
	}
	if item.ID == uuid.Nil {
		item.ID = uuid.New()
	}
	if item.Status == "" {
		item.Status = model.PersonalSpaceItemStatusActive
	}
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *personalSpaceRepository) UpdateItem(ctx context.Context, item *model.PersonalSpaceItem) error {
	if item == nil {
		return errors.New("item is nil")
	}
	return r.db.WithContext(ctx).Save(item).Error
}

func (r *personalSpaceRepository) DeleteItem(ctx context.Context, ownerID, itemID uuid.UUID) error {
	return r.db.WithContext(ctx).Where("owner_id = ? AND id = ?", ownerID, itemID).Delete(&model.PersonalSpaceItem{}).Error
}
