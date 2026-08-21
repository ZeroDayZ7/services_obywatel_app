-- name: CreateAuditLog :exec
INSERT INTO citizen_audit_logs (
    id, user_id, action, actor_id, ip_address, payload_hash
) VALUES (
    $1, $2, $3, $4, $5, $6
);

-- name: GetUnsyncedAuditLogs :many
SELECT id, user_id, action, actor_id, ip_address, payload_hash, created_at
FROM citizen_audit_logs
WHERE synced_to_global_audit = FALSE
ORDER BY created_at ASC LIMIT $1;

-- name: MarkAuditLogsAsSynced :exec
UPDATE citizen_audit_logs
SET synced_to_global_audit = TRUE
WHERE id = ANY($1::uuid[]);