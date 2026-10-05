package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/services/citizen-docs/internal/model"
)

type UserDocumentService interface {
	CreateDocument(ctx context.Context, payload model.CreateDocumentPayload) (*model.CitizenDocument, error)
	GetDocumentByID(ctx context.Context, id uuid.UUID) (*model.CitizenDocument, error)
	GetDocumentByDocumentNumber(ctx context.Context, documentNumber string) (*model.CitizenDocument, error)
	GetDocumentsByUserID(ctx context.Context, userID uuid.UUID) ([]model.CitizenDocument, error)
	UpdateDocumentStatus(ctx context.Context, id uuid.UUID, status model.DocumentStatus) (*model.CitizenDocument, error)
	GetDocumentPDF(ctx context.Context, id uuid.UUID) ([]byte, string, error)
}
