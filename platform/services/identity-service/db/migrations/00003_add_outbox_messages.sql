-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS outbox_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_type VARCHAR(64) NOT NULL,
    aggregate_id UUID NOT NULL,
    event_type VARCHAR(128) NOT NULL,
    payload JSONB NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    retry_count SMALLINT NOT NULL DEFAULT 0,
    last_error TEXT,
    processed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Indeks dedykowany dla workera wyciągającego oczekujące wiadomości chronologicznie
CREATE INDEX IF NOT EXISTS idx_outbox_pending_messages 
ON outbox_messages (created_at ASC) 
WHERE status = 'PENDING' AND deleted_at IS NULL;

-- Indeksy wyszukiwania po obiekcie biznesowym oraz soft-delete
CREATE INDEX IF NOT EXISTS idx_outbox_aggregate ON outbox_messages (aggregate_type, aggregate_id);
CREATE INDEX IF NOT EXISTS idx_outbox_deleted_at ON outbox_messages (deleted_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS outbox_messages;
-- +goose StatementEnd