-- name: CreateEncryptedLog :exec
INSERT INTO encrypted_audit_logs (
    user_id, service_name, action, ip_address, encrypted_data, encrypted_key, status
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
);

-- name: GetAllLogs :many
SELECT * FROM encrypted_audit_logs
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: GetLogsByAction :many
SELECT * FROM encrypted_audit_logs
WHERE action = $1
ORDER BY created_at DESC;

-- name: GetLogByID :one
SELECT * FROM encrypted_audit_logs
WHERE id = $1 LIMIT 1;

-- name: GetLogsByUserId :many
SELECT * FROM encrypted_audit_logs
WHERE user_id = $1
ORDER BY created_at DESC;