-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS citizen_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    document_type VARCHAR(64) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'PENDING',
    document_number_hash VARCHAR(64) NOT NULL,
    encrypted_metadata BYTEA NOT NULL,
    encrypted_dek BYTEA NOT NULL,
    issued_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_citizen_documents_user_id ON citizen_documents(user_id);
CREATE INDEX IF NOT EXISTS idx_citizen_documents_document_type ON citizen_documents(document_type);
CREATE UNIQUE INDEX IF NOT EXISTS ux_citizen_documents_document_number_hash ON citizen_documents(document_number_hash);
CREATE INDEX IF NOT EXISTS idx_citizen_documents_status ON citizen_documents(status);
CREATE INDEX IF NOT EXISTS idx_citizen_documents_issued_at ON citizen_documents(issued_at);
CREATE INDEX IF NOT EXISTS idx_citizen_documents_expires_at ON citizen_documents(expires_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS citizen_documents;
-- +goose StatementEnd
