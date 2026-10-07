package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// #region Contact Enums
type ContactStatus string

type ContactSyncState string

type ContactDirection string

const (
	ContactStatusPending  ContactStatus = "pending"
	ContactStatusAccepted ContactStatus = "accepted"
	ContactStatusBlocked  ContactStatus = "blocked"
)

const (
	ContactSyncStateSynced        ContactSyncState = "synced"
	ContactSyncStatePendingCreate ContactSyncState = "pending_create"
	ContactSyncStatePendingUpdate ContactSyncState = "pending_update"
	ContactSyncStatePendingDelete ContactSyncState = "pending_delete"
)

const (
	ContactDirectionIncoming ContactDirection = "incoming"
	ContactDirectionOutgoing ContactDirection = "outgoing"
)

// #endregion

// #region Contact Entities
// Contact – Relacja między użytkownikami z natywnym wsparciem dla synchronizacji i aliasów.
// local_alias jest prywatnym polem aplikacji lokalnej i nie powinno być nadpisywane przez serwer.
type Contact struct {
	ID        uuid.UUID     `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	OwnerID   uuid.UUID     `gorm:"type:uuid;index:idx_owner_contact,unique;not null"`
	ContactID uuid.UUID     `gorm:"type:uuid;index:idx_owner_contact,unique;not null"`
	Status    ContactStatus `gorm:"type:varchar(20);not null;default:'pending';index"`

	// Synchronization metadata
	SyncState   ContactSyncState `gorm:"type:varchar(30);not null;default:'synced';index"`
	Direction   ContactDirection `gorm:"type:varchar(20);not null;default:'incoming';index"`
	ChangeSeq   uint64           `gorm:"not null;default:1;index"`
	DeletedAt   gorm.DeletedAt   `gorm:"index" json:"deleted_at,omitempty"`
	LocalAlias  string           `gorm:"type:varchar(128)" json:"local_alias,omitempty"`
	EncryptedAlias []byte        `gorm:"type:bytea" json:"-"`

	// Wersjonowanie zmiany relacji dla silnika synchronizacji (Delta Sync)
	Version uint64 `gorm:"not null;default:1;index"`

	BaseModel
}

// #endregion

// #region Contact DTOs
type SendContactRequest struct {
	TargetUserID uuid.UUID `json:"target_user_id"`
	IdempotencyKey string `json:"idempotency_key,omitempty"`
}

type RespondContactRequest struct {
	Accept bool `json:"accept"`
	Action ContactActionType `json:"action,omitempty"`
	IdempotencyKey string `json:"idempotency_key,omitempty"`
}

type ContactActionType string

const (
	ContactActionAccept ContactActionType = "ACCEPT"
	ContactActionReject ContactActionType = "REJECT"
	ContactActionBlock  ContactActionType = "BLOCK"
	ContactActionRemove ContactActionType = "REMOVE"
)

type ContactActionRequest struct {
	Action        ContactActionType `json:"action"`
	IdempotencyKey string           `json:"idempotency_key,omitempty"`
	Reason        string           `json:"reason,omitempty"`
}

type ContactActionResponse struct {
	Status        string    `json:"status"`
	ContactID     uuid.UUID `json:"contact_id"`
	Action        ContactActionType `json:"action"`
	ProcessedAt   time.Time `json:"processed_at"`
	IdempotencyKey string   `json:"idempotency_key,omitempty"`
}

type AcceptContactRequest struct {
	IdempotencyKey string `json:"idempotency_key,omitempty"`
}

type RejectContactRequest struct {
	Reason        string `json:"reason,omitempty"`
	IdempotencyKey string `json:"idempotency_key,omitempty"`
}

type BlockContactRequest struct {
	Reason        string `json:"reason,omitempty"`
	IdempotencyKey string `json:"idempotency_key,omitempty"`
}

type RemoveContactRequest struct {
	Reason        string `json:"reason,omitempty"`
	IdempotencyKey string `json:"idempotency_key,omitempty"`
}

// #endregion
