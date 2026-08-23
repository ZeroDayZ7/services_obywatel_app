package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/zerodayz7/services/identity-service/db/dbgen"
	"github.com/zerodayz7/services/identity-service/internal/model"
)

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

func (r *outboxRepository) FetchPendingMessages(ctx context.Context, limit int) ([]model.OutboxMessage, error) {
	rows, err := r.q.FetchPendingOutboxMessages(ctx, limit)
	if err != nil {
		return nil, err
	}

	messages := make([]model.OutboxMessage, len(rows))
	for i, row := range rows {
		messages[i] = model.OutboxMessage{
			ID:            row.ID,
			AggregateType: row.AggregateType,
			AggregateID:   row.AggregateID,
			EventType:     row.EventType,
			Payload:       row.Payload,
			RetryCount:    int8(row.RetryCount),
		}
	}

	return messages, nil
}

func (r *outboxRepository) MarkAsSent(ctx context.Context, id uuid.UUID) error {
	return r.q.MarkOutboxMessageAsSent(ctx, id)
}

func (r *outboxRepository) MarkAsFailed(ctx context.Context, id uuid.UUID, maxRetries int16, lastErr string) error {
	var lastErrPtr *string
	if lastErr != "" {
		lastErrPtr = &lastErr
	}

	return r.q.MarkOutboxMessageAsFailed(ctx, dbgen.MarkOutboxMessageAsFailedParams{
		ID:         id,
		RetryCount: maxRetries,
		LastError:  lastErrPtr,
	})
}
