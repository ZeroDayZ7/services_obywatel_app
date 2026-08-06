package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/pkg/envelope"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/citizen-docs/config"
	"github.com/zerodayz7/platform/services/citizen-docs/internal/model"
	"github.com/zerodayz7/platform/services/citizen-docs/internal/repository"
)

type citizenService struct {
	repo    repository.CitizenRepo
	cfg     *config.Config
	logger  *shared.Logger
	cryptor *envelope.EnvelopeCryptor
}

func NewCitizenService(
	repo repository.CitizenRepo,
	cfg *config.Config,
	logger *shared.Logger,
	cryptor *envelope.EnvelopeCryptor,
) CitizenService {
	return &citizenService{
		repo:    repo,
		cfg:     cfg,
		logger:  logger,
		cryptor: cryptor,
	}
}

// #region CREATE PROFILE
func (s *citizenService) CreateProfile(ctx context.Context, userID uuid.UUID, data *model.CitizenData) error {
	plainBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	// 1. Szyfrujemy kopertowo (KMS generuje/zaszyfrowuje DEK dla aliasu "docs-id-cards")
	payload, err := s.cryptor.Seal(ctx, "docs-id-cards", plainBytes)
	if err != nil {
		return err
	}

	hash := sha256.New()
	hash.Write([]byte(data.PESEL + s.cfg.Internal.DocsPeselSalt))
	peselHash := hex.EncodeToString(hash.Sum(nil))

	profile := &model.CitizenProfile{
		UserID:        userID,
		EncryptedData: payload.EncryptedData,
		EncryptedDEK:  payload.EncryptedDEK,
		PeselHash:     peselHash,
	}

	return s.repo.Create(ctx, profile)
}
// #endregion