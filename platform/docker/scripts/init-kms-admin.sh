#!/usr/bin/env bash
set -e

KMS_USER="${KMS_ADMIN_USER:-kms_admin}"
KMS_PASSWORD=$(cat /run/secrets/kms_admin_pass)
TARGET_DB="${POSTGRES_DB:-auth_database}"

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$TARGET_DB" <<-EOSQL
    -- 1. Tworzenie użytkownika kms_admin z prawem CREATEROLE
    DO \$block\$
    BEGIN
        IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = '$KMS_USER') THEN
            EXECUTE format('CREATE USER %I WITH PASSWORD %L CREATEROLE', '$KMS_USER', '$KMS_PASSWORD');
        ELSE
            EXECUTE format('ALTER USER %I WITH CREATEROLE PASSWORD %L', '$KMS_USER', '$KMS_PASSWORD');
        END IF;
    END
    \$block\$;

    -- 2. Nadanie praw do bazy danych
    GRANT ALL PRIVILEGES ON DATABASE "$TARGET_DB" TO "$KMS_USER";

    -- 3. Nadanie praw do schematu public i istniejących tabel
    GRANT ALL ON SCHEMA public TO "$KMS_USER";
    GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO "$KMS_USER";
    GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO "$KMS_USER";

    -- 4. Nadanie domyślnych praw na przyszłe tabele i sekwencje
    ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO "$KMS_USER";
    ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO "$KMS_USER";
EOSQL

echo "Użytkownik $KMS_USER został pomyślnie skonfigurowany w bazie $TARGET_DB."