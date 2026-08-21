-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS citizens (
    user_id UUID PRIMARY KEY,
    pesel_hash VARCHAR(64) UNIQUE NOT NULL,
    encrypted_data BYTEA NOT NULL,
    encrypted_dek BYTEA NOT NULL,
    key_version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_citizens_pesel_hash ON citizens(pesel_hash);

CREATE TABLE IF NOT EXISTS citizen_audit_logs (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    action VARCHAR(50) NOT NULL,
    actor_id UUID NOT NULL,
    ip_address VARCHAR(45),
    payload_hash VARCHAR(64),
    prev_hash VARCHAR(64) NOT NULL,
    hash VARCHAR(64) NOT NULL,
    synced_to_global_audit BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_unsynced ON citizen_audit_logs(synced_to_global_audit) WHERE synced_to_global_audit = FALSE;
CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON citizen_audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON citizen_audit_logs(created_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS citizen_audit_logs;
DROP TABLE IF EXISTS citizens;
-- +goose StatementEnd