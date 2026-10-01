package service

import (
	"context"

	"github.com/google/uuid"
)

type citizenService struct{}

func NewCitizenService() *citizenService {
	return &citizenService{}
}

func (s *citizenService) CreateProfile(_ context.Context, _ uuid.UUID, _ any) error {
	return nil
}
