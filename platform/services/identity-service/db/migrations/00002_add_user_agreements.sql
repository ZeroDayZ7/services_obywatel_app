-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS user_agreements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE,
    agreement_number VARCHAR(64) NOT NULL UNIQUE,
    pesel_encrypted VARCHAR(256) NOT NULL,
    verified_phone VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    signed_at TIMESTAMP WITH TIME ZONE NOT NULL,
    verified_at TIMESTAMP WITH TIME ZONE,
    verified_via VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_user_agreements_pesel_encrypted ON user_agreements(pesel_encrypted);
CREATE INDEX IF NOT EXISTS idx_user_agreements_deleted_at ON user_agreements(deleted_at);

CREATE TABLE IF NOT EXISTS user_puk_codes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_agreement_id UUID NOT NULL UNIQUE REFERENCES user_agreements(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    puk_hash VARCHAR(128) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    failed_attempts SMALLINT NOT NULL DEFAULT 0,
    max_attempts SMALLINT NOT NULL DEFAULT 3,
    expires_at TIMESTAMP WITH TIME ZONE,
    used_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_user_puk_codes_user_id ON user_puk_codes(user_id);
CREATE INDEX IF NOT EXISTS idx_user_puk_codes_expires_at ON user_puk_codes(expires_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS user_puk_codes;
DROP TABLE IF EXISTS user_agreements;
-- +goose StatementEnd