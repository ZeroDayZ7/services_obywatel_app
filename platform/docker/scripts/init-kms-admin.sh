#!/usr/bin/env bash
set -e

KMS_USER="${KMS_ADMIN_USER:-kms_admin}"
KMS_PASSWORD=$(cat /run/secrets/kms_admin_pass)
TARGET_DB="${POSTGRES_DB:-auth_database}"
TARGET_ROLE="kms_auth-service_postgres_auth"
MAIN_DB_USER="${POSTGRES_USER:-postgres}"

psql -v ON_ERROR_STOP=1 --username "$MAIN_DB_USER" --dbname "$TARGET_DB" <<-EOSQL
    -- 1. Tworzenie użytkownika KMS (zarządcy ról) oraz roli aplikacyjnej (NOLOGIN)
    DO \$block\$
    BEGIN
        IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = '$KMS_USER') THEN
            EXECUTE format('CREATE USER %I WITH PASSWORD %L CREATEROLE INHERIT', '$KMS_USER', '$KMS_PASSWORD');
        ELSE
            EXECUTE format('ALTER USER %I WITH CREATEROLE INHERIT PASSWORD %L', '$KMS_USER', '$KMS_PASSWORD');
        END IF;

        IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = '$TARGET_ROLE') THEN
            EXECUTE format('CREATE ROLE %I NOLOGIN INHERIT', '$TARGET_ROLE');
        END IF;
    END
    \$block\$;

    -- 2. Dostęp do bazy i schematu public (tylko USAGE, bez prawa CREATE w schemacie)
    GRANT CONNECT ON DATABASE "$TARGET_DB" TO "$TARGET_ROLE";
    GRANT USAGE ON SCHEMA public TO "$TARGET_ROLE";
    
    -- Dostęp administracyjny dla KMS_USER (potrzebny do zarzadzania obiektami/reassign)
    GRANT ALL PRIVILEGES ON DATABASE "$TARGET_DB" TO "$KMS_USER";
    GRANT ALL ON SCHEMA public TO "$KMS_USER";

    -- 3. Nadanie stricte operacyjnych uprawnień DML do ISTNIEJĄCYCH tabel i sekwencji
    GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO "$TARGET_ROLE";
    GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO "$TARGET_ROLE";

    -- KMS_USER otrzymuje pełne uprawnienia do tabel
    GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO "$KMS_USER";
    GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO "$KMS_USER";

    -- 4. Ustawienie domyślnych uprawnień dla NOWO TWORZONYCH tabel (np. przy migracjach)
    -- Gdy tabele tworzy główny użytkownik bazy (MAIN_DB_USER / postgres):
    ALTER DEFAULT PRIVILEGES FOR ROLE "$MAIN_DB_USER" IN SCHEMA public 
        GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO "$TARGET_ROLE";
    ALTER DEFAULT PRIVILEGES FOR ROLE "$MAIN_DB_USER" IN SCHEMA public 
        GRANT USAGE, SELECT ON SEQUENCES TO "$TARGET_ROLE";

    -- Gdy tabele tworzy KMS_USER:
    ALTER DEFAULT PRIVILEGES FOR ROLE "$KMS_USER" IN SCHEMA public 
        GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO "$TARGET_ROLE";
    ALTER DEFAULT PRIVILEGES FOR ROLE "$KMS_USER" IN SCHEMA public 
        GRANT USAGE, SELECT ON SEQUENCES TO "$TARGET_ROLE";

    -- 5. Nadanie praw zarządzania rolą TARGET_ROLE dla KMS_USER (wymagane do GRANT/REVOKE w Rust)
    GRANT "$TARGET_ROLE" TO "$KMS_USER" WITH ADMIN OPTION;
EOSQL