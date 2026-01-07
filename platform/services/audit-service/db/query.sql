-- name: CreateLog :exec
INSERT INTO audit_logs (
    user_id, service_name, action, ip_address, metadata, status
) VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetAllLogs :many
SELECT * FROM audit_logs
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: GetLogsByAction :many
SELECT * FROM audit_logs
WHERE action = $1
ORDER BY created_at DESC;

-- name: GetLogByID :one
SELECT * FROM audit_logs
WHERE id = $1
LIMIT 1;

-- name: GetLogsByUserId :many
SELECT * FROM audit_logs
WHERE user_id = $1
ORDER BY created_at DESC;
