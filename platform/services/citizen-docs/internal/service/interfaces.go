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
		profileID uuid.UUID,
		typeCode string,
		meta *model.DocumentMeta,
		front []byte,
		back []byte,
		issuerSignature []byte,
		signingKeyID string,
		revocationSerial string,
	) error
	GetDocumentsByUserID(ctx context.Context, userID uuid.UUID) ([]model.UserDocument, error)
	GetDocumentsSinceVersion(ctx context.Context, profileID uuid.UUID, sinceVersion uint64) ([]model.UserDocument, error)
}
