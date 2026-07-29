package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/services/citizen-docs/internal/model"
)

type CitizenRepo interface {
	Create(ctx context.Context, profile *model.CitizenProfile) error
	GetByUserID(ctx context.Context, userID uuid.UUID) (*model.CitizenProfile, error)
	GetByPeselHash(ctx context.Context, hash string) (*model.CitizenProfile, error)
}

type UserDocumentRepo interface {
	Create(ctx context.Context, doc *model.UserDocument) error
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]model.UserDocument, error)
	GetSinceVersion(ctx context.Context, profileID uuid.UUID, sinceVersion uint64) ([]model.UserDocument, error)
}
