DROP TABLE IF EXISTS audit_logs;

CREATE TABLE audit_logs (
    id            BIGSERIAL PRIMARY KEY,
    user_id       UUID NOT NULL,
    service_name  VARCHAR(50) NOT NULL,
    action        VARCHAR(100) NOT NULL,
    ip_address    VARCHAR(45) NOT NULL,
    metadata      JSONB NOT NULL,
    status        VARCHAR(20) NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_user ON audit_logs(user_id);
CREATE INDEX idx_audit_ip ON audit_logs(ip_address);
CREATE INDEX idx_audit_service ON audit_logs(service_name);
