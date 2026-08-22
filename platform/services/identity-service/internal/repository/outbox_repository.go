package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zerodayz7/services/identity-service/db/dbgen"
	"github.com/zerodayz7/services/identity-service/internal/model"
)

type OutboxRepository interface {
	FetchPendingMessages(ctx context.Context, limit int32) ([]model.OutboxMessage, error)
	MarkAsSent(ctx context.Context, id uuid.UUID) error
	MarkAsFailed(ctx context.Context, id uuid.UUID, maxRetries int16, lastErr string) error
}

type outboxRepository struct {
	dbPool *pgxpool.Pool
	q      *dbgen.Queries
}

func NewOutboxRepository(dbPool *pgxpool.Pool) OutboxRepository {
	return &outboxRepository{
		dbPool: dbPool,
		q:      dbgen.New(dbPool),
	}
}

func (r *outboxRepository) FetchPendingMessages(ctx context.Context, limit int32) ([]model.OutboxMessage, error) {
	rows, err := r.q.FetchPendingOutboxMessages(ctx, limit)
	if err != nil {
		return nil, err
	}

	messages := make([]model.OutboxMessage, len(rows))
	for i, row := range rows {
		messages[i] = model.OutboxMessage{
			ID:            row.ID.Bytes,
			AggregateType: row.AggregateType,
			AggregateID:   row.AggregateID.Bytes,
			EventType:     row.EventType,
			Payload:       row.Payload,
			RetryCount:    int8(row.RetryCount),
		}
	}

	return messages, nil
}

func (r *outboxRepository) MarkAsSent(ctx context.Context, id uuid.UUID) error {
	return r.q.MarkOutboxMessageAsSent(ctx, uuidToPgType(id))
}

func (r *outboxRepository) MarkAsFailed(ctx context.Context, id uuid.UUID, maxRetries int16, lastErr string) error {
	return r.q.MarkOutboxMessageAsFailed(ctx, dbgen.MarkOutboxMessageAsFailedParams{
		ID:         uuidToPgType(id),
		RetryCount: maxRetries,
		LastError:  stringToPgText(&lastErr),
	})
}
