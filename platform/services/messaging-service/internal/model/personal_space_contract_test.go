package model

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestPersonalSpaceItemJSONUsesCamelCaseOnly(t *testing.T) {
	item := PersonalSpaceItem{
		ID:               uuid.New(),
		SpaceID:          uuid.New(),
		OwnerID:          uuid.New(),
		DeviceID:         "device-1",
		Title:            "Moja notatka",
		ContentType:      PersonalSpaceItemTypeText,
		EncryptedPayload: []byte("ciphertext"),
		Status:           PersonalSpaceItemStatusActive,
		SyncVersion:      3,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}

	payload, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("marshal personal space item: %v", err)
	}
	jsonText := string(payload)
	for _, key := range []string{"\"space_id\"", "\"owner_id\"", "\"device_id\"", "\"encrypted_payload\"", "\"created_at\""} {
		if strings.Contains(jsonText, key) {
			t.Fatalf("legacy snake_case key %s should not exist in public contract; got %s", key, jsonText)
		}
	}
	for _, key := range []string{"\"spaceId\"", "\"ownerId\"", "\"deviceId\"", "\"encryptedPayload\"", "\"createdAt\"", "\"syncVersion\""} {
		if !strings.Contains(jsonText, key) {
			t.Fatalf("expected camelCase key %s; got %s", key, jsonText)
		}
	}
}

func TestCreatePersonalSpaceItemRequestUsesCamelCaseOnly(t *testing.T) {
	payload := []byte(`{"spaceId":"` + uuid.NewString() + `","title":"test","contentType":"text","deviceId":"device-1","encryptedPayload":"Y2lwaGVydGV4dA=="}`)
	var req CreatePersonalSpaceItemRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		t.Fatalf("unmarshal personal-space request: %v", err)
	}
	if req.ContentType != PersonalSpaceItemTypeText {
		t.Fatalf("expected contentType text, got %q", req.ContentType)
	}
	if req.DeviceID != "device-1" {
		t.Fatalf("expected deviceId to be parsed, got %q", req.DeviceID)
	}
	if len(req.EncryptedPayload) == 0 {
		t.Fatal("expected encryptedPayload to be decoded")
	}
}
