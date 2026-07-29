package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/services/citizen-docs/internal/model"
	"gorm.io/gorm"
)

type userDocumentRepository struct {
	db *gorm.DB
}

func NewUserDocumentRepository(db *gorm.DB) UserDocumentRepo {
	return &userDocumentRepository{db: db}
}

// #region CREATE
func (r *userDocumentRepository) Create(ctx context.Context, doc *model.UserDocument) error {
	return r.db.WithContext(ctx).Create(doc).Error
}

// #region READ BY PROFILE ID
func (r *userDocumentRepository) GetByProfileID(ctx context.Context, profileID uuid.UUID) ([]model.UserDocument, error) {
	var docs []model.UserDocument

	err := r.db.WithContext(ctx).
		Where("profile_id = ? AND deleted_at IS NULL", profileID).
		Find(&docs).Error

	return docs, err
}

// #region DELTA SYNC (GET SINCE VERSION)
func (r *userDocumentRepository) GetSinceVersion(ctx context.Context, profileID uuid.UUID, sinceVersion uint64) ([]model.UserDocument, error) {
	var docs []model.UserDocument

	err := r.db.WithContext(ctx).
		Where("profile_id = ? AND version > ? AND deleted_at IS NULL", profileID, sinceVersion).
		Order("version ASC").
		Find(&docs).Error

	return docs, err
}
