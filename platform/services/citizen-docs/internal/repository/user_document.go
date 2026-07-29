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

// #region READ BY USER ID
func (r *userDocumentRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]model.UserDocument, error) {
	var docs []model.UserDocument

	err := r.db.WithContext(ctx).
		Table("user_documents").
		Joins("JOIN citizen_profiles ON citizen_profiles.id = user_documents.profile_id").
		Where("citizen_profiles.user_id = ? AND user_documents.deleted_at IS NULL", userID).
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
