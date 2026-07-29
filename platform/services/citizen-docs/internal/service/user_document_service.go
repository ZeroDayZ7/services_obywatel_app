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
	docRepo     repository.UserDocumentRepo
	citizenRepo repository.CitizenRepo
	cfg         *config.Config
	logger      *shared.Logger
}

func NewUserDocumentService(
	docRepo repository.UserDocumentRepo,
	citizenRepo repository.CitizenRepo,
	cfg *config.Config,
	logger *shared.Logger,
) UserDocumentService {
	return &userDocumentService{
		docRepo:     docRepo,
		citizenRepo: citizenRepo,
		cfg:         cfg,
		logger:      logger,
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

	return s.docRepo.Create(ctx, doc)
}

// GetDocumentsByUserID pobiera pełny zestaw dokumentów dla wybranego obywatela (mapuje userID -> profileID).
func (s *userDocumentService) GetDocumentsByUserID(ctx context.Context, userID uuid.UUID) ([]model.UserDocument, error) {
	profile, err := s.citizenRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch profile for user %s: %w", userID, err)
	}

	return s.docRepo.GetByProfileID(ctx, profile.ID)
}

// GetDocumentsSinceVersion pobiera różnicowo (Delta Sync) dokumenty nowsze niż podana wersja dla wybranego obywatela.
func (s *userDocumentService) GetDocumentsSinceVersion(ctx context.Context, userID uuid.UUID, sinceVersion uint64) ([]model.UserDocument, error) {
	profile, err := s.citizenRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch profile for user %s: %w", userID, err)
	}

	return s.docRepo.GetSinceVersion(ctx, profile.ID, sinceVersion)
}
