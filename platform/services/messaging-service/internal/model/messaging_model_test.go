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

func TestUploadDeviceKeysRequestAcceptsLegacySnakeCaseAndCamelCase(t *testing.T) {
	payload := []byte(`{
		"device_id": "device-1",
		"registration_id": 42,
		"identity_public_key": "aWQtcHVibGljLWtleQ==",
		"public_key": "aWQtcHVibGljLWtleQ==",
		"signed_pre_key": "c2lnbmVkLXByZWs=",
		"signed_pre_key_sig": "c2lnbmVkLXNpZw==",
		"signed_pre_key_id": 9,
		"one_time_pre_keys": [{"key_id": 1, "public_key": "b3RrLTE="}]
	}`)

	var req UploadDeviceKeysRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		t.Fatalf("unmarshal legacy snake_case payload: %v", err)
	}
	if req.DeviceID != "device-1" {
		t.Fatalf("expected deviceId to be parsed, got %q", req.DeviceID)
	}
	if req.RegistrationID != 42 {
		t.Fatalf("expected registrationId 42, got %d", req.RegistrationID)
	}
	if len(req.IdentityPublicKey) == 0 {
		t.Fatal("expected identityPublicKey to be populated from legacy payload")
	}
	if len(req.OneTimePreKeys) != 1 {
		t.Fatalf("expected 1 one-time prekey, got %d", len(req.OneTimePreKeys))
	}
	if req.OneTimePreKeys[0].KeyID != 1 {
		t.Fatalf("expected keyId 1, got %d", req.OneTimePreKeys[0].KeyID)
	}
}
