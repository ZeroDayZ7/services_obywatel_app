package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/services/citizen-docs/internal/model"
)

type CitizenService interface {
	CreateProfile(ctx context.Context, userID uuid.UUID, data *model.CitizenData) error
}

type UserDocumentService interface {
	CreateDocument(
		ctx context.Context,
		meta *model.DocumentMeta,
		front []byte,
		back []byte,
		profileID uuid.UUID,
		docType model.DocumentType,
	) error
	GetDocumentsByProfileID(ctx context.Context, profileID uuid.UUID) ([]model.UserDocument, error)
}
