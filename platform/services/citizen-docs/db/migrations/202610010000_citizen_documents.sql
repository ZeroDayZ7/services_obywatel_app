-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS citizen_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    document_type VARCHAR(64) NOT NULL,
    document_number VARCHAR(128) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'PENDING',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    issued_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_citizen_documents_user_id ON citizen_documents(user_id);
CREATE INDEX IF NOT EXISTS idx_citizen_documents_document_type ON citizen_documents(document_type);
CREATE INDEX IF NOT EXISTS idx_citizen_documents_document_number ON citizen_documents(document_number);
CREATE INDEX IF NOT EXISTS idx_citizen_documents_status ON citizen_documents(status);
CREATE INDEX IF NOT EXISTS idx_citizen_documents_issued_at ON citizen_documents(issued_at);
CREATE INDEX IF NOT EXISTS idx_citizen_documents_expires_at ON citizen_documents(expires_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS citizen_documents;
-- +goose StatementEnd
