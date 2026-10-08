package service

import (
	"bytes"
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

func TestRejectsFakeSignalKeyMaterial(t *testing.T) {
	fake := model.UploadDeviceKeysRequest{
		DeviceID:          "device-123",
		RegistrationID:    42,
		IdentityPublicKey: []byte("PUBKEY_JAN_IDENTITY"),
		SignedPreKey:      []byte("SIGNED_PREKEY_JAN"),
		SignedPreKeySig:   bytes.Repeat([]byte{'A'}, 64),
		SignedPreKeyID:    9,
		OneTimePreKeys:    [][]byte{[]byte("OTK_JAN_1")},
	}
	if err := validateDeviceKeyUpload(fake); err == nil {
		t.Fatal("expected fake mock E2EE keys to be rejected")
	}
}

func TestAcceptsRealisticSignalKeyLengths(t *testing.T) {
	valid := model.UploadDeviceKeysRequest{
		DeviceID:          "device-123",
		RegistrationID:    42,
		IdentityPublicKey: bytes.Repeat([]byte{0x02, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff, 0x10, 0x20, 0x30, 0x40, 0x50, 0x60, 0x70, 0x80, 0x90, 0xa0, 0xb0, 0xc0, 0xd0, 0xe0, 0xf0, 0x01, 0x23}, 1),
		SignedPreKey:      bytes.Repeat([]byte{0x03, 0x12, 0x23, 0x34, 0x45, 0x56, 0x67, 0x78, 0x89, 0x9a, 0xab, 0xbc, 0xcd, 0xde, 0xef, 0xf0, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff, 0x00, 0x11}, 1),
		SignedPreKeySig:   bytes.Repeat([]byte{0x04}, 64),
		SignedPreKeyID:    9,
		OneTimePreKeys:    [][]byte{bytes.Repeat([]byte{0x05, 0x15, 0x25, 0x35, 0x45, 0x55, 0x65, 0x75, 0x85, 0x95, 0xa5, 0xb5, 0xc5, 0xd5, 0xe5, 0xf5, 0x06, 0x16, 0x26, 0x36, 0x46, 0x56, 0x66, 0x76, 0x86, 0x96, 0xa6, 0xb6, 0xc6, 0xd6, 0xe6, 0xf6, 0x07}, 1)},
	}
	if err := validateDeviceKeyUpload(valid); err != nil {
		t.Fatalf("expected valid signal-shaped key material to pass: %v", err)
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

func TestValidateSenderDeviceBindingAllowsRegisteredDevice(t *testing.T) {
	userID := uuid.New()
	if err := ValidateSenderDeviceBinding(userID, "device-42", []string{"device-42", "device-77"}); err != nil {
		t.Fatalf("expected device to be accepted, got %v", err)
	}
}

func TestValidateSenderDeviceBindingRejectsUnknownDevice(t *testing.T) {
	userID := uuid.New()
	if err := ValidateSenderDeviceBinding(userID, "device-999", []string{"device-42", "device-77"}); err == nil {
		t.Fatal("expected mismatch to be rejected")
	}
}
