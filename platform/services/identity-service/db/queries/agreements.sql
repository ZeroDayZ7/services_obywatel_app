-- name: GetAgreementByUserID :one
SELECT * FROM user_agreements
WHERE user_id = $1 AND deleted_at IS NULL LIMIT 1;

-- name: GetAgreementByNumber :one
SELECT * FROM user_agreements
WHERE agreement_number = $1 AND deleted_at IS NULL LIMIT 1;

-- name: UpdateAgreementStatus :exec
UPDATE user_agreements
SET status = $2, verified_at = $3, updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL;

-- name: CreateUserPukCode :one
INSERT INTO user_puk_codes (
    id, user_agreement_id, user_id, puk_hash, status, failed_attempts, max_attempts, expires_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: GetPukCodeByUserID :one
SELECT * FROM user_puk_codes
WHERE user_id = $1 AND deleted_at IS NULL LIMIT 1;

-- name: IncrementPukFailedAttempts :exec
UPDATE user_puk_codes
SET failed_attempts = failed_attempts + 1,
    status = CASE WHEN failed_attempts + 1 >= max_attempts THEN 'BLOCKED'::varchar ELSE status END,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL;

-- name: MarkPukAsUsed :exec
UPDATE user_puk_codes
SET status = 'USED', used_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL;

-- name: CreateUserAgreement :one
INSERT INTO user_agreements (
    id, user_id, agreement_number, s3_key, s3_bucket, encrypted_dek, key_version, pesel_encrypted, verified_phone, status, signed_at, verified_at, verified_via
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
) RETURNING *;

-- name: GetAgreementByID :one
SELECT * FROM user_agreements
WHERE id = $1 AND deleted_at IS NULL LIMIT 1;