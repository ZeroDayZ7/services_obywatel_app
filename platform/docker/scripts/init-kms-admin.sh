#!/usr/bin/env bash
set -e

KMS_USER="${KMS_ADMIN_USER:-kms_admin}"
KMS_PASSWORD=$(cat /run/secrets/kms_admin_pass)
TARGET_DB="${POSTGRES_DB:-auth_database}"
TARGET_ROLE="kms_auth-service_postgres_auth"
MAIN_DB_USER="${POSTGRES_USER:-postgres}"

psql -v ON_ERROR_STOP=1 --username "$MAIN_DB_USER" --dbname "$TARGET_DB" <<-EOSQL
    -- 1. Tworzenie użytkownika kms_admin i roli grupowej
    DO \$block\$
    BEGIN
        IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = '$KMS_USER') THEN
            EXECUTE format('CREATE USER %I WITH PASSWORD %L CREATEROLE', '$KMS_USER', '$KMS_PASSWORD');
        ELSE
            EXECUTE format('ALTER USER %I WITH CREATEROLE PASSWORD %L', '$KMS_USER', '$KMS_PASSWORD');
        END IF;

        IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = '$TARGET_ROLE') THEN
            EXECUTE format('CREATE ROLE %I NOLOGIN', '$TARGET_ROLE');
        END IF;
    END
    \$block\$;

    -- 2. Nadanie praw do bazy i schematu public
    GRANT ALL PRIVILEGES ON DATABASE "$TARGET_DB" TO "$KMS_USER";
    GRANT ALL PRIVILEGES ON DATABASE "$TARGET_DB" TO "$TARGET_ROLE";
    GRANT ALL ON SCHEMA public TO "$KMS_USER";
    GRANT USAGE, CREATE ON SCHEMA public TO "$TARGET_ROLE";

    -- 3. Przypisanie praw do obecnych tabel i zmiana właściciela
    GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO "$TARGET_ROLE";
    GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO "$TARGET_ROLE";

    DO \$block\$
    DECLARE
        r RECORD;
    BEGIN
        FOR r IN (SELECT tablename FROM pg_tables WHERE schemaname = 'public') LOOP
            EXECUTE format('ALTER TABLE public.%I OWNER TO %I', r.tablename, '$TARGET_ROLE');
        END LOOP;
    END
    \$block\$;

    -- 4. Domyślne uprawnienia dla przyszłych tabel (używamy MAIN_DB_USER)
    ALTER DEFAULT PRIVILEGES FOR ROLE "$MAIN_DB_USER" IN SCHEMA public GRANT ALL ON TABLES TO "$TARGET_ROLE";
    ALTER DEFAULT PRIVILEGES FOR ROLE "$MAIN_DB_USER" IN SCHEMA public GRANT ALL ON SEQUENCES TO "$TARGET_ROLE";

    ALTER DEFAULT PRIVILEGES FOR ROLE "$KMS_USER" IN SCHEMA public GRANT ALL ON TABLES TO "$TARGET_ROLE";
    ALTER DEFAULT PRIVILEGES FOR ROLE "$KMS_USER" IN SCHEMA public GRANT ALL ON SEQUENCES TO "$TARGET_ROLE";

    ALTER DEFAULT PRIVILEGES FOR ROLE "$TARGET_ROLE" IN SCHEMA public GRANT ALL ON TABLES TO "$TARGET_ROLE";
    ALTER DEFAULT PRIVILEGES FOR ROLE "$TARGET_ROLE" IN SCHEMA public GRANT ALL ON SEQUENCES TO "$TARGET_ROLE";

    -- 5. Kluczowe: Nadanie praw administratora do roli dla KMS_USER
    GRANT "$TARGET_ROLE" TO "$KMS_USER" WITH ADMIN OPTION;
EOSQL