-- +goose Up
-- +goose StatementBegin

-- 1. Tabela USERS z natywną funkcją uuidv7() z PostgreSQL 17+
CREATE TABLE IF NOT EXISTS users (
    id                    UUID PRIMARY KEY DEFAULT uuidv7(),
    username              VARCHAR(30) NOT NULL,
    email                 VARCHAR(100) NOT NULL,
    password              VARCHAR(128) NOT NULL,
    role                  VARCHAR(20) NOT NULL DEFAULT 'CITIZEN',
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

    CONSTRAINT idx_users_username UNIQUE (username),
    CONSTRAINT idx_users_email UNIQUE (email)
);

CREATE INDEX IF NOT EXISTS idx_users_locked_until ON users (locked_until);
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users (deleted_at);

-- 2. Tabela AVAILABLE_PERMISSIONS z natywnym uuidv7()
CREATE TABLE IF NOT EXISTS available_permissions (
    id          UUID PRIMARY KEY DEFAULT uuidv7(),
    key         VARCHAR(100) NOT NULL,
    department  VARCHAR(50) NOT NULL,
    description VARCHAR(255) NOT NULL,
    is_special  BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT idx_available_permissions_key UNIQUE (key)
);

CREATE INDEX IF NOT EXISTS idx_available_permissions_department ON available_permissions (department);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS available_permissions;
DROP TABLE IF EXISTS users;

-- +goose StatementEnd