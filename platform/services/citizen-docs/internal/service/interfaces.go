package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/services/citizen-docs/internal/dto"
	"github.com/zerodayz7/platform/services/citizen-docs/internal/model"
)

type UserDocumentService interface {
	CreateDocument(ctx context.Context, payload model.CreateDocumentPayload) (*dto.DocumentResponse, error)
	GetDocumentByID(ctx context.Context, id uuid.UUID) (*dto.DocumentResponse, error)
	GetDocumentByDocumentNumber(ctx context.Context, documentNumber string) (*dto.DocumentResponse, error)
	GetDocumentsByUserID(ctx context.Context, userID uuid.UUID) ([]dto.DocumentResponse, error)
	GetDocumentsByUserIDWithSyncState(ctx context.Context, userID uuid.UUID, sinceVersion uint64, ifNoneMatch string) ([]dto.DocumentResponse, string, uint64, bool, error)
	UpdateDocumentStatus(ctx context.Context, id uuid.UUID, status model.DocumentStatus) (*dto.DocumentResponse, error)
	GetDocumentPDF(ctx context.Context, id uuid.UUID) ([]byte, string, error)
}
