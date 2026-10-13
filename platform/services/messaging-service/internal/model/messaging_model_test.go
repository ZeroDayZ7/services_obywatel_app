package model

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestMessageJSONIncludesEncryptedPayloadForSync(t *testing.T) {
	msg := Message{
		ID:               uuid.New(),
		ConversationID:   uuid.New(),
		SenderID:         uuid.New(),
		SenderDeviceID:   "device-1",
		Type:             MessageTypeText,
		EncryptedPayload: []byte("signal-ciphertext"),
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal message: %v", err)
	}

	jsonText := string(payload)
	encoded := base64.StdEncoding.EncodeToString([]byte("signal-ciphertext"))
	if !strings.Contains(jsonText, "\"encryptedPayload\":") {
		t.Fatalf("expected encryptedPayload in JSON, got %s", jsonText)
	}
	if !strings.Contains(jsonText, encoded) {
		t.Fatalf("expected encoded ciphertext payload in JSON, got %s", jsonText)
	}
}

func TestMessageJSONUsesCamelCaseOnly(t *testing.T) {
	msg := Message{
		ID:               uuid.New(),
		IdempotencyKey:   "idem-1",
		ConversationID:   uuid.New(),
		SenderID:         uuid.New(),
		SenderDeviceID:   "device-1",
		Type:             MessageTypeText,
		EncryptedPayload: []byte("signal-ciphertext"),
		Sequence:         7,
		Version:          2,
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal message: %v", err)
	}

	jsonText := string(payload)
	for _, key := range []string{"\"idempotency_key\"", "\"conversation_id\"", "\"sender_device_id\"", "\"encrypted_payload\"", "\"created_at\""} {
		if strings.Contains(jsonText, key) {
			t.Fatalf("legacy snake_case key %s should not be serialized in camelCase contract; got %s", key, jsonText)
		}
	}
	for _, key := range []string{"\"idempotencyKey\"", "\"conversationId\"", "\"senderDeviceId\"", "\"encryptedPayload\"", "\"createdAt\""} {
		if !strings.Contains(jsonText, key) {
			t.Fatalf("expected camelCase key %s; got %s", key, jsonText)
		}
	}
}

func TestMessageEnvelopeJSONUsesCamelCaseOnly(t *testing.T) {
	envelope := MessageEnvelope{
		MessageID:         uuid.NewString(),
		ConversationID:    uuid.New(),
		SenderUserID:      uuid.New(),
		SenderDeviceID:    "device-1",
		RecipientUserID:   uuid.New(),
		RecipientDeviceID: "device-2",
		Ciphertext:        []byte("ciphertext"),
		Type:              3,
		Nonce:             []byte("nonce"),
	}

	payload, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}

	jsonText := string(payload)
	for _, key := range []string{"\"message_id\"", "\"conversation_id\"", "\"sender_device_id\"", "\"recipient_device_id\"", "\"signal_message_type\""} {
		if strings.Contains(jsonText, key) {
			t.Fatalf("legacy snake_case key %s should not be serialized in camelCase contract; got %s", key, jsonText)
		}
	}
	if !strings.Contains(jsonText, "\"messageId\"") {
		t.Fatalf("expected camelCase messageId; got %s", jsonText)
	}
	if !strings.Contains(jsonText, "\"conversationId\"") {
		t.Fatalf("expected camelCase conversationId; got %s", jsonText)
	}
}
