package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/citizen-docs/internal/model"
	"gorm.io/gorm"
)

type userDocumentRepository struct {
	db *gorm.DB
}

func NewUserDocumentRepository(db *gorm.DB) UserDocumentRepo {
	return &userDocumentRepository{db: db}
}

func (r *userDocumentRepository) CreateDocument(ctx context.Context, doc *model.CitizenDocument) error {
	if doc == nil {
		return fmt.Errorf("document is nil")
	}
	return r.db.WithContext(ctx).Create(doc).Error
}

func (r *userDocumentRepository) GetDocumentByID(ctx context.Context, id uuid.UUID) (*model.CitizenDocument, error) {
	var doc model.CitizenDocument
	if err := r.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&doc).Error; err != nil {
		return nil, err
	}
	return &doc, nil
}

func (r *userDocumentRepository) GetDocumentsByUserID(ctx context.Context, userID uuid.UUID) ([]model.CitizenDocument, error) {
	log := shared.GetLogger()
	log.InfoMap("[userDocumentRepository.GetDocumentsByUserID] 1. Executing document lookup query", map[string]any{"user_id": userID.String()})

	var docs []model.CitizenDocument
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Order("issued_at DESC, created_at DESC").
		Find(&docs).Error; err != nil {
		log.ErrorMap("[userDocumentRepository.GetDocumentsByUserID] 2. DB query failed", map[string]any{"user_id": userID.String(), "err": err.Error()})
		return nil, err
	}

	log.InfoMap("[userDocumentRepository.GetDocumentsByUserID] 3. Document query finished", map[string]any{"user_id": userID.String(), "count": len(docs)})
	return docs, nil
}

func (r *userDocumentRepository) UpdateDocumentStatus(ctx context.Context, id uuid.UUID, status model.DocumentStatus) (*model.CitizenDocument, error) {
	var doc model.CitizenDocument
	if err := r.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&doc).Error; err != nil {
		return nil, err
	}

	doc.Status = status
	if err := r.db.WithContext(ctx).Save(&doc).Error; err != nil {
		return nil, err
	}
	return &doc, nil
}
