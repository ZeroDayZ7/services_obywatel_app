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
	if !strings.Contains(jsonText, "\"encrypted_payload\":") {
		t.Fatalf("expected encrypted_payload in JSON, got %s", jsonText)
	}
	if !strings.Contains(jsonText, encoded) {
		t.Fatalf("expected encoded ciphertext payload in JSON, got %s", jsonText)
	}
}
