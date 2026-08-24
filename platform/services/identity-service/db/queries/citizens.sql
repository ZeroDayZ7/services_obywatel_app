-- name: CreateCitizenWithAudit :one
INSERT INTO citizens (
    user_id, pesel_hash, email_hash, phone_hash, encrypted_data, encrypted_dek
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING user_id, created_at;

-- name: GetCitizenByPeselHash :one
SELECT user_id, pesel_hash, email_hash, phone_hash, encrypted_data, encrypted_dek, created_at
FROM citizens WHERE pesel_hash = $1 LIMIT 1;

-- name: GetCitizenByEmailHash :one
SELECT user_id, pesel_hash, email_hash, phone_hash, encrypted_data, encrypted_dek, created_at
FROM citizens WHERE email_hash = $1 LIMIT 1;

-- name: GetCitizenByPhoneHash :one
SELECT user_id, pesel_hash, email_hash, phone_hash, encrypted_data, encrypted_dek, created_at
FROM citizens WHERE phone_hash = $1 LIMIT 1;

-- name: GetCitizenByUserID :one
SELECT user_id, pesel_hash, email_hash, phone_hash, encrypted_data, encrypted_dek, created_at
FROM citizens WHERE user_id = $1 LIMIT 1;