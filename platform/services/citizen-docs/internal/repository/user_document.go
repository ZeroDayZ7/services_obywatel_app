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

func (r *userDocumentRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]model.UserDocument, error) {
	var docs []model.UserDocument
	
	// Zapytanie podrzędne lub JOIN łączące user_documents z citizen_profiles
	err := r.db.WithContext(ctx).
		Table("user_documents").
		Joins("JOIN citizen_profiles ON citizen_profiles.id = user_documents.profile_id").
		Where("citizen_profiles.user_id = ? AND user_documents.deleted_at IS NULL", userID).
		Find(&docs).Error

	return docs, err
}
