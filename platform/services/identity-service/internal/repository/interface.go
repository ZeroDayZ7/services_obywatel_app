package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/zerodayz7/services/identity-service/internal/model"
)

var ErrCitizenAlreadyExists = errors.New("citizen already exists")

type CitizenRepository interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
	Create(ctx context.Context, citizen *model.Citizen) error
	CreateAgreement(ctx context.Context, agreement *model.UserAgreement) error
	CreatePukCode(ctx context.Context, puk *model.UserPukCode) error
	CreateAuditLog(ctx context.Context, audit *model.CitizenAuditLog) error
	CreateOutboxMessage(ctx context.Context, outbox *model.OutboxMessage) error
	GetByID(ctx context.Context, userID uuid.UUID) (*model.Citizen, error)

	GetByPESELHash(ctx context.Context, peselHash string) (*model.Citizen, error)
	GetByEmailHash(ctx context.Context, emailHash string) (*model.Citizen, error)
	GetByPhoneHash(ctx context.Context, phoneHash string) (*model.Citizen, error)

	GetAgreementByID(ctx context.Context, agreementID uuid.UUID) (*model.UserAgreement, error)
}

type OutboxRepository interface {
	FetchPendingMessages(ctx context.Context, limit int) ([]model.OutboxMessage, error)
	MarkAsSent(ctx context.Context, id uuid.UUID) error
	MarkAsFailed(ctx context.Context, id uuid.UUID, maxRetries int16, lastErr string) error
}
