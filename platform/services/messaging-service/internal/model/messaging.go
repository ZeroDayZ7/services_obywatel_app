package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// BaseModel – Standardowe pola audytowe z indykatorem wersji dla Delta Sync
type BaseModel struct {
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// --- ENUMY DOMENOWE ---

type ContactStatus string

const (
	ContactStatusPending  ContactStatus = "pending"
	ContactStatusAccepted ContactStatus = "accepted"
	ContactStatusBlocked  ContactStatus = "blocked"
)

type ConversationType string

const (
	ConversationTypeDirect ConversationType = "direct"
	ConversationTypeGroup  ConversationType = "group"
)

type MessageType string

const (
	MessageTypeText   MessageType = "text"
	MessageTypeMedia  MessageType = "media"
	MessageTypeSystem MessageType = "system"
)

// --- KRYPTOGRAFIA I ZARZĄDZANIE KLUCZAMI (Signal Protocol / E2EE Architecture) ---

// UserDeviceIdentity – Przechowuje publiczne klucze urządzenia użytkownika potrzebne do nawiązania sesji E2EE
type UserDeviceIdentity struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	UserID    uuid.UUID `gorm:"type:uuid;index;not null"`
	DeviceID  string    `gorm:"type:varchar(64);not null;index"` // Identyfikator instalacji / sprzętu
	PublicKey []byte    `gorm:"type:bytea;not null"`             // Długowieczny publiczny klucz tożsamości (Identity Key)

	// Klucze jednorazowe/okresowe do wymiany kluczy (X3DH Key Exchange)
	SignedPreKey    []byte `gorm:"type:bytea;not null"`
	SignedPreKeySig []byte `gorm:"type:bytea;not null"`
	SignedPreKeyID  uint32 `gorm:"not null"`

	// Licznik dostępnych jednorazowych kluczy (One-Time PreKeys) na serwerze
	OneTimePreKeysCount int `gorm:"not null;default:0"`

	BaseModel
}

// UserPreKey – Jednorazowe klucze publiczne (One-Time PreKeys) używane przy inicjalizacji czatu E2EE
type UserPreKey struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	DeviceID  uuid.UUID `gorm:"type:uuid;index;not null"`
	KeyID     uint32    `gorm:"not null;index"`
	PublicKey []byte    `gorm:"type:bytea;not null"`

	BaseModel
}

// --- DOMENA KONTAKTY (Contacts) ---

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

// --- DOMENA KONWERSACJE I WIADOMOŚCI (Chats & Messages) ---

// Conversation – Konwersacja prywatna lub grupowa
type Conversation struct {
	ID    uuid.UUID        `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	Type  ConversationType `gorm:"type:varchar(20);not null;default:'direct'"`
	Title string           `gorm:"type:varchar(128)"` // Wypełniane tylko dla grup (może być zaszyfrowane)

	// Ostatnia sekwencja wiadomości w konwersacji (zwiększana monotonicznie przy każdej wiadomości)
	LastSequence uint64 `gorm:"not null;default:0"`

	// Relacje GORM
	Members  []ConversationMember `gorm:"foreignKey:ConversationID"`
	Messages []Message            `gorm:"foreignKey:ConversationID"`

	BaseModel
}

// ConversationMember – Uczestnicy danej konwersacji i ich stan przeczytania
type ConversationMember struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	ConversationID uuid.UUID `gorm:"type:uuid;index:idx_conv_user,unique;not null"`
	UserID         uuid.UUID `gorm:"type:uuid;index:idx_conv_user,unique;not null"`
	Role           string    `gorm:"type:varchar(20);default:'member'"` // 'admin', 'member'

	// Wskaźnik synchronizacji: ostatnia odebrana/przeczytana sekwencja wiadomości przez tego użytkownika
	LastReadSequence uint64 `gorm:"not null;default:0"`

	BaseModel
}

// Message – Zaszyfrowana koperta z wiadomością (Payload E2EE jest nieczytelny dla serwera)
type Message struct {
	ID             uuid.UUID   `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	ConversationID uuid.UUID   `gorm:"type:uuid;index:idx_conv_seq,unique;not null"`
	SenderID       uuid.UUID   `gorm:"type:uuid;index;not null"`
	SenderDeviceID string      `gorm:"type:varchar(64);not null"`
	Type           MessageType `gorm:"type:varchar(20);not null;default:'text'"`

	// Monotoniczny numer sekwencyjny w ramach danej konwersacji (służy do sortowania i synchronizacji delta)
	Sequence uint64 `gorm:"index:idx_conv_seq,unique;not null"`

	// Szyfrowany ładunek wiadomości (AES-GCM / Signal Protocol Payload)
	// Serwer widzi wyłącznie ciąg bajtów i nie ma możliwości jego odszyfrowania
	EncryptedPayload []byte `gorm:"type:bytea;not null" json:"-"`

	// Odnośniki do załączników (dla wiadomości typu media)
	MediaHeader []byte `gorm:"type:bytea" json:"-"`

	// Globalny wskaźnik wersji w mikroserwisie dla synchronizacji offline -> online
	Version uint64 `gorm:"not null;default:1;index"`

	// Relacja do konwersacji
	Conversation *Conversation `gorm:"foreignKey:ConversationID;constraint:OnDelete:CASCADE"`

	BaseModel
}

// --- STRUKTURY PAMIĘCIOWE (DTOs / Dynamic Sync Outbox Payload) ---

// SyncDeltaRequest – Żądanie synchronizacji różnicowej wysyłane z aplikacji mobilnej
type SyncDeltaRequest struct {
	LastKnownContactVersion uint64 `json:"last_known_contact_version"`
	LastKnownMessageVersion uint64 `json:"last_known_message_version"`
}

// SyncDeltaResponse – Paczka zmian do zaaplikowania w lokalnej bazie Drift/SQLite
type SyncDeltaResponse struct {
	UpdatedContacts []Contact `json:"updated_contacts"`
	NewMessages     []Message `json:"new_messages"`
	HasMore         bool      `json:"has_more"`
}

// OutboxEventPayload – Struktura kolejkowana w lokalnej bazie urządzenia w trybie Offline
type OutboxEventPayload struct {
	EventID        uuid.UUID      `json:"event_id"`
	EventType      string         `json:"event_type"` // "SEND_MESSAGE", "ADD_CONTACT"
	ConversationID *uuid.UUID     `json:"conversation_id,omitempty"`
	Payload        datatypes.JSON `json:"payload"`
	CreatedAt      time.Time      `json:"created_at"`
}
