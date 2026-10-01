-- +goose Up
-- +goose StatementBegin

-- 1. Tabela USERS
CREATE TABLE IF NOT EXISTS users (
    id                    UUID PRIMARY KEY DEFAULT uuidv7(),
    username              VARCHAR(30) NOT NULL,
    email                 VARCHAR(100) NOT NULL,
    password              VARCHAR(128) NOT NULL,
    role                  VARCHAR(20) NOT NULL DEFAULT 'CITIZEN',
    permissions           JSONB NOT NULL DEFAULT '[]'::jsonb,
    status                VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    failed_login_attempts SMALLINT NOT NULL DEFAULT 0,
    locked_until          TIMESTAMPTZ NULL,
    last_login            TIMESTAMPTZ NULL,
    password_changed_at   TIMESTAMPTZ NULL,
    last_ip               VARCHAR(45) NULL,
    two_factor_enabled    BOOLEAN NOT NULL DEFAULT FALSE,
    two_factor_secret     VARCHAR(64) NULL,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at            TIMESTAMPTZ NULL,

    CONSTRAINT uq_users_username UNIQUE (username),
    CONSTRAINT uq_users_email UNIQUE (email)
);

CREATE INDEX IF NOT EXISTS idx_users_locked_until ON users (locked_until);
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users (deleted_at);

-- 2. Tabela EMPLOYEE_PROFILES
CREATE TABLE IF NOT EXISTS employee_profiles (
    user_id          UUID PRIMARY KEY,
    employee_number  VARCHAR(64) NOT NULL,
    institution_id   UUID NOT NULL,
    department_id    UUID NOT NULL,
    permissions      JSONB NOT NULL DEFAULT '[]'::jsonb,
    active           BOOLEAN NOT NULL DEFAULT TRUE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT uq_employee_number UNIQUE (employee_number),
    CONSTRAINT fk_employee_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_employee_profiles_institution_id ON employee_profiles (institution_id);
CREATE INDEX IF NOT EXISTS idx_employee_profiles_department_id ON employee_profiles (department_id);

-- 3. Tabela EMPLOYEE_CREDENTIALS
CREATE TABLE IF NOT EXISTS employee_credentials (
    id                 UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id            UUID NOT NULL,
    card_serial_number VARCHAR(128) NOT NULL,
    public_key         TEXT NOT NULL,
    key_algorithm      VARCHAR(30) NOT NULL DEFAULT 'ED25519',
    status             VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    issued_by          UUID NOT NULL,
    expires_at         TIMESTAMPTZ NULL,
    last_used_at       TIMESTAMPTZ NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at         TIMESTAMPTZ NULL,

    CONSTRAINT uq_card_serial_number UNIQUE (card_serial_number),
    CONSTRAINT fk_employee_cred_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_employee_credentials_user_id ON employee_credentials (user_id);

-- 4. Tabela USER_DEVICES
CREATE TABLE IF NOT EXISTS user_devices (
    id                      UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id                 UUID NOT NULL,
    device_fingerprint      VARCHAR(128) NOT NULL,
    public_key              TEXT NOT NULL,
    device_name_encrypted   VARCHAR(256) NULL,
    platform                VARCHAR(30) NULL,
    is_active               BOOLEAN NOT NULL DEFAULT TRUE,
    is_verified             BOOLEAN NOT NULL DEFAULT FALSE,
    last_ip                 VARCHAR(45) NULL,
    last_used_at            TIMESTAMPTZ NULL,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at              TIMESTAMPTZ NULL,

    CONSTRAINT uq_user_device UNIQUE (user_id, device_fingerprint),
    CONSTRAINT fk_user_device_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_user_devices_user_id ON user_devices (user_id);

-- 5. Tabela REFRESH_TOKENS
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id                 UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id            UUID NOT NULL,
    device_id          UUID NULL,
    token              VARCHAR(128) NOT NULL,
    device_fingerprint VARCHAR(128) NOT NULL,
    revoked            BOOLEAN NOT NULL DEFAULT FALSE,
    expires_at         TIMESTAMPTZ NOT NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at         TIMESTAMPTZ NULL,

    CONSTRAINT uq_refresh_token UNIQUE (token),
    CONSTRAINT fk_refresh_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_refresh_device FOREIGN KEY (device_id) REFERENCES user_devices(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens (user_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_revoked ON refresh_tokens (revoked);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expires_at ON refresh_tokens (expires_at);

-- 6. Tabela AVAILABLE_PERMISSIONS
CREATE TABLE IF NOT EXISTS available_permissions (
    id          UUID PRIMARY KEY DEFAULT uuidv7(),
    key         VARCHAR(100) NOT NULL,
    department  VARCHAR(50) NOT NULL,
    description VARCHAR(255) NOT NULL,
    is_special  BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT uq_available_permissions_key UNIQUE (key)
);

CREATE INDEX IF NOT EXISTS idx_available_permissions_department ON available_permissions (department);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS available_permissions;
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS user_devices;
DROP TABLE IF EXISTS employee_credentials;
DROP TABLE IF EXISTS employee_profiles;
DROP TABLE IF EXISTS users;

-- +goose StatementEnd