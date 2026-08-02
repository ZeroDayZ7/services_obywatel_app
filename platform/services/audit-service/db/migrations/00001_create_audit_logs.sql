-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS audit.audit_logs (
    id            BIGSERIAL PRIMARY KEY,
    user_id       UUID NOT NULL,
    service_name  VARCHAR(50) NOT NULL,
    action        VARCHAR(100) NOT NULL,
    ip_address    VARCHAR(45) NOT NULL,
    metadata      JSONB NOT NULL,
    status        VARCHAR(20) NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_user ON audit.audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_ip ON audit.audit_logs(ip_address);
CREATE INDEX IF NOT EXISTS idx_audit_service ON audit.audit_logs(service_name);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS audit.audit_logs;
-- +goose StatementEnd