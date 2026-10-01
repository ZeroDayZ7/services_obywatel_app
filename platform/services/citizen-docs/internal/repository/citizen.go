package repository

import (
	"context"

	"github.com/google/uuid"
)

type citizenRepository struct{}

func NewCitizenRepository() *citizenRepository {
	return &citizenRepository{}
}

func (r *citizenRepository) CreateProfile(_ context.Context, _ uuid.UUID, _ map[string]any) error {
	return nil
}
