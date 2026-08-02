package model

import "github.com/google/uuid"

// #region Contact Enums
type ContactStatus string

const (
	ContactStatusPending  ContactStatus = "pending"
	ContactStatusAccepted ContactStatus = "accepted"
	ContactStatusBlocked  ContactStatus = "blocked"
)

// #endregion

// #region Contact Entities
// Contact – Relacja między użytkownikami z natywnym wsparciem dla synchronizacji i aliasów
type Contact struct {
	ID        uuid.UUID     `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	OwnerID   uuid.UUID     `gorm:"type:uuid;index:idx_owner_contact,unique;not null"` // Użytkownik posiadający ten kontakt
	ContactID uuid.UUID     `gorm:"type:uuid;index:idx_owner_contact,unique;not null"` // Użytkownik docelowy
	Status    ContactStatus `gorm:"type:varchar(20);not null;default:'pending';index"`

	// Zaszyfrowany lokalnie alias/nazwa dostosowana przez użytkownika
	EncryptedAlias []byte `gorm:"type:bytea" json:"-"`

	// Wersjonowanie zmiany relacji dla silnika synchronizacji (Delta Sync)
	Version uint64 `gorm:"not null;default:1;index"`

	BaseModel
}

// #endregion

// #region Contact DTOs
type SendContactRequest struct {
	TargetUserID uuid.UUID `json:"target_user_id"`
}

type RespondContactRequest struct {
	Accept bool `json:"accept"` // true = accepted, false = rejected/deleted
}

// #endregion
