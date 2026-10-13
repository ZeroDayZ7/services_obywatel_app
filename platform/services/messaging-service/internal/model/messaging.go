package model

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// #region Base Models
// BaseModel – Standardowe pola audytowe z indykatorem wersji dla Delta Sync
type BaseModel struct {
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deletedAt,omitempty"`
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
	UserID         uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_user_device_identity_active"`
	DeviceID       string    `gorm:"type:varchar(64);not null;uniqueIndex:idx_user_device_identity_active"` // Identyfikator instalacji / urządzenia aplikacji (UUID)
	SignalDeviceID uint32    `gorm:"not null;default:1"`                                                    // Numeryczny identyfikator urządzenia Signal dla sesji protokołu, zawsze liczba > 0
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
	PublishedAt time.Time `json:"publishedAt"`
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
	DeviceID     string `json:"deviceId,omitempty"`
	TermsVersion string `json:"termsVersion,omitempty"`
	Consent      bool   `json:"consent,omitempty"`
}

type AcceptTermsRequest struct {
	DeviceID     string `json:"deviceId,omitempty"`
	TermsVersion string `json:"termsVersion,omitempty"`
}

type MessagingActivationStatusResponse struct {
	UserID                  uuid.UUID        `json:"userId"`
	Status                  ActivationStatus `json:"status"`
	ConsentAccepted         bool             `json:"consentAccepted"`
	TermsVersion            string           `json:"termsVersion,omitempty"`
	CurrentTermsVersion     string           `json:"currentTermsVersion,omitempty"`
	RequiresTermsAcceptance bool             `json:"requiresTermsAcceptance"`
	DeviceID                string           `json:"deviceId,omitempty"`
	ActivatedAt             *time.Time       `json:"activatedAt,omitempty"`
	CreatedAt               time.Time        `json:"createdAt"`
	UpdatedAt               time.Time        `json:"updatedAt"`
}

func (a *MessagingActivation) ToResponse() *MessagingActivationStatusResponse {
	if a == nil {
		return nil
	}
	currentTerms := CurrentMessagingTerms()
	requiresTermsAcceptance := a.Status != ActivationStatusActive || !a.ConsentAccepted || a.TermsVersion != currentTerms.Version
	return &MessagingActivationStatusResponse{
		UserID:                  a.UserID,
		Status:                  a.Status,
		ConsentAccepted:         a.ConsentAccepted,
		TermsVersion:            a.TermsVersion,
		CurrentTermsVersion:     currentTerms.Version,
		RequiresTermsAcceptance: requiresTermsAcceptance,
		DeviceID:                a.DeviceID,
		ActivatedAt:             a.ActivatedAt,
		CreatedAt:               a.CreatedAt,
		UpdatedAt:               a.UpdatedAt,
	}
}

// #endregion

// #region Conversation & Message Entities
// Conversation – Konwersacja prywatna lub grupowa
type Conversation struct {
	ID    uuid.UUID        `gorm:"type:uuid;primaryKey;default:uuidv7()" json:"id"`
	Type  ConversationType `gorm:"type:varchar(20);not null;default:'direct'" json:"type"`
	Title string           `gorm:"type:varchar(128)" json:"title,omitempty"` // Wypełniane tylko dla grup (może być zaszyfrowane)

	// Ostatnia sekwencja wiadomości w konwersacji (zwiększana monotonicznie przy każdej wiadomości)
	LastSequence uint64 `gorm:"not null;default:0" json:"lastSequence"`

	// Relacje GORM
	Members  []ConversationMember `gorm:"foreignKey:ConversationID" json:"members,omitempty"`
	Messages []Message            `gorm:"foreignKey:ConversationID" json:"messages,omitempty"`

	BaseModel
}

// ConversationMember – Uczestnicy danej konwersacji i ich stan przeczytania
type ConversationMember struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:uuidv7()" json:"id"`
	ConversationID uuid.UUID `gorm:"type:uuid;index:idx_conv_user,unique;not null" json:"conversationId"`
	UserID         uuid.UUID `gorm:"type:uuid;index:idx_conv_user,unique;not null" json:"userId"`
	Role           string    `gorm:"type:varchar(20);default:'member'" json:"role"` // 'admin', 'member'

	// Wskaźnik synchronizacji: ostatnia odebrana/przeczytana sekwencja wiadomości przez tego użytkownika
	LastReadSequence uint64 `gorm:"not null;default:0" json:"lastReadSequence"`

	BaseModel
}

// Message – Zaszyfrowana koperta z wiadomością (Payload E2EE jest nieczytelny dla serwera)
type Message struct {
	ID             uuid.UUID   `gorm:"type:uuid;primaryKey;default:uuidv7()" json:"id,omitempty"`
	IdempotencyKey string      `gorm:"type:varchar(128);index:idx_message_idempotency,unique;not null;default:''" json:"idempotencyKey,omitempty"`
	ConversationID uuid.UUID   `gorm:"type:uuid;index:idx_conv_seq,unique;not null" json:"conversationId,omitempty"`
	SenderID       uuid.UUID   `gorm:"type:uuid;index;not null" json:"senderId,omitempty"`
	SenderDeviceID string      `gorm:"type:varchar(64);not null" json:"senderDeviceId,omitempty"`
	Type           MessageType `gorm:"type:varchar(20);not null;default:'text'" json:"type,omitempty"`

	// Monotoniczny numer sekwencyjny w ramach danej konwersacji (służy do sortowania i synchronizacji delta)
	Sequence uint64 `gorm:"index:idx_conv_seq,unique;not null" json:"sequence,omitempty"`

	// Szyfrowany ładunek wiadomości (AES-GCM / Signal Protocol Payload)
	// Serwer widzi wyłącznie ciąg bajtów i nie ma możliwości jego odszyfrowania.
	// Pole jest jawnie serializowane do API, by Flutter mógł zdeserializować go jako ciphertext/encryptedPayload.
	EncryptedPayload []byte `gorm:"type:bytea;not null" json:"encryptedPayload,omitempty"`

	// Odnośniki do załączników (dla wiadomości typu media)
	MediaHeader []byte `gorm:"type:bytea" json:"mediaHeader,omitempty"`

	// Globalny wskaźnik wersji w mikroserwisie dla synchronizacji offline -> online
	Version uint64 `gorm:"not null;default:1;index" json:"version,omitempty"`

	// Relacja do konwersacji
	Conversation *Conversation `gorm:"foreignKey:ConversationID;constraint:OnDelete:CASCADE" json:"conversation,omitempty"`

	BaseModel
}

func (m Message) MarshalJSON() ([]byte, error) {
	payload := map[string]any{
		"id":               m.ID,
		"idempotencyKey":   m.IdempotencyKey,
		"conversationId":   m.ConversationID,
		"senderId":         m.SenderID,
		"senderDeviceId":   m.SenderDeviceID,
		"type":             m.Type,
		"sequence":         m.Sequence,
		"encryptedPayload": base64.StdEncoding.EncodeToString(m.EncryptedPayload),
		"mediaHeader":      m.MediaHeader,
		"version":          m.Version,
		"createdAt":        m.CreatedAt,
		"updatedAt":        m.UpdatedAt,
	}
	return json.Marshal(payload)
}

// #endregion

// #region Persistent Encrypted Vault Contracts
// MessageEnvelope – transportowa i archiwalna koperta zaszyfrowanej wiadomości.
// Serwer nie dekoduje treści, a jedynie przechowuje ciphertext i metadane retencji.
type MessageEnvelope struct {
	MessageID         string     `json:"messageId,omitempty"`
	ConversationID    uuid.UUID  `json:"conversationId,omitempty"`
	SenderUserID      uuid.UUID  `json:"senderUserId,omitempty"`
	SenderDeviceID    string     `json:"senderDeviceId,omitempty"`
	RecipientUserID   uuid.UUID  `json:"recipientUserId,omitempty"`
	RecipientDeviceID string     `json:"recipientDeviceId,omitempty"`
	Ciphertext        []byte     `json:"ciphertext,omitempty"`
	Type              uint8      `json:"type,omitempty"`
	Nonce             []byte     `json:"nonce,omitempty"`
	CreatedAt         time.Time  `json:"createdAt,omitempty"`
	ExpiresAt         *time.Time `json:"expiresAt,omitempty"`
	Version           uint64     `json:"version,omitempty"`
}

// MessageRecord – zapis chronologii wiadomości w przechowalni zaszyfrowanych kopert.
type MessageRecord struct {
	ID                uuid.UUID   `json:"id,omitempty"`
	MessageID         uuid.UUID   `json:"messageId,omitempty"`
	ConversationID    uuid.UUID   `json:"conversationId,omitempty"`
	SenderUserID      uuid.UUID   `json:"senderUserId,omitempty"`
	SenderDeviceID    string      `json:"senderDeviceId,omitempty"`
	RecipientUserID   uuid.UUID   `json:"recipientUserId,omitempty"`
	RecipientDeviceID string      `json:"recipientDeviceId,omitempty"`
	Ciphertext        []byte      `json:"ciphertext,omitempty"`
	Type              MessageType `json:"type,omitempty"`
	CreatedAt         time.Time   `json:"createdAt,omitempty"`
	ExpiresAt         *time.Time  `json:"expiresAt,omitempty"`
	IsDelivered       bool        `json:"isDelivered,omitempty"`
	Version           uint64      `json:"version,omitempty"`
}

// HistoryFetchRequest – żądanie pobrania historii wiadomości z szyfrowanego archiwum.
type HistoryFetchRequest struct {
	ConversationID *uuid.UUID `json:"conversationId,omitempty"`
	Since          uint64     `json:"since,omitempty"`
	Limit          int        `json:"limit,omitempty"`
}

// NewMessageNotification – lekkie powiadomienie WebSocket o nowej wiadomości z archiwum i message_id.
type NewMessageNotification struct {
	EventType      string    `json:"eventType,omitempty"`
	ConversationID uuid.UUID `json:"conversationId,omitempty"`
	MessageID      string    `json:"messageId,omitempty"`
	SenderUserID   uuid.UUID `json:"senderUserId,omitempty"`
	SenderDeviceID string    `json:"senderDeviceId,omitempty"`
	Timestamp      time.Time `json:"timestamp,omitempty"`
}

// #endregion

// #region Sync & Outbox DTOs
// SyncDeltaRequest – Żądanie synchronizacji różnicowej wysyłane z aplikacji mobilnej.
// `last_known_sequence` jest globalnym kursorem zmian dla użytkownika i jest preferowanym polem
// w nowym modelu Offline-First; stare pola wersji są zachowane dla kompatybilności wstecznej.
type SyncDeltaRequest struct {
	LastKnownSequence       uint64 `json:"lastKnownSequence,omitempty"`
	LastKnownContactVersion uint64 `json:"lastKnownContactVersion,omitempty"`
	LastKnownMessageVersion uint64 `json:"lastKnownMessageVersion,omitempty"`
	IdempotencyKey          string `json:"idempotencyKey,omitempty"`
	Limit                   int    `json:"limit,omitempty"`
}

// SyncDeltaResponse – Paczka zmian do zaaplikowania w lokalnej bazie Drift/SQLite.
type SyncDeltaResponse struct {
	UpdatedContacts       []Contact `json:"updatedContacts"`
	NewMessages           []Message `json:"newMessages"`
	NextSequence          uint64    `json:"nextSequence,omitempty"`
	HasMore               bool      `json:"hasMore"`
	AppliedIdempotencyKey string    `json:"appliedIdempotencyKey,omitempty"`
}

// OutboxEventPayload – Struktura kolejkowana w lokalnej bazie urządzenia w trybie Offline
type OutboxEventPayload struct {
	EventID        uuid.UUID      `json:"eventId,omitempty"`
	IdempotencyKey string         `json:"idempotencyKey,omitempty"`
	EventType      string         `json:"eventType,omitempty"` // "SEND_MESSAGE", "ADD_CONTACT"
	ConversationID *uuid.UUID     `json:"conversationId,omitempty"`
	Payload        datatypes.JSON `json:"payload,omitempty"`
	CreatedAt      time.Time      `json:"createdAt,omitempty"`
	MessageID      string         `json:"messageId,omitempty"`
	Ciphertext     string         `json:"ciphertext,omitempty"`
	SenderDeviceID string         `json:"senderDeviceId,omitempty"`
	Content        string         `json:"content,omitempty"`
	Type           uint8          `json:"type,omitempty"`
}

// OutboxBatchRequest - Paczka zdarzeń wysyłana z klienta w trybie offline
type OutboxBatchRequest struct {
	Messages []OutboxEventPayload `json:"messages"`
}

// OutboxBatchResponse - Wynik przetworzenia paczki outbox
type OutboxBatchResponse struct {
	ProcessedCount int `json:"processedCount"`
}

// #endregion

// #region Conversation & Crypto DTOs
// CreateConversationRequest - Żądanie utworzenia nowej konwersacji
type CreateConversationRequest struct {
	Type         ConversationType `json:"type"`
	Title        string           `json:"title,omitempty"`
	RecipientIDs []uuid.UUID      `json:"recipientIds"`
}

// OneTimePreKeyUpload preserves the exact Signal keyId from the client so the
// backend can bind the correct private pre-key to the correct message.
type OneTimePreKeyUpload struct {
	KeyID     uint32 `json:"keyId"`
	PublicKey []byte `json:"publicKey"`
}

func (k *OneTimePreKeyUpload) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	keyIDValue, ok := raw["keyId"]
	if !ok {
		keyIDValue, ok = raw["key_id"]
	}
	if ok {
		var keyID uint32
		if err := json.Unmarshal(keyIDValue, &keyID); err != nil {
			return err
		}
		k.KeyID = keyID
	}

	publicKeyValue, ok := raw["publicKey"]
	if !ok {
		publicKeyValue, ok = raw["public_key"]
	}
	if ok {
		var publicKey []byte
		if err := json.Unmarshal(publicKeyValue, &publicKey); err != nil {
			return err
		}
		k.PublicKey = publicKey
	}

	return nil
}

// UploadDeviceKeysRequest - Rejestracja kluczy E2EE dla urządzenia.
type UploadDeviceKeysRequest struct {
	DeviceID          string                `json:"deviceId,omitempty"`
	SignalDeviceID    uint32                `json:"signalDeviceId,omitempty"`
	RegistrationID    uint32                `json:"registrationId,omitempty"`
	IdentityPublicKey []byte                `json:"identityPublicKey,omitempty"`
	PublicKey         []byte                `json:"publicKey,omitempty"`
	SignedPreKey      []byte                `json:"signedPreKey,omitempty"`
	SignedPreKeySig   []byte                `json:"signedPreKeySig,omitempty"`
	SignedPreKeyID    uint32                `json:"signedPreKeyId,omitempty"`
	OneTimePreKeys    []OneTimePreKeyUpload `json:"oneTimePreKeys,omitempty"`
}

func (r *UploadDeviceKeysRequest) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	decodeString := func(candidates ...string) (string, bool) {
		for _, key := range candidates {
			if value, ok := raw[key]; ok {
				var decoded string
				if err := json.Unmarshal(value, &decoded); err != nil {
					return "", false
				}
				return decoded, true
			}
		}
		return "", false
	}
	decodeBytes := func(candidates ...string) ([]byte, bool) {
		for _, key := range candidates {
			if value, ok := raw[key]; ok {
				var decoded []byte
				if err := json.Unmarshal(value, &decoded); err != nil {
					return nil, false
				}
				return decoded, true
			}
		}
		return nil, false
	}
	decodeUint32 := func(candidates ...string) (uint32, bool) {
		for _, key := range candidates {
			if value, ok := raw[key]; ok {
				var decoded uint32
				if err := json.Unmarshal(value, &decoded); err != nil {
					return 0, false
				}
				return decoded, true
			}
		}
		return 0, false
	}
	decodeNestedPreKeys := func(candidates ...string) ([]OneTimePreKeyUpload, bool) {
		for _, key := range candidates {
			if value, ok := raw[key]; ok {
				var decoded []OneTimePreKeyUpload
				if err := json.Unmarshal(value, &decoded); err != nil {
					return nil, false
				}
				return decoded, true
			}
		}
		return nil, false
	}

	if deviceID, ok := decodeString("deviceId", "device_id"); ok {
		r.DeviceID = deviceID
	}
	if signalDeviceID, ok := decodeUint32("signalDeviceId", "signal_device_id"); ok {
		r.SignalDeviceID = signalDeviceID
	}
	if registrationID, ok := decodeUint32("registrationId", "registration_id"); ok {
		r.RegistrationID = registrationID
	}
	if identityPublicKey, ok := decodeBytes("identityPublicKey", "identity_public_key"); ok {
		r.IdentityPublicKey = identityPublicKey
	}
	if publicKey, ok := decodeBytes("publicKey", "public_key"); ok {
		r.PublicKey = publicKey
	}
	if signedPreKey, ok := decodeBytes("signedPreKey", "signed_pre_key"); ok {
		r.SignedPreKey = signedPreKey
	}
	if signedPreKeySig, ok := decodeBytes("signedPreKeySig", "signed_pre_key_sig"); ok {
		r.SignedPreKeySig = signedPreKeySig
	}
	if signedPreKeyID, ok := decodeUint32("signedPreKeyId", "signed_pre_key_id"); ok {
		r.SignedPreKeyID = signedPreKeyID
	}
	if oneTimePreKeys, ok := decodeNestedPreKeys("oneTimePreKeys", "one_time_pre_keys"); ok {
		r.OneTimePreKeys = oneTimePreKeys
	}

	if len(r.IdentityPublicKey) == 0 && len(r.PublicKey) > 0 {
		r.IdentityPublicKey = r.PublicKey
	}
	if len(r.PublicKey) == 0 && len(r.IdentityPublicKey) > 0 {
		r.PublicKey = r.IdentityPublicKey
	}

	return nil
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
	Type              uint8     `json:"type,omitempty"`
	Ciphertext        []byte    `json:"ciphertext,omitempty"`
	SenderDeviceID    string    `json:"senderDeviceId,omitempty"`
	RecipientUserID   uuid.UUID `json:"recipientUserId,omitempty"`
	RecipientDeviceID string    `json:"recipientDeviceId,omitempty"`
}

// SendMessageRequest – request dla endpointu wysyłki wiadomości z obsługą zaszyfrowanych kopert Signal.
type SendMessageRequest struct {
	MessageID         string     `json:"messageId,omitempty"`
	ConversationID    *uuid.UUID `json:"conversationId,omitempty"`
	SenderDeviceID    string     `json:"senderDeviceId,omitempty"`
	RecipientUserID   uuid.UUID  `json:"recipientUserId,omitempty"`
	RecipientDeviceID string     `json:"recipientDeviceId,omitempty"`
	Ciphertext        []byte     `json:"ciphertext,omitempty"`
	Type              uint8      `json:"type,omitempty"`
	Content           string     `json:"content,omitempty"`
	IdempotencyKey    string     `json:"idempotencyKey,omitempty"`
}

// #endregion
