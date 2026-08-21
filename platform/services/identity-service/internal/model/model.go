package model

import (
	"time"

	"github.com/google/uuid"
)

// AuditAction reprezentuje silnie typowany rodzaj akcji audytowej.
type AuditAction string

const (
	ActionCitizenRegistered AuditAction = "CITIZEN_REGISTERED"
	ActionPIIAccessed       AuditAction = "PII_ACCESSED"
	ActionKeyRotated        AuditAction = "KEY_ROTATED"
)

// CitizenPayload reprezentuje strukturę PII po odszyfrowaniu zawartości pola encrypted_data.
type CitizenPayload struct {
	FirstName   string  `json:"first_name"`
	SecondName  *string `json:"second_name,omitempty"`
	LastName    string  `json:"last_name"`
	PESEL       string  `json:"pesel"`
	Email       string  `json:"email"`
	PhoneNumber string  `json:"phone_number"`

	// Adres
	City        string  `json:"city"`
	Street      string  `json:"street"`
	HouseNumber string  `json:"house_number"`
	FlatNumber  *string `json:"flat_number,omitempty"`
	PostalCode  string  `json:"postal_code"`
}

// Citizen odwzorowuje tabelę "citizens" z kopertowym szyfrowaniem PII i blind indeksem.
type Citizen struct {
	UserID        uuid.UUID `db:"user_id" json:"user_id"`
	PESELHash     string    `db:"pesel_hash" json:"pesel_hash"`
	EncryptedData []byte    `db:"encrypted_data" json:"-"`
	EncryptedDEK  []byte    `db:"encrypted_dek" json:"-"`
	Nonce         []byte    `db:"nonce" json:"-"`
	KeyVersion    int       `db:"key_version" json:"key_version"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time `db:"updated_at" json:"updated_at"`
}

// CitizenAuditLog odwzorowuje tabelę "citizen_audit_logs" (Outbox Pattern dla audit-service).
type CitizenAuditLog struct {
	ID                  uuid.UUID   `db:"id" json:"id"`
	UserID              uuid.UUID   `db:"user_id" json:"user_id"`
	Action              AuditAction `db:"action" json:"action"`
	ActorID             uuid.UUID   `db:"actor_id" json:"actor_id"`
	IPAddress           *string     `db:"ip_address" json:"ip_address,omitempty"`
	PayloadHash         *string     `db:"payload_hash" json:"payload_hash,omitempty"`
	SyncedToGlobalAudit bool        `db:"synced_to_global_audit" json:"synced_to_global_audit"`
	CreatedAt           time.Time   `db:"created_at" json:"created_at"`
}
