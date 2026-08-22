-- name: CreateOutboxMessage :one
INSERT INTO outbox_messages (
    id, aggregate_type, aggregate_id, event_type, payload, status
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: FetchPendingOutboxMessages :many
SELECT id, aggregate_type, aggregate_id, event_type, payload, retry_count
FROM outbox_messages
WHERE status = 'PENDING' AND deleted_at IS NULL
ORDER BY created_at ASC
LIMIT $1;

-- name: MarkOutboxMessageAsSent :exec
UPDATE outbox_messages
SET status = 'SENT', processed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: MarkOutboxMessageAsFailed :exec
UPDATE outbox_messages
SET status = CASE WHEN retry_count + 1 >= $2 THEN 'FAILED' ELSE 'PENDING' END,
    retry_count = retry_count + 1,
    last_error = $3,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1;