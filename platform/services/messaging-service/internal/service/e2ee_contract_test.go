package service

import (
	"testing"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/services/messaging-service/internal/model"
)

func TestUploadDeviceKeysRequestNormalizeAcceptsSignalStyleFields(t *testing.T) {
	req := model.UploadDeviceKeysRequest{
		DeviceIDSnake:          "device-123",
		RegistrationIDSnake:    42,
		IdentityPublicKeySnake: []byte("identity-key"),
		SignedPreKeySnake:      []byte("signed-pre-key"),
		SignedPreKeySigSnake:   []byte("signature"),
		SignedPreKeyIDSnake:    9,
		OneTimePreKeysSnake:    [][]byte{[]byte("otk-1"), []byte("otk-2")},
	}

	req.Normalize()

	if req.DeviceID != "device-123" {
		t.Fatalf("expected normalized device ID, got %q", req.DeviceID)
	}
	if req.RegistrationID != 42 {
		t.Fatalf("expected normalized registration ID 42, got %d", req.RegistrationID)
	}
	if len(req.PublicKey) == 0 {
		t.Fatal("expected public key to be populated")
	}
	if req.SignedPreKeyID != 9 {
		t.Fatalf("expected signed prekey ID 9, got %d", req.SignedPreKeyID)
	}
	if len(req.OneTimePreKeys) != 2 {
		t.Fatalf("expected 2 one-time prekeys, got %d", len(req.OneTimePreKeys))
	}
}

func TestBuildPreKeyBundleIncludesOneTimePreKey(t *testing.T) {
	userID := uuid.New()
	identity := &model.UserDeviceIdentity{
		ID:              uuid.New(),
		UserID:          userID,
		DeviceID:        "device-123",
		RegistrationID:  42,
		PublicKey:       []byte("identity-key"),
		SignedPreKey:    []byte("signed-pre-key"),
		SignedPreKeySig: []byte("signed-signature"),
		SignedPreKeyID:  9,
	}
	preKey := &model.UserPreKey{KeyID: 7, PublicKey: []byte("otk-value")}

	bundle := BuildPreKeyBundle(identity, preKey)
	if bundle == nil {
		t.Fatal("bundle should not be nil")
	}
	if bundle.RegistrationID != 42 {
		t.Fatalf("expected registration ID 42, got %d", bundle.RegistrationID)
	}
	if bundle.PreKeyID == nil || *bundle.PreKeyID != 7 {
		t.Fatalf("expected one-time prekey id 7, got %#v", bundle.PreKeyID)
	}
	if string(bundle.PreKeyPublic) != "otk-value" {
		t.Fatalf("expected one-time prekey payload to match, got %q", string(bundle.PreKeyPublic))
	}
	if len(bundle.SignedPreKeySignature) == 0 {
		t.Fatal("expected signed prekey signature to be set")
	}
}

func TestBuildPreKeyBundleWithoutOneTimePreKey(t *testing.T) {
	identity := &model.UserDeviceIdentity{
		ID:              uuid.New(),
		UserID:          uuid.New(),
		DeviceID:        "device-456",
		RegistrationID:  7,
		PublicKey:       []byte("identity-key"),
		SignedPreKey:    []byte("signed-pre-key"),
		SignedPreKeySig: []byte("signed-signature"),
		SignedPreKeyID:  11,
	}

	bundle := BuildPreKeyBundle(identity, nil)
	if bundle == nil {
		t.Fatal("bundle should not be nil")
	}
	if bundle.PreKeyID != nil {
		t.Fatalf("expected no one-time prekey, got %#v", *bundle.PreKeyID)
	}
	if len(bundle.PreKeyPublic) != 0 {
		t.Fatalf("expected empty prekey public payload, got %q", string(bundle.PreKeyPublic))
	}
}
