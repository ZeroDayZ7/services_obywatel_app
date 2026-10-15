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
type UserDeviceIdentity struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:uuidv7()" json:"id"`
	UserID         uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_user_device_identity_active" json:"userId"`
	DeviceID       string    `gorm:"type:varchar(64);not null;uniqueIndex:idx_user_device_identity_active" json:"deviceId"`
	SignalDeviceID uint32    `gorm:"not null;default:0" json:"signalDeviceId"`
	RegistrationID uint32    `gorm:"not null;default:0" json:"registrationId"`
	PublicKey      []byte    `gorm:"type:bytea;not null" json:"publicKey"`

	SignedPreKey    []byte `gorm:"type:bytea;not null" json:"signedPreKey"`
	SignedPreKeySig []byte `gorm:"type:bytea;not null" json:"signedPreKeySig"`
	SignedPreKeyID  uint32 `gorm:"not null" json:"signedPreKeyId"`

	OneTimePreKeysCount int `gorm:"not null;default:0" json:"oneTimePreKeysCount"`

	BaseModel
}

type UserPreKey struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuidv7()" json:"id"`
	DeviceID  uuid.UUID `gorm:"type:uuid;index;not null" json:"deviceId"`
	KeyID     uint32    `gorm:"not null;index" json:"keyId"`
	PublicKey []byte    `gorm:"type:bytea;not null" json:"publicKey"`

	BaseModel
}

type ActivationStatus string

const (
	ActivationStatusNotStarted ActivationStatus = "not_started"
	ActivationStatusPending    ActivationStatus = "pending"
	ActivationStatusActive     ActivationStatus = "active"
	ActivationStatusDisabled   ActivationStatus = "disabled"
)

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

type MessagingActivation struct {
	ID              uuid.UUID        `gorm:"type:uuid;primaryKey;default:uuidv7()" json:"id"`
	UserID          uuid.UUID        `gorm:"type:uuid;uniqueIndex:idx_msg_activation_user;not null" json:"userId"`
	DeviceID        string           `gorm:"type:varchar(64);not null;default:''" json:"deviceId"`
	Status          ActivationStatus `gorm:"type:varchar(32);not null;default:'not_started';index" json:"status"`
	ConsentAccepted bool             `gorm:"not null;default:false" json:"consentAccepted"`
	TermsVersion    string           `gorm:"type:varchar(32);not null;default:''" json:"termsVersion"`
	ActivatedAt     *time.Time       `gorm:"index" json:"activatedAt,omitempty"`
	LastSeenAt      *time.Time       `gorm:"index" json:"lastSeenAt,omitempty"`

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
type Conversation struct {
	ID           uuid.UUID        `gorm:"type:uuid;primaryKey;default:uuidv7()" json:"id"`
	Type         ConversationType `gorm:"type:varchar(20);not null;default:'direct'" json:"type"`
	Title        string           `gorm:"type:varchar(128)" json:"title,omitempty"`
	LastSequence uint64           `gorm:"not null;default:0" json:"lastSequence"`

	Members  []ConversationMember `gorm:"foreignKey:ConversationID" json:"members,omitempty"`
	Messages []Message            `gorm:"foreignKey:ConversationID" json:"messages,omitempty"`

	BaseModel
}

type ConversationMember struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:uuidv7()" json:"id"`
	ConversationID uuid.UUID `gorm:"type:uuid;index:idx_conv_user,unique;not null" json:"conversationId"`
	UserID         uuid.UUID `gorm:"type:uuid;index:idx_conv_user,unique;not null" json:"userId"`
	Role           string    `gorm:"type:varchar(20);default:'member'" json:"role"`

	LastReadSequence uint64 `gorm:"not null;default:0" json:"lastReadSequence"`

	BaseModel
}

type Message struct {
	ID             uuid.UUID   `gorm:"type:uuid;primaryKey;default:uuidv7()" json:"id,omitempty"`
	IdempotencyKey string      `gorm:"type:varchar(128);index:idx_message_idempotency,unique;not null;default:''" json:"idempotencyKey,omitempty"`
	ConversationID uuid.UUID   `gorm:"type:uuid;index:idx_conv_seq,unique;not null" json:"conversationId,omitempty"`
	SenderID       uuid.UUID   `gorm:"type:uuid;index;not null" json:"senderId,omitempty"`
	SenderDeviceID string      `gorm:"type:varchar(64);not null" json:"senderDeviceId,omitempty"`
	AttachmentID   *uuid.UUID  `gorm:"type:uuid;index" json:"attachmentId,omitempty"`
	Type           MessageType `gorm:"type:varchar(20);not null;default:'text'" json:"type,omitempty"`
	Sequence       uint64      `gorm:"index:idx_conv_seq,unique;not null" json:"sequence,omitempty"`

	EncryptedPayload []byte `gorm:"type:bytea;not null" json:"encryptedPayload,omitempty"`
	MediaHeader      []byte `gorm:"type:bytea" json:"mediaHeader,omitempty"`
	Version          uint64 `gorm:"not null;default:1;index" json:"version,omitempty"`

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
		"attachmentId":     m.AttachmentID,
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

type HistoryFetchRequest struct {
	ConversationID *uuid.UUID `json:"conversationId,omitempty"`
	Since          uint64     `json:"since,omitempty"`
	Limit          int        `json:"limit,omitempty"`
}

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
type SyncDeltaRequest struct {
	LastKnownSequence       uint64 `json:"lastKnownSequence,omitempty"`
	LastKnownContactVersion uint64 `json:"lastKnownContactVersion,omitempty"`
	LastKnownMessageVersion uint64 `json:"lastKnownMessageVersion,omitempty"`
	IdempotencyKey          string `json:"idempotencyKey,omitempty"`
	Limit                   int    `json:"limit,omitempty"`
}

type SyncDeltaResponse struct {
	UpdatedContacts       []Contact `json:"updatedContacts"`
	NewMessages           []Message `json:"newMessages"`
	NextSequence          uint64    `json:"nextSequence,omitempty"`
	HasMore               bool      `json:"hasMore"`
	AppliedIdempotencyKey string    `json:"appliedIdempotencyKey,omitempty"`
}

type OutboxEventPayload struct {
	EventID        uuid.UUID      `json:"eventId,omitempty"`
	IdempotencyKey string         `json:"idempotencyKey,omitempty"`
	EventType      string         `json:"eventType,omitempty"`
	ConversationID *uuid.UUID     `json:"conversationId,omitempty"`
	AttachmentID   *uuid.UUID     `json:"attachmentId,omitempty"`
	Payload        datatypes.JSON `json:"payload,omitempty"`
	CreatedAt      time.Time      `json:"createdAt,omitempty"`
	MessageID      string         `json:"messageId,omitempty"`
	Ciphertext     string         `json:"ciphertext,omitempty"`
	SenderDeviceID string         `json:"senderDeviceId,omitempty"`
	Content        string         `json:"content,omitempty"`
	Type           uint8          `json:"type,omitempty"`
}

type OutboxBatchRequest struct {
	Messages []OutboxEventPayload `json:"messages"`
}

type OutboxBatchResponse struct {
	ProcessedCount int `json:"processedCount"`
}

// #endregion

// #region Conversation & Crypto DTOs
type CreateConversationRequest struct {
	Type         ConversationType `json:"type"`
	Title        string           `json:"title,omitempty"`
	RecipientIDs []uuid.UUID      `json:"recipientIds"`
}

// OneTimePreKeyUpload – Czysty model camelCase bez zbędnego boilerplate'u
type OneTimePreKeyUpload struct {
	KeyID     uint32 `json:"keyId"`
	PublicKey []byte `json:"publicKey"`
}

// UploadDeviceKeysRequest – Czysty model camelCase bez niestandardowego UnmarshalJSON
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

type SignalCiphertextEnvelope struct {
	Type              uint8     `json:"type,omitempty"`
	Ciphertext        []byte    `json:"ciphertext,omitempty"`
	SenderDeviceID    string    `json:"senderDeviceId,omitempty"`
	RecipientUserID   uuid.UUID `json:"recipientUserId,omitempty"`
	RecipientDeviceID string    `json:"recipientDeviceId,omitempty"`
}

type SendMessageRequest struct {
	MessageID         string     `json:"messageId,omitempty"`
	ConversationID    *uuid.UUID `json:"conversationId,omitempty"`
	SenderDeviceID    string     `json:"senderDeviceId,omitempty"`
	RecipientUserID   uuid.UUID  `json:"recipientUserId,omitempty"`
	RecipientDeviceID string     `json:"recipientDeviceId,omitempty"`
	AttachmentID      *uuid.UUID `json:"attachmentId,omitempty"`
	Ciphertext        []byte     `json:"ciphertext,omitempty"`
	Type              uint8      `json:"type,omitempty"`
	Content           string     `json:"content,omitempty"`
	IdempotencyKey    string     `json:"idempotencyKey,omitempty"`
}

// #endregion
