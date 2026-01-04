CREATE TABLE encrypted_audit_logs (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT NOT NULL,
    service_name    VARCHAR(50) NOT NULL,    -- Dodano: źródło logu
    action          VARCHAR(100) NOT NULL,
    ip_address      VARCHAR(45) NOT NULL,    -- Dodano: jawne IP (45 znaków obsłuży IPv6)
    encrypted_data  BYTEA NOT NULL,          -- Zaszyfrowane Metadata
    encrypted_key   BYTEA NOT NULL,
    status          VARCHAR(20) NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indeksy dla szybkiego wyszukiwania
CREATE INDEX idx_audit_user ON encrypted_audit_logs(user_id);
CREATE INDEX idx_audit_ip ON encrypted_audit_logs(ip_address);
CREATE INDEX idx_audit_service ON encrypted_audit_logs(service_name);