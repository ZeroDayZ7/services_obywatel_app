package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/citizen-docs/config"
	"github.com/zerodayz7/platform/services/citizen-docs/internal/model"
	"github.com/zerodayz7/platform/services/citizen-docs/internal/repository"
)

type userDocumentService struct {
	repo   repository.UserDocumentRepo
	cfg    *config.Config
	logger *shared.Logger
}

func NewUserDocumentService(repo repository.UserDocumentRepo, cfg *config.Config, logger *shared.Logger) UserDocumentService {
	return &userDocumentService{
		repo:   repo,
		cfg:    cfg,
		logger: logger,
	}
}

// CreateDocument szyfruje metadane i obrazy oraz tworzy dokument ze wsparciem dla weryfikacji offline.
func (s *userDocumentService) CreateDocument(
	ctx context.Context,
	profileID uuid.UUID,
	typeCode string,
	meta *model.DocumentMeta,
	front []byte,
	back []byte,
	issuerSignature []byte,
	signingKeyID string,
	revocationSerial string,
) error {
	encryptionKey := []byte(s.cfg.Internal.DocsEncryptionKey)

	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("failed to marshal document meta: %w", err)
	}

	encMeta, err := shared.Encrypt(metaBytes, encryptionKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt document meta: %w", err)
	}

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
		ProfileID:        profileID,
		TypeCode:         typeCode,
		EncryptedMeta:    encMeta,
		EncryptedFront:   encFront,
		EncryptedBack:    encBack,
		IssuerSignature:  issuerSignature,
		SigningKeyID:     signingKeyID,
		RevocationSerial: revocationSerial,
		Status:           model.DocumentStatusActive,
		Version:          1,
	}

	return s.repo.Create(ctx, doc)
}

// GetDocumentsByUserID pobiera pełny zestaw dokumentów dla wybranego obywatela.
func (s *userDocumentService) GetDocumentsByUserID(ctx context.Context, userID uuid.UUID) ([]model.UserDocument, error) {
	return s.repo.GetByUserID(ctx, userID)
}

// GetDocumentsSinceVersion pobiera różnicowo (Delta Sync) dokumenty nowsze niż podana wersja.
func (s *userDocumentService) GetDocumentsSinceVersion(ctx context.Context, profileID uuid.UUID, sinceVersion uint64) ([]model.UserDocument, error) {
	return s.repo.GetSinceVersion(ctx, profileID, sinceVersion)
}
