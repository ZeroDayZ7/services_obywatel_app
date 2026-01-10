// platform/services/citizen-docs/internal/service/citizen_service.go

package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/pkg/viper"
	"github.com/zerodayz7/platform/services/citizen-docs/internal/model"
	"github.com/zerodayz7/platform/services/citizen-docs/internal/repository"
)

type CitizenService struct {
	repo   repository.CitizenRepo
	cfg    *viper.Config
	logger *shared.Logger
}

func NewCitizenService(repo repository.CitizenRepo, cfg *viper.Config, logger *shared.Logger) *CitizenService {
	return &CitizenService{
		repo:   repo,
		cfg:    cfg,
		logger: logger,
	}
}

func (s *CitizenService) CreateProfile(ctx context.Context, userID uuid.UUID, data *model.CitizenData) error {
	encryptionKey := []byte(s.cfg.Internal.EncryptionKey)

	plainBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	encryptedBlob, err := shared.Encrypt(plainBytes, encryptionKey)
	if err != nil {
		return err
	}

	hash := sha256.New()
	hash.Write([]byte(data.PESEL + s.cfg.Internal.HashSalt))
	peselHash := hex.EncodeToString(hash.Sum(nil))

	profile := &model.CitizenProfile{
		UserID:        userID,
		EncryptedData: encryptedBlob,
		PeselHash:     peselHash,
	}

	return s.repo.Create(ctx, profile)
}
