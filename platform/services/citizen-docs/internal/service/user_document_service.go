package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/pkg/envelope"
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
	cryptor     *envelope.EnvelopeCryptor
}

func NewUserDocumentService(
	docRepo repository.UserDocumentRepo,
	citizenRepo repository.CitizenRepo,
	cfg *config.Config,
	logger *shared.Logger,
	cryptor *envelope.EnvelopeCryptor,
) UserDocumentService {
	return &userDocumentService{
		docRepo:     docRepo,
		citizenRepo: citizenRepo,
		cfg:         cfg,
		logger:      logger,
		cryptor:     cryptor,
	}
}

// CreateDocument szyfruje metadane i obrazy przy użyciu Envelope Encryption i KMS.
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
	// Określamy alias KMS na podstawie typu dokumentu (ID, PRAWO JAZDY, PASZPORT)
	keyAlias := getKeyAliasForTypeCode(typeCode)

	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("failed to marshal document meta: %w", err)
	}

	// 1. Szyfrowanie Metadanych
	encMetaPayload, err := s.cryptor.Seal(ctx, keyAlias, metaBytes)
	if err != nil {
		return fmt.Errorf("failed to encrypt document meta via KMS: %w", err)
	}

	// 2. Szyfrowanie Frontu (opcjonalne)
	var encFrontData, encFrontDEK []byte
	if len(front) > 0 {
		frontPayload, err := s.cryptor.Seal(ctx, keyAlias, front)
		if err != nil {
			return fmt.Errorf("failed to encrypt front image via KMS: %w", err)
		}
		encFrontData = frontPayload.EncryptedData
		encFrontDEK = frontPayload.EncryptedDEK
	}

	// 3. Szyfrowanie Tyłu (opcjonalne)
	var encBackData, encBackDEK []byte
	if len(back) > 0 {
		backPayload, err := s.cryptor.Seal(ctx, keyAlias, back)
		if err != nil {
			return fmt.Errorf("failed to encrypt back image via KMS: %w", err)
		}
		encBackData = backPayload.EncryptedData
		encBackDEK = backPayload.EncryptedDEK
	}

	doc := &model.UserDocument{
		ProfileID:        profileID,
		TypeCode:         typeCode,
		EncryptedMeta:    encMetaPayload.EncryptedData,
		EncryptedMetaDEK: encMetaPayload.EncryptedDEK, // DEK dla metadanych
		EncryptedFront:   encFrontData,
		EncryptedFrontDEK: encFrontDEK,               // DEK dla zdjęcia przodu
		EncryptedBack:    encBackData,
		EncryptedBackDEK:  encBackDEK,                // DEK dla zdjęcia tyłu
		IssuerSignature:  issuerSignature,
		SigningKeyID:     signingKeyID,
		RevocationSerial: revocationSerial,
		Status:           model.DocumentStatusActive,
		Version:          1,
	}

	return s.docRepo.Create(ctx, doc)
}

func (s *userDocumentService) GetDocumentsByUserID(ctx context.Context, userID uuid.UUID) ([]model.UserDocument, error) {
	profile, err := s.citizenRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch profile for user %s: %w", userID, err)
	}

	return s.docRepo.GetByProfileID(ctx, profile.ID)
}

func (s *userDocumentService) GetDocumentsSinceVersion(ctx context.Context, userID uuid.UUID, sinceVersion uint64) ([]model.UserDocument, error) {
	profile, err := s.citizenRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch profile for user %s: %w", userID, err)
	}

	return s.docRepo.GetSinceVersion(ctx, profile.ID, sinceVersion)
}

// Pomocnik do pobierania właściwego aliasu klucza w zależności od typu dokumentu
func getKeyAliasForTypeCode(typeCode string) string {
	switch typeCode {
	case "DRIVER_LICENSE":
		return "docs-driver-license"
	case "PASSPORT":
		return "docs-passport"
	default:
		return "docs-id-cards"
	}
}