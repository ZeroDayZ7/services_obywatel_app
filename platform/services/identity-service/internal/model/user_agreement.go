package model

import (
	"time"

	"github.com/google/uuid"
)

type AgreementStatus string

const (
	AgreementStatusPending AgreementStatus = "PENDING"
	AgreementStatusActive  AgreementStatus = "ACTIVE"
	AgreementStatusRevoked AgreementStatus = "REVOKED"
	AgreementStatusExpired AgreementStatus = "EXPIRED"
)

type PukStatus string

const (
	PukStatusActive  PukStatus = "ACTIVE"
	PukStatusUsed    PukStatus = "USED"
	PukStatusBlocked PukStatus = "BLOCKED"
)

// #region UserAgreement
type UserAgreement struct {
	ID              uuid.UUID       `db:"id" json:"id"`
	UserID          uuid.UUID       `db:"user_id" json:"user_id"`
	AgreementNumber string          `db:"agreement_number" json:"agreement_number"`
	S3Key           string          `db:"s3_key" json:"s3_key"`
	S3Bucket        string          `db:"s3_bucket" json:"s3_bucket"`
	EncryptedDEK    []byte          `db:"encrypted_dek" json:"-"`
	EncryptedEmail  []byte          `db:"encrypted_email" json:"-"`
	EncryptedPhone  []byte          `db:"encrypted_phone" json:"-"`
	Status          AgreementStatus `db:"status" json:"status"`
	SignedAt        time.Time       `db:"signed_at" json:"signed_at"`
	VerifiedAt      *time.Time      `db:"verified_at" json:"verified_at,omitempty"`
	VerifiedVia     string          `db:"verified_via" json:"verified_via"`
	CreatedAt       time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time       `db:"updated_at" json:"updated_at"`
	DeletedAt       *time.Time      `db:"deleted_at" json:"deleted_at,omitempty"`

	PukCode *UserPukCode `db:"-" json:"puk_code,omitempty"`
}

// #region UserPukCode
type UserPukCode struct {
	ID              uuid.UUID  `db:"id" json:"id"`
	UserAgreementID uuid.UUID  `db:"user_agreement_id" json:"user_agreement_id"`
	UserID          uuid.UUID  `db:"user_id" json:"user_id"`
	PukHash         string     `db:"puk_hash" json:"-"`
	Status          PukStatus  `db:"status" json:"status"`
	FailedAttempts  int8       `db:"failed_attempts" json:"failed_attempts"`
	MaxAttempts     int8       `db:"max_attempts" json:"max_attempts"`
	ExpiresAt       *time.Time `db:"expires_at" json:"expires_at,omitempty"`
	UsedAt          *time.Time `db:"used_at" json:"used_at,omitempty"`
	CreatedAt       time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time  `db:"updated_at" json:"updated_at"`
	DeletedAt       *time.Time `db:"deleted_at" json:"deleted_at,omitempty"`
}
