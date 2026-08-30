#!/usr/bin/env bash
set -e

KMS_USER="${KMS_ADMIN_USER:-kms_admin}"
KMS_PASSWORD=$(cat /run/secrets/kms_admin_pass)
TARGET_DB="${POSTGRES_DB:-auth_database}"
TARGET_ROLE="kms_auth-service_postgres_auth"

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$TARGET_DB" <<-EOSQL
    -- 1. Tworzenie użytkownika kms_admin z prawem CREATEROLE oraz roli grupowej
    DO \$block\$
    BEGIN
        IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = '$KMS_USER') THEN
            EXECUTE format('CREATE USER %I WITH PASSWORD %L CREATEROLE', '$KMS_USER', '$KMS_PASSWORD');
        ELSE
            EXECUTE format('ALTER USER %I WITH CREATEROLE PASSWORD %L', '$KMS_USER', '$KMS_PASSWORD');
        END IF;

        -- Tworzenie roli grupowej, do której KMS będzie przypisywał tymczasowych userów
        IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = '$TARGET_ROLE') THEN
            EXECUTE format('CREATE ROLE %I NOLOGIN', '$TARGET_ROLE');
        END IF;
    END
    \$block\$;

    -- 2. Nadanie praw do bazy danych dla KMS
    GRANT ALL PRIVILEGES ON DATABASE "$TARGET_DB" TO "$KMS_USER";

    -- 3. Nadanie praw do schematu public i istniejących tabel dla KMS
    GRANT ALL ON SCHEMA public TO "$KMS_USER";
    GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO "$KMS_USER";
    GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO "$KMS_USER";

    -- 4. Nadanie domyślnych praw na przyszłe tabele i sekwencje dla KMS oraz TARGET_ROLE
    ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO "$KMS_USER";
    ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO "$KMS_USER";
    ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO "$TARGET_ROLE";
    ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO "$TARGET_ROLE";

    -- 5. KONFIGURACJA ROLI DLA AGENTÓW (ZMIANA: dodano CREATE oraz ALL PRIVILEGES):
    -- Nadanie uprawnień do tworzenia i używania obiektów w schemacie public
    GRANT USAGE, CREATE ON SCHEMA public TO "$TARGET_ROLE";
    GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO "$TARGET_ROLE";
    GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO "$TARGET_ROLE";

    -- 6. KLUCZOWE: Pozwól użytkownikowi KMS nadawać tę rolę innym użytkownikom (ADMIN OPTION)
    GRANT "$TARGET_ROLE" TO "$KMS_USER" WITH ADMIN OPTION;
EOSQL

echo "Użytkownik $KMS_USER oraz rola $TARGET_ROLE zostały pomyślnie skonfigurowane w bazie $TARGET_DB."