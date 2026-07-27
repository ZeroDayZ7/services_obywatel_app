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

// #region READ
func (r *userDocumentRepository) GetByProfileID(ctx context.Context, profileID uuid.UUID) ([]model.UserDocument, error) {
	var docs []model.UserDocument
	err := r.db.WithContext(ctx).Where("profile_id = ?", profileID).Find(&docs).Error
	return docs, err
}
