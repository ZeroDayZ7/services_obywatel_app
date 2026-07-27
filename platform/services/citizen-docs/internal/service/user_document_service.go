// platform/services/citizen-docs/internal/service/user_document_service.go

package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/pkg/viper"
	"github.com/zerodayz7/platform/services/citizen-docs/internal/model"
	"github.com/zerodayz7/platform/services/citizen-docs/internal/repository"
)

type UserDocumentService struct {
	repo   repository.UserDocumentRepo
	cfg    *viper.Config
	logger *shared.Logger
}

func NewUserDocumentService(repo repository.UserDocumentRepo, cfg *viper.Config, logger *shared.Logger) *UserDocumentService {
	return &UserDocumentService{
		repo:   repo,
		cfg:    cfg,
		logger: logger,
	}
}

// CreateDocument zaktualizowany o uuid.UUID dla profileID
func (s *UserDocumentService) CreateDocument(
	ctx context.Context,
	meta *model.DocumentMeta,
	front []byte,
	back []byte,
	profileID uuid.UUID,
	docType model.DocumentType,
) error {
	encryptionKey := []byte(s.cfg.Internal.EncryptionKey)

	// Szyfrowanie metadanych (JSON)
	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("failed to marshal document meta: %w", err)
	}

	encMeta, err := shared.Encrypt(metaBytes, encryptionKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt document meta: %w", err)
	}

	// Szyfrowanie plików binarnych
	var encFront, encBack []byte
	if len(front) > 0 {
		encFront, err = shared.Encrypt(front, encryptionKey)
		if err != nil {
			return fmt.Errorf("failed to encrypt front document: %w", err)
		}
	}

	if len(back) > 0 {
		encBack, err = shared.Encrypt(back, encryptionKey)
		if err != nil {
			return fmt.Errorf("failed to encrypt back document: %w", err)
		}
	}

	doc := &model.UserDocument{
		ProfileID:      profileID,
		Type:           docType,
		EncryptedMeta:  encMeta,
		EncryptedFront: encFront,
		EncryptedBack:  encBack,
		Status:         model.DocumentStatusActive,
	}

	return s.repo.Create(ctx, doc)
}

func (s *UserDocumentService) GetDocumentsByProfileID(ctx context.Context, profileID uuid.UUID) ([]model.UserDocument, error) {
	return s.repo.GetByProfileID(ctx, profileID)
}
