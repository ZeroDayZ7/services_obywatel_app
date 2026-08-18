package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/zerodayz7/services/identity-service/internal/model"
	"github.com/zerodayz7/services/identity-service/internal/repository"
)

type CitizenService interface {
	RegisterCitizen(ctx context.Context, payload model.CitizenPayload) (*model.Citizen, error)
	GetCitizenByID(ctx context.Context, userID uuid.UUID) (*model.Citizen, error)
}

type citizenService struct {
	repo repository.CitizenRepository
}

func NewCitizenService(repo repository.CitizenRepository) CitizenService {
	return &citizenService{repo: repo}
}

func (s *citizenService) RegisterCitizen(ctx context.Context, payload model.CitizenPayload) (*model.Citizen, error) {
	// TODO: Wykorzystać KMS/Kopertę do wygenerowania DEK, zaszyfrowania payloadu i utworzenia pesel_hash
	userID, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}

	citizen := &model.Citizen{
		UserID:        userID,
		PESELHash:     "placeholder_blind_index_hash", // Do zastąpienia HMAC/SHA256 z salt
		EncryptedData: []byte("encrypted_payload"),
		EncryptedDEK:  []byte("encrypted_dek"),
		Nonce:         []byte("12byte_nonce"),
		KeyVersion:    1,
	}

	if err := s.repo.Create(ctx, citizen); err != nil {
		return nil, err
	}

	return citizen, nil
}

func (s *citizenService) GetCitizenByID(ctx context.Context, userID uuid.UUID) (*model.Citizen, error) {
	return s.repo.GetByID(ctx, userID)
}
