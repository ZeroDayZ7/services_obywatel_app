-- 1. Tabela główna tożsamości
CREATE TABLE IF NOT EXISTS citizens (
    -- Identyfikator użytkownika bezpośrednio z auth-service (UUIDv7 generowany po stronie aplikacji Go)
    user_id UUID PRIMARY KEY,
    
    -- BLIND INDEX (Do szukania po PESEL bez ujawniania wartości)
    pesel_hash VARCHAR(64) UNIQUE NOT NULL,
    
    -- ENCRYPTED PAYLOAD (Zaszyfrowany JSON: pesel, first_name, last_name, city, street, itp.)
    encrypted_data BYTEA NOT NULL,
    
    -- METADANE SZYFROWANIA KOPERTOWEGO
    encrypted_dek BYTEA NOT NULL,      -- Klucz DEK zaszyfrowany KEK-iem
    nonce BYTEA NOT NULL,              -- Nonce AES-GCM (12 bajtów)
    key_version INT NOT NULL DEFAULT 1, -- Wersja klucza głównego (rotacja)
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_citizens_pesel_hash ON citizens(pesel_hash);

-- 2. Tabela lokalnego audytu (Outbox Pattern dla audit-service)
CREATE TABLE IF NOT EXISTS citizen_audit_logs (
    id UUID PRIMARY KEY,               -- Generowany w Go UUIDv7
    user_id UUID NOT NULL,             -- Odniesienie do obywatela
    action VARCHAR(50) NOT NULL,       -- np. 'CITIZEN_REGISTERED', 'PII_ACCESSED', 'KEY_ROTATED'
    actor_id UUID NOT NULL,            -- Kto wykonał akcję (user_id / system_id)
    ip_address VARCHAR(45),            -- IPv4 / IPv6
    payload_hash VARCHAR(64),          -- Hash wygenerowanego zdarzenia dla weryfikacji integralności
    synced_to_global_audit BOOLEAN NOT NULL DEFAULT FALSE, -- Czy przekazane do głównego audit-service
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_audit_logs_un-synced ON citizen_audit_logs(synced_to_global_audit) WHERE synced_to_global_audit = FALSE;
CREATE INDEX idx_audit_logs_user_id ON citizen_audit_logs(user_id);