package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/zerodayz7/services/officer-bff/internal/model"
)

type OfficialService interface {
	RegisterCitizenWorkflow(ctx context.Context, req model.RegisterCitizenRequest) (*model.RegisterCitizenResponse, error)
}

type officialService struct{}

func NewOfficialService() OfficialService {
	return &officialService{}
}

func (s *officialService) RegisterCitizenWorkflow(ctx context.Context, req model.RegisterCitizenRequest) (*model.RegisterCitizenResponse, error) {
	citizenID, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}

	return &model.RegisterCitizenResponse{
		CitizenID:      citizenID,
		PUKCode:        "PUK-STUB-12345",
		ActivationCode: "ACT-8899",
		RegisteredAt:   time.Now().UTC(),
	}, nil
}
