package model

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// #region Base Models
// BaseModel – Standardowe pola audytowe z indykatorem wersji dla Delta Sync
type BaseModel struct {
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// #endregion

// #region Messaging Enums
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

// #endregion

// #region E2EE & Crypto Entities
// UserDeviceIdentity – Przechowuje publiczne klucze urządzenia użytkownika potrzebne do nawiązania sesji E2EE
type UserDeviceIdentity struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	UserID         uuid.UUID `gorm:"type:uuid;index;not null"`
	DeviceID       string    `gorm:"type:varchar(64);not null;index"` // Identyfikator instalacji / sprzętu
	RegistrationID uint32    `gorm:"not null;default:0"`
	PublicKey      []byte    `gorm:"type:bytea;not null"` // Długowieczny publiczny klucz tożsamości (Identity Key)

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

type ActivationStatus string

const (
	ActivationStatusNotStarted ActivationStatus = "not_started"
	ActivationStatusPending    ActivationStatus = "pending"
	ActivationStatusActive     ActivationStatus = "active"
	ActivationStatusDisabled   ActivationStatus = "disabled"
)

// MessagingTermsDocument stores the controlled, versioned terms text for the communicator.
// The backend decides which version is current; the client only accepts a value that matches it.
type MessagingTermsDocument struct {
	Version     string    `json:"version"`
	Text        string    `json:"text"`
	PublishedAt time.Time `json:"published_at"`
}

var messagingTermsArchive = []MessagingTermsDocument{
	{
		Version:     "v1",
		Text:        "Komunikator XYZ umożliwia prowadzenie prywatnych rozmów z innymi użytkownikami Obywatel Plus. Przed rozpoczęciem korzystania z komunikatora zapoznaj się z zasadami usługi i potwierdź ich akceptację.",
		PublishedAt: time.Date(2026, time.October, 9, 0, 0, 0, 0, time.UTC),
	},
}

func CurrentMessagingTerms() MessagingTermsDocument {
	if len(messagingTermsArchive) == 0 {
		return MessagingTermsDocument{Version: "v1", Text: "Komunikator XYZ umożliwia prowadzenie prywatnych rozmów z innymi użytkownikami Obywatel Plus. Przed rozpoczęciem korzystania z komunikatora zapoznaj się z zasadami usługi i potwierdź ich akceptację."}
	}
	return messagingTermsArchive[len(messagingTermsArchive)-1]
}

func IsKnownMessagingTermsVersion(version string) bool {
	trimmed := strings.TrimSpace(version)
	for _, terms := range messagingTermsArchive {
		if terms.Version == trimmed {
			return true
		}
	}
	return false
}

// MessagingActivation tracks the real onboarding lifecycle for the communicator and is the
// production replacement for ad-hoc demo or fake seed-based activation states.
type MessagingActivation struct {
	ID              uuid.UUID        `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	UserID          uuid.UUID        `gorm:"type:uuid;uniqueIndex:idx_msg_activation_user;not null"`
	DeviceID        string           `gorm:"type:varchar(64);not null;default:''"`
	Status          ActivationStatus `gorm:"type:varchar(32);not null;default:'not_started';index"`
	ConsentAccepted bool             `gorm:"not null;default:false"`
	TermsVersion    string           `gorm:"type:varchar(32);not null;default:''"`
	ActivatedAt     *time.Time       `gorm:"index"`
	LastSeenAt      *time.Time       `gorm:"index"`

	BaseModel
}

type ActivateMessagingRequest struct {
	DeviceID     string `json:"device_id,omitempty"`
	TermsVersion string `json:"terms_version,omitempty"`
	Consent      bool   `json:"consent,omitempty"`
}

type AcceptTermsRequest struct {
	DeviceID     string `json:"device_id,omitempty"`
	TermsVersion string `json:"terms_version,omitempty"`
}

type MessagingActivationStatusResponse struct {
	UserID                 uuid.UUID        `json:"user_id"`
	Status                 ActivationStatus `json:"status"`
	ConsentAccepted        bool             `json:"consent_accepted"`
	TermsVersion           string           `json:"terms_version,omitempty"`
	CurrentTermsVersion    string           `json:"current_terms_version,omitempty"`
	RequiresTermsAcceptance bool            `json:"requires_terms_acceptance"`
	DeviceID               string           `json:"device_id,omitempty"`
	ActivatedAt            *time.Time       `json:"activated_at,omitempty"`
	CreatedAt              time.Time        `json:"created_at"`
	UpdatedAt              time.Time        `json:"updated_at"`
}

func (a *MessagingActivation) ToResponse() *MessagingActivationStatusResponse {
	if a == nil {
		return nil
	}
	currentTerms := CurrentMessagingTerms()
	requiresTermsAcceptance := a.Status != ActivationStatusActive || !a.ConsentAccepted || a.TermsVersion != currentTerms.Version
	return &MessagingActivationStatusResponse{
		UserID:                 a.UserID,
		Status:                 a.Status,
		ConsentAccepted:        a.ConsentAccepted,
		TermsVersion:           a.TermsVersion,
		CurrentTermsVersion:    currentTerms.Version,
		RequiresTermsAcceptance: requiresTermsAcceptance,
		DeviceID:               a.DeviceID,
		ActivatedAt:            a.ActivatedAt,
		CreatedAt:              a.CreatedAt,
		UpdatedAt:              a.UpdatedAt,
	}
}

// #endregion

// #region Conversation & Message Entities
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
	IdempotencyKey string      `gorm:"type:varchar(128);index:idx_message_idempotency,unique;not null;default:''" json:"idempotency_key,omitempty"`
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

// #endregion

// #region Persistent Encrypted Vault Contracts
// MessageEnvelope – transportowa i archiwalna koperta zaszyfrowanej wiadomości.
// Serwer nie dekoduje treści, a jedynie przechowuje ciphertext i metadane retencji.
type MessageEnvelope struct {
	MessageID              string     `json:"messageId,omitempty"`
	MessageIDSnake         string     `json:"message_id,omitempty"`
	ConversationID         uuid.UUID  `json:"conversationId,omitempty"`
	ConversationIDSnake    uuid.UUID  `json:"conversation_id,omitempty"`
	SenderUserID           uuid.UUID  `json:"senderUserId,omitempty"`
	SenderUserIDSnake      uuid.UUID  `json:"sender_user_id,omitempty"`
	SenderDeviceID         string     `json:"senderDeviceId,omitempty"`
	SenderDeviceIDSnake    string     `json:"sender_device_id,omitempty"`
	RecipientUserID        uuid.UUID  `json:"recipientUserId,omitempty"`
	RecipientUserIDSnake   uuid.UUID  `json:"recipient_user_id,omitempty"`
	RecipientDeviceID      string     `json:"recipientDeviceId,omitempty"`
	RecipientDeviceIDSnake string     `json:"recipient_device_id,omitempty"`
	Ciphertext             []byte     `json:"ciphertext,omitempty"`
	Type                   uint8      `json:"type,omitempty"`
	TypeSnake              uint8      `json:"signal_message_type,omitempty"`
	Nonce                  []byte     `json:"nonce,omitempty"`
	CreatedAt              time.Time  `json:"createdAt,omitempty"`
	ExpiresAt              *time.Time `json:"expiresAt,omitempty"`
	Version                uint64     `json:"version,omitempty"`
}

func (r *MessageEnvelope) Normalize() {
	if r.MessageID == "" {
		r.MessageID = r.MessageIDSnake
	}
	if r.ConversationID == uuid.Nil {
		r.ConversationID = r.ConversationIDSnake
	}
	if r.SenderUserID == uuid.Nil {
		r.SenderUserID = r.SenderUserIDSnake
	}
	if r.SenderDeviceID == "" {
		r.SenderDeviceID = r.SenderDeviceIDSnake
	}
	if r.RecipientUserID == uuid.Nil {
		r.RecipientUserID = r.RecipientUserIDSnake
	}
	if r.RecipientDeviceID == "" {
		r.RecipientDeviceID = r.RecipientDeviceIDSnake
	}
	if r.Type == 0 {
		r.Type = r.TypeSnake
	}
}

// MessageRecord – zapis chronologii wiadomości w przechowalni zaszyfrowanych kopert.
type MessageRecord struct {
	ID                     uuid.UUID   `json:"id,omitempty"`
	MessageID              uuid.UUID   `json:"messageId,omitempty"`
	MessageIDSnake         uuid.UUID   `json:"message_id,omitempty"`
	ConversationID         uuid.UUID   `json:"conversationId,omitempty"`
	ConversationIDSnake    uuid.UUID   `json:"conversation_id,omitempty"`
	SenderUserID           uuid.UUID   `json:"senderUserId,omitempty"`
	SenderUserIDSnake      uuid.UUID   `json:"sender_user_id,omitempty"`
	SenderDeviceID         string      `json:"senderDeviceId,omitempty"`
	SenderDeviceIDSnake    string      `json:"sender_device_id,omitempty"`
	RecipientUserID        uuid.UUID   `json:"recipientUserId,omitempty"`
	RecipientUserIDSnake   uuid.UUID   `json:"recipient_user_id,omitempty"`
	RecipientDeviceID      string      `json:"recipientDeviceId,omitempty"`
	RecipientDeviceIDSnake string      `json:"recipient_device_id,omitempty"`
	Ciphertext             []byte      `json:"ciphertext,omitempty"`
	Type                   MessageType `json:"type,omitempty"`
	CreatedAt              time.Time   `json:"createdAt,omitempty"`
	ExpiresAt              *time.Time  `json:"expiresAt,omitempty"`
	IsDelivered            bool        `json:"isDelivered,omitempty"`
	Version                uint64      `json:"version,omitempty"`
}

func (r *MessageRecord) Normalize() {
	if r.MessageID == uuid.Nil {
		r.MessageID = r.MessageIDSnake
	}
	if r.ConversationID == uuid.Nil {
		r.ConversationID = r.ConversationIDSnake
	}
	if r.SenderUserID == uuid.Nil {
		r.SenderUserID = r.SenderUserIDSnake
	}
	if r.SenderDeviceID == "" {
		r.SenderDeviceID = r.SenderDeviceIDSnake
	}
	if r.RecipientUserID == uuid.Nil {
		r.RecipientUserID = r.RecipientUserIDSnake
	}
	if r.RecipientDeviceID == "" {
		r.RecipientDeviceID = r.RecipientDeviceIDSnake
	}
}

// HistoryFetchRequest – żądanie pobrania historii wiadomości z szyfrowanego archiwum.
type HistoryFetchRequest struct {
	ConversationID      *uuid.UUID `json:"conversationId,omitempty"`
	ConversationIDSnake *uuid.UUID `json:"conversation_id,omitempty"`
	Since               uint64     `json:"since,omitempty"`
	Limit               int        `json:"limit,omitempty"`
}

func (r *HistoryFetchRequest) Normalize() {
	if r.ConversationID == nil {
		r.ConversationID = r.ConversationIDSnake
	}
	if r.Limit <= 0 {
		r.Limit = 50
	}
}

// NewMessageNotification – lekkie powiadomienie WebSocket o nowej wiadomości z archiwum i message_id.
type NewMessageNotification struct {
	EventType           string    `json:"eventType,omitempty"`
	EventTypeSnake      string    `json:"event_type,omitempty"`
	ConversationID      uuid.UUID `json:"conversationId,omitempty"`
	ConversationIDSnake uuid.UUID `json:"conversation_id,omitempty"`
	MessageID           string    `json:"messageId,omitempty"`
	MessageIDSnake      string    `json:"message_id,omitempty"`
	SenderUserID        uuid.UUID `json:"senderUserId,omitempty"`
	SenderUserIDSnake   uuid.UUID `json:"sender_user_id,omitempty"`
	SenderDeviceID      string    `json:"senderDeviceId,omitempty"`
	SenderDeviceIDSnake string    `json:"sender_device_id,omitempty"`
	Timestamp           time.Time `json:"timestamp,omitempty"`
}

func (r *NewMessageNotification) Normalize() {
	if r.EventType == "" {
		r.EventType = r.EventTypeSnake
	}
	if r.ConversationID == uuid.Nil {
		r.ConversationID = r.ConversationIDSnake
	}
	if r.MessageID == "" {
		r.MessageID = r.MessageIDSnake
	}
	if r.SenderUserID == uuid.Nil {
		r.SenderUserID = r.SenderUserIDSnake
	}
	if r.SenderDeviceID == "" {
		r.SenderDeviceID = r.SenderDeviceIDSnake
	}
}

// #endregion

// #region Sync & Outbox DTOs
// SyncDeltaRequest – Żądanie synchronizacji różnicowej wysyłane z aplikacji mobilnej.
// `last_known_sequence` jest globalnym kursorem zmian dla użytkownika i jest preferowanym polem
// w nowym modelu Offline-First; stare pola wersji są zachowane dla kompatybilności wstecznej.
type SyncDeltaRequest struct {
	LastKnownSequence       uint64 `json:"last_known_sequence,omitempty"`
	LastKnownContactVersion uint64 `json:"last_known_contact_version,omitempty"`
	LastKnownMessageVersion uint64 `json:"last_known_message_version,omitempty"`
	IdempotencyKey          string `json:"idempotency_key,omitempty"`
	Limit                   int    `json:"limit,omitempty"`
}

// SyncDeltaResponse – Paczka zmian do zaaplikowania w lokalnej bazie Drift/SQLite.
type SyncDeltaResponse struct {
	UpdatedContacts       []Contact `json:"updated_contacts"`
	NewMessages           []Message `json:"new_messages"`
	NextSequence          uint64    `json:"next_sequence,omitempty"`
	HasMore               bool      `json:"has_more"`
	AppliedIdempotencyKey string    `json:"applied_idempotency_key,omitempty"`
}

// OutboxEventPayload – Struktura kolejkowana w lokalnej bazie urządzenia w trybie Offline
type OutboxEventPayload struct {
	EventID        uuid.UUID      `json:"event_id"`
	IdempotencyKey string         `json:"idempotency_key,omitempty"`
	EventType      string         `json:"event_type"` // "SEND_MESSAGE", "ADD_CONTACT"
	ConversationID *uuid.UUID     `json:"conversation_id,omitempty"`
	Payload        datatypes.JSON `json:"payload"`
	CreatedAt      time.Time      `json:"created_at"`
}

// OutboxBatchRequest - Paczka zdarzeń wysyłana z klienta w trybie offline
type OutboxBatchRequest struct {
	Messages []OutboxEventPayload `json:"messages"`
}

// OutboxBatchResponse - Wynik przetworzenia paczki outbox
type OutboxBatchResponse struct {
	ProcessedCount int `json:"processed_count"`
}

// #endregion

// #region Conversation & Crypto DTOs
// CreateConversationRequest - Żądanie utworzenia nowej konwersacji
type CreateConversationRequest struct {
	Type         ConversationType `json:"type"`
	Title        string           `json:"title,omitempty"`
	RecipientIDs []uuid.UUID      `json:"recipient_ids"`
}

// UploadDeviceKeysRequest - Rejestracja kluczy E2EE dla urządzenia.
// Akceptuje oba formaty pól: camelCase i snake_case, aby wspierać klienta Flutter i backendowy kontrakt Signal.
type UploadDeviceKeysRequest struct {
	DeviceID               string   `json:"deviceId,omitempty"`
	DeviceIDSnake          string   `json:"device_id,omitempty"`
	RegistrationID         uint32   `json:"registrationId,omitempty"`
	RegistrationIDSnake    uint32   `json:"registration_id,omitempty"`
	IdentityPublicKey      []byte   `json:"identityPublicKey,omitempty"`
	IdentityPublicKeySnake []byte   `json:"identity_public_key,omitempty"`
	PublicKey              []byte   `json:"publicKey,omitempty"`
	SignedPreKey           []byte   `json:"signedPreKey,omitempty"`
	SignedPreKeySnake      []byte   `json:"signed_pre_key,omitempty"`
	SignedPreKeySig        []byte   `json:"signedPreKeySig,omitempty"`
	SignedPreKeySigSnake   []byte   `json:"signed_pre_key_sig,omitempty"`
	SignedPreKeyID         uint32   `json:"signedPreKeyId,omitempty"`
	SignedPreKeyIDSnake    uint32   `json:"signed_pre_key_id,omitempty"`
	OneTimePreKeys         [][]byte `json:"oneTimePreKeys,omitempty"`
	OneTimePreKeysSnake    [][]byte `json:"one_time_pre_keys,omitempty"`
}

func (r *UploadDeviceKeysRequest) Normalize() {
	if r.DeviceID == "" {
		r.DeviceID = r.DeviceIDSnake
	}
	if r.RegistrationID == 0 {
		r.RegistrationID = r.RegistrationIDSnake
	}
	if len(r.IdentityPublicKey) == 0 {
		r.IdentityPublicKey = r.IdentityPublicKeySnake
	}
	if len(r.PublicKey) == 0 {
		r.PublicKey = r.IdentityPublicKey
	}
	if len(r.IdentityPublicKey) == 0 && len(r.PublicKey) > 0 {
		r.IdentityPublicKey = r.PublicKey
	}
	if len(r.SignedPreKey) == 0 {
		r.SignedPreKey = r.SignedPreKeySnake
	}
	if len(r.SignedPreKeySig) == 0 {
		r.SignedPreKeySig = r.SignedPreKeySigSnake
	}
	if r.SignedPreKeyID == 0 {
		r.SignedPreKeyID = r.SignedPreKeyIDSnake
	}
	if len(r.OneTimePreKeys) == 0 {
		r.OneTimePreKeys = r.OneTimePreKeysSnake
	}
}

// UserPreKeysResponse - Klucze publiczne użytkownika do zestawienia sesji E2EE (X3DH)
type UserPreKeysResponse struct {
	UserID          uuid.UUID `json:"user_id"`
	DeviceID        string    `json:"device_id"`
	IdentityKey     []byte    `json:"identity_key"`
	SignedPreKey    []byte    `json:"signed_pre_key"`
	SignedPreKeySig []byte    `json:"signed_pre_key_sig"`
	SignedPreKeyID  uint32    `json:"signed_pre_key_id"`
	OneTimePreKey   []byte    `json:"one_time_pre_key,omitempty"`
	OneTimePreKeyID uint32    `json:"one_time_pre_key_id,omitempty"`
}

// PreKeyBundleDto - Bundle X3DH wymagany przez Signal Protocol po stronie klienta.
type PreKeyBundleDto struct {
	RegistrationID        uint32  `json:"registrationId"`
	DeviceID              string  `json:"deviceId"`
	PreKeyID              *uint32 `json:"preKeyId,omitempty"`
	PreKeyPublic          []byte  `json:"preKeyPublic,omitempty"`
	SignedPreKeyID        uint32  `json:"signedPreKeyId"`
	SignedPreKeyPublic    []byte  `json:"signedPreKeyPublic"`
	SignedPreKeySignature []byte  `json:"signedPreKeySignature"`
	IdentityKey           []byte  `json:"identityKey"`
}

// SignalCiphertextEnvelope – transportowa koperta E2EE dla wiadomości wysyłanych do serwera.
// Backend nie dekoduje jej treści; służy wyłącznie jako bezpieczny kanał przekazu i walidacji device binding.
type SignalCiphertextEnvelope struct {
	Type                   uint8     `json:"type,omitempty"`
	TypeSnake              uint8     `json:"signal_message_type,omitempty"`
	Ciphertext             []byte    `json:"ciphertext,omitempty"`
	SenderDeviceID         string    `json:"senderDeviceId,omitempty"`
	SenderDeviceIDSnake    string    `json:"sender_device_id,omitempty"`
	RecipientUserID        uuid.UUID `json:"recipientUserId,omitempty"`
	RecipientUserIDSnake   uuid.UUID `json:"recipient_user_id,omitempty"`
	RecipientDeviceID      string    `json:"recipientDeviceId,omitempty"`
	RecipientDeviceIDSnake string    `json:"recipient_device_id,omitempty"`
}

func (r *SignalCiphertextEnvelope) Normalize() {
	if r.Type == 0 {
		r.Type = r.TypeSnake
	}
	if r.SenderDeviceID == "" {
		r.SenderDeviceID = r.SenderDeviceIDSnake
	}
	if r.RecipientUserID == uuid.Nil {
		r.RecipientUserID = r.RecipientUserIDSnake
	}
	if r.RecipientDeviceID == "" {
		r.RecipientDeviceID = r.RecipientDeviceIDSnake
	}
}

// SendMessageRequest – request dla endpointu wysyłki wiadomości z obsługą zaszyfrowanych kopert Signal.
type SendMessageRequest struct {
	ConversationID         *uuid.UUID `json:"conversationId,omitempty"`
	ConversationIDSnake    *uuid.UUID `json:"conversation_id,omitempty"`
	SenderDeviceID         string     `json:"senderDeviceId,omitempty"`
	SenderDeviceIDSnake    string     `json:"sender_device_id,omitempty"`
	RecipientUserID        uuid.UUID  `json:"recipientUserId,omitempty"`
	RecipientUserIDSnake   uuid.UUID  `json:"recipient_user_id,omitempty"`
	RecipientDeviceID      string     `json:"recipientDeviceId,omitempty"`
	RecipientDeviceIDSnake string     `json:"recipient_device_id,omitempty"`
	Ciphertext             []byte     `json:"ciphertext,omitempty"`
	Type                   uint8      `json:"type,omitempty"`
	Content                string     `json:"content,omitempty"`
}

func (r *SendMessageRequest) Normalize() {
	if r.ConversationID == nil {
		r.ConversationID = r.ConversationIDSnake
	}
	if r.SenderDeviceID == "" {
		r.SenderDeviceID = r.SenderDeviceIDSnake
	}
	if r.RecipientUserID == uuid.Nil {
		r.RecipientUserID = r.RecipientUserIDSnake
	}
	if r.RecipientDeviceID == "" {
		r.RecipientDeviceID = r.RecipientDeviceIDSnake
	}
}

// #endregion
