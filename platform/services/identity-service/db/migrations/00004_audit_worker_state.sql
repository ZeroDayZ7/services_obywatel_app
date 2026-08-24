-- +goose Up
-- +goose StatementBegin
ALTER TABLE citizen_audit_logs
    ADD COLUMN IF NOT EXISTS sync_state VARCHAR(32) NOT NULL DEFAULT 'PENDING',
    ADD COLUMN IF NOT EXISTS retry_count INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS last_error TEXT,
    ADD COLUMN IF NOT EXISTS processing_started_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS processed_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP;

CREATE INDEX IF NOT EXISTS idx_audit_logs_sync_state
    ON citizen_audit_logs (sync_state, created_at)
    WHERE sync_state IN ('PENDING', 'RETRY');

CREATE INDEX IF NOT EXISTS idx_audit_logs_processing
    ON citizen_audit_logs (processing_started_at)
    WHERE processing_started_at IS NOT NULL;

UPDATE citizen_audit_logs
SET updated_at = created_at
WHERE updated_at IS NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_audit_logs_processing;
DROP INDEX IF EXISTS idx_audit_logs_sync_state;

ALTER TABLE citizen_audit_logs
    DROP COLUMN IF EXISTS updated_at,
    DROP COLUMN IF EXISTS processed_at,
    DROP COLUMN IF EXISTS processing_started_at,
    DROP COLUMN IF EXISTS last_error,
    DROP COLUMN IF EXISTS retry_count,
    DROP COLUMN IF EXISTS sync_state;
-- +goose StatementEnd
