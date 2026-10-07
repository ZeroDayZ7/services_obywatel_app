package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/citizen-docs/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type userDocumentRepository struct {
	db *gorm.DB
}

func NewUserDocumentRepository(db *gorm.DB) UserDocumentRepo {
	return &userDocumentRepository{db: db}
}

func computeUserDocumentAggregateHash(docs []model.CitizenDocument) string {
	if len(docs) == 0 {
		sum := sha256.Sum256([]byte("user-documents:empty"))
		return hex.EncodeToString(sum[:])
	}

	payload := make([]map[string]any, 0, len(docs))
	for _, doc := range docs {
		payload = append(payload, map[string]any{
			"id":                   doc.ID.String(),
			"document_type":        doc.DocumentType,
			"status":               string(doc.Status),
			"document_number_hash": doc.DocumentNumberHash,
			"version":              doc.Version,
			"updated_at":           doc.UpdatedAt.UTC().Format(time.RFC3339Nano),
		})
	}

	serialized, err := json.Marshal(payload)
	if err != nil {
		serialized = []byte("user-documents:fallback")
	}
	sum := sha256.Sum256(serialized)
	return hex.EncodeToString(sum[:])
}

func (r *userDocumentRepository) recomputeUserDocumentState(tx *gorm.DB, userID uuid.UUID) error {
	var docs []model.CitizenDocument
	if err := tx.Where("user_id = ? AND deleted_at IS NULL", userID).
		Order("version DESC, updated_at DESC, created_at DESC, id ASC").
		Find(&docs).Error; err != nil {
		return err
	}

	state := model.UserDocumentState{
		UserID:        userID,
		DocumentCount: len(docs),
		AggregateHash: computeUserDocumentAggregateHash(docs),
	}
	if len(docs) > 0 {
		maxVersion := uint64(0)
		for _, doc := range docs {
			if doc.Version > maxVersion {
				maxVersion = doc.Version
			}
		}
		state.StateVersion = maxVersion + 1
	}

	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"state_version", "aggregate_hash", "document_count", "updated_at"}),
	}).Create(&state).Error
}

func (r *userDocumentRepository) CreateDocument(ctx context.Context, doc *model.CitizenDocument) error {
	if doc == nil {
		return fmt.Errorf("document is nil")
	}
	if doc.Version == 0 {
		doc.Version = 1
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(doc).Error; err != nil {
			return err
		}
		return r.recomputeUserDocumentState(tx, doc.UserID)
	})
}

func (r *userDocumentRepository) GetDocumentByID(ctx context.Context, id uuid.UUID) (*model.CitizenDocument, error) {
	var doc model.CitizenDocument
	if err := r.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&doc).Error; err != nil {
		return nil, err
	}
	return &doc, nil
}

func (r *userDocumentRepository) GetDocumentByDocumentNumberHash(ctx context.Context, documentNumberHash string) (*model.CitizenDocument, error) {
	var doc model.CitizenDocument
	if err := r.db.WithContext(ctx).
		Where("document_number_hash = ? AND deleted_at IS NULL", documentNumberHash).
		First(&doc).Error; err != nil {
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
		Order("issued_at DESC, created_at DESC, version DESC").
		Find(&docs).Error; err != nil {
		log.ErrorMap("[userDocumentRepository.GetDocumentsByUserID] 2. DB query failed", map[string]any{"user_id": userID.String(), "err": err.Error()})
		return nil, err
	}

	log.InfoMap("[userDocumentRepository.GetDocumentsByUserID] 3. Document query finished", map[string]any{"user_id": userID.String(), "count": len(docs)})
	return docs, nil
}

func (r *userDocumentRepository) GetUserDocumentState(ctx context.Context, userID uuid.UUID) (*model.UserDocumentState, error) {
	var state model.UserDocumentState
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&state).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &state, nil
}

func (r *userDocumentRepository) UpdateDocumentStatus(ctx context.Context, id uuid.UUID, status model.DocumentStatus) (*model.CitizenDocument, error) {
	var doc model.CitizenDocument
	if err := r.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&doc).Error; err != nil {
		return nil, err
	}

	doc.Status = status
	doc.Version = doc.Version + 1
	if err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&doc).Error; err != nil {
			return err
		}
		return r.recomputeUserDocumentState(tx, doc.UserID)
	}); err != nil {
		return nil, err
	}
	return &doc, nil
}
