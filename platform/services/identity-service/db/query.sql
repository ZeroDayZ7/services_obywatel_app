-- name: CreateCitizenWithAudit :one
INSERT INTO citizens (
    user_id, pesel_hash, encrypted_data, encrypted_dek, nonce, key_version
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING user_id, created_at;

-- name: CreateAuditLog :exec
INSERT INTO citizen_audit_logs (
    id, user_id, action, actor_id, ip_address, payload_hash
) VALUES (
    $1, $2, $3, $4, $5, $6
);

-- name: GetCitizenByPeselHash :one
SELECT user_id, pesel_hash, encrypted_data, encrypted_dek, nonce, key_version, created_at
FROM citizens
WHERE pesel_hash = $1 LIMIT 1;

-- name: GetCitizenByUserID :one
SELECT user_id, pesel_hash, encrypted_data, encrypted_dek, nonce, key_version, created_at
FROM citizens
WHERE user_id = $1 LIMIT 1;

-- name: GetUnsyncedAuditLogs :many
SELECT id, user_id, action, actor_id, ip_address, payload_hash, created_at
FROM citizen_audit_logs
WHERE synced_to_global_audit = FALSE
ORDER BY created_at ASC
LIMIT $1;

-- name: MarkAuditLogsAsSynced :exec
UPDATE citizen_audit_logs
SET synced_to_global_audit = TRUE
WHERE id = ANY($1::uuid[]);