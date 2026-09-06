#!/bin/sh
set -eu

REDIS_HOST="${REDIS_HOST:-127.0.0.1}"
REDIS_PORT="${REDIS_PORT:-6379}"

ROOT_PASS="$(cat /run/secrets/redis_root_pass)"
KMS_USER="${REDIS_KMS_ADMIN_USER:-redis_kms_admin}"
KMS_PASS="$(cat /run/secrets/redis_kms_admin_pass)"

echo "Waiting for Redis to start with root authentication..."

until redis-cli \
    -h "$REDIS_HOST" \
    -p "$REDIS_PORT" \
    -a "$ROOT_PASS" \
    --no-auth-warning \
    ping >/dev/null 2>&1
do
    sleep 1
done

echo "Configuring Redis ACL..."

# 1. Tworzymy konto KMS Admin (z prawami +acl i +ping)
redis-cli \
    -h "$REDIS_HOST" \
    -p "$REDIS_PORT" \
    -a "$ROOT_PASS" \
    --no-auth-warning \
    ACL SETUSER "$KMS_USER" on ">$KMS_PASS" "~*" "&*" "-@all" "+acl" "+ping"

# 2. Upewniamy się, że użytkownik default ma właściwe parametry
redis-cli \
    -h "$REDIS_HOST" \
    -p "$REDIS_PORT" \
    -a "$ROOT_PASS" \
    --no-auth-warning \
    ACL SETUSER default on ">$ROOT_PASS" "~*" "&*" "+@all"

echo "Redis ACL initialization completed."