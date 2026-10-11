package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/services/messaging-service/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newMessagingTestRepo(t *testing.T) (*messagingRepository, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec(`DROP INDEX IF EXISTS idx_user_device_identity_active; DROP TABLE IF EXISTS user_device_identities;`).Error; err != nil {
		t.Fatalf("reset user_device_identities table: %v", err)
	}
	if err := db.Exec(`
		CREATE TABLE user_device_identities (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			device_id TEXT NOT NULL,
			registration_id INTEGER NOT NULL DEFAULT 0,
			public_key BLOB NOT NULL,
			signed_pre_key BLOB NOT NULL,
			signed_pre_key_sig BLOB NOT NULL,
			signed_pre_key_id INTEGER NOT NULL,
			one_time_pre_keys_count INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		);
	`).Error; err != nil {
		t.Fatalf("create user_device_identities table: %v", err)
	}
	if err := db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_user_device_identity_active
		ON user_device_identities (user_id, device_id)
		WHERE deleted_at IS NULL;
	`).Error; err != nil {
		t.Fatalf("create active unique index: %v", err)
	}

	return &messagingRepository{db: db}, db
}

func TestMessagingRepositoryCreateMessage_RejectsDuplicateIdempotencyKey(t *testing.T) {
	repo, db := newMessagingTestRepo(t)
	ctx := context.Background()
	if err := db.Exec(`
		DROP TABLE IF EXISTS messages;
		DROP TABLE IF EXISTS conversation_members;
		DROP TABLE IF EXISTS conversations;
		CREATE TABLE conversations (
			id TEXT PRIMARY KEY,
			type TEXT NOT NULL DEFAULT 'direct',
			title TEXT,
			last_sequence INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		);
		CREATE TABLE conversation_members (
			id TEXT PRIMARY KEY,
			conversation_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'member',
			last_read_sequence INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		);
		CREATE TABLE messages (
			id TEXT PRIMARY KEY,
			idempotency_key TEXT NOT NULL DEFAULT '',
			conversation_id TEXT NOT NULL,
			sender_id TEXT NOT NULL,
			sender_device_id TEXT NOT NULL,
			type TEXT NOT NULL DEFAULT 'text',
			sequence INTEGER NOT NULL,
			encrypted_payload BLOB NOT NULL,
			media_header BLOB,
			version INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		);
	`).Error; err != nil {
		t.Fatalf("migrate message schema: %v", err)
	}

	conversationID := uuid.New()
	if err := db.Create(&model.Conversation{ID: conversationID, Type: model.ConversationTypeDirect, Title: "demo"}).Error; err != nil {
		t.Fatalf("create conversation: %v", err)
	}

	msgA := &model.Message{
		ID:               uuid.New(),
		ConversationID:   conversationID,
		SenderID:         uuid.New(),
		SenderDeviceID:   "device-1",
		Type:             model.MessageTypeText,
		EncryptedPayload: []byte("hello"),
		IdempotencyKey:   "same-key",
	}
	msgB := &model.Message{
		ID:               uuid.New(),
		ConversationID:   conversationID,
		SenderID:         uuid.New(),
		SenderDeviceID:   "device-1",
		Type:             model.MessageTypeText,
		EncryptedPayload: []byte("hello-again"),
		IdempotencyKey:   "same-key",
	}

	if err := repo.CreateMessage(ctx, msgA); err != nil {
		t.Fatalf("create first message: %v", err)
	}
	if err := repo.CreateMessage(ctx, msgB); err == nil {
		t.Fatal("expected duplicate idempotency key to be rejected")
	}

	var total int64
	if err := db.Model(&model.Message{}).Count(&total).Error; err != nil {
		t.Fatalf("count messages: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected exactly one persisted message for same idempotency key, got %d", total)
	}
}

func TestMessagingRepositoryFindDirectConversationByUsers_ReturnsExistingConversation(t *testing.T) {
	repo, db := newMessagingTestRepo(t)
	ctx := context.Background()
	userA := uuid.New()
	userB := uuid.New()
	conversationID := uuid.New()

	if err := db.Exec(`
		DROP TABLE IF EXISTS messages;
		DROP TABLE IF EXISTS conversation_members;
		DROP TABLE IF EXISTS conversations;
		CREATE TABLE conversations (
			id TEXT PRIMARY KEY,
			type TEXT NOT NULL DEFAULT 'direct',
			title TEXT,
			last_sequence INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		);
		CREATE TABLE conversation_members (
			id TEXT PRIMARY KEY,
			conversation_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'member',
			last_read_sequence INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		);
	`).Error; err != nil {
		t.Fatalf("create tables: %v", err)
	}

	if err := db.Create(&model.Conversation{
		ID:    conversationID,
		Type:  model.ConversationTypeDirect,
		Title: "direct",
	}).Error; err != nil {
		t.Fatalf("create conversation: %v", err)
	}

	if err := db.Create(&model.ConversationMember{
		ID:             uuid.New(),
		ConversationID: conversationID,
		UserID:         userA,
		Role:           "admin",
	}).Error; err != nil {
		t.Fatalf("create member A: %v", err)
	}
	if err := db.Create(&model.ConversationMember{
		ID:             uuid.New(),
		ConversationID: conversationID,
		UserID:         userB,
		Role:           "member",
	}).Error; err != nil {
		t.Fatalf("create member B: %v", err)
	}

	found, err := repo.FindDirectConversationByUsers(ctx, userA, userB)
	if err != nil {
		t.Fatalf("find direct conversation: %v", err)
	}
	if found == nil {
		t.Fatal("expected existing direct conversation to be found")
	}
	if found.ID != conversationID {
		t.Fatalf("expected conversation %s, got %s", conversationID, found.ID)
	}
}

func TestMessagingRepositorySaveDeviceIdentity_UpsertsSameDevice(t *testing.T) {
	repo, db := newMessagingTestRepo(t)
	ctx := context.Background()
	userID := uuid.New()
	deviceID := "device-1"

	first := &model.UserDeviceIdentity{
		ID:              uuid.New(),
		UserID:          userID,
		DeviceID:        deviceID,
		RegistrationID:  99,
		PublicKey:       []byte("public-1"),
		SignedPreKey:    []byte("spk-1"),
		SignedPreKeySig: []byte("sig-1"),
		SignedPreKeyID:  7,
	}
	if err := repo.SaveDeviceIdentity(ctx, first); err != nil {
		t.Fatalf("save first identity: %v", err)
	}

	second := &model.UserDeviceIdentity{
		ID:              uuid.New(),
		UserID:          userID,
		DeviceID:        deviceID,
		RegistrationID:  88,
		PublicKey:       []byte("public-2"),
		SignedPreKey:    []byte("spk-2"),
		SignedPreKeySig: []byte("sig-2"),
		SignedPreKeyID:  9,
	}
	if err := repo.SaveDeviceIdentity(ctx, second); err != nil {
		t.Fatalf("save second identity for same device: %v", err)
	}

	var count int64
	if err := db.Model(&model.UserDeviceIdentity{}).Count(&count).Error; err != nil {
		t.Fatalf("count identities: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly one active identity for same user/device, got %d", count)
	}

	var stored model.UserDeviceIdentity
	if err := db.Unscoped().Where("user_id = ? AND device_id = ?", userID, deviceID).First(&stored).Error; err != nil {
		t.Fatalf("load stored identity: %v", err)
	}
	if stored.RegistrationID != second.RegistrationID {
		t.Fatalf("expected registration id %d, got %d", second.RegistrationID, stored.RegistrationID)
	}
	if stored.DeletedAt.Valid {
		t.Fatalf("expected active identity to not be soft deleted")
	}
}

func TestMessagingRepositorySaveDeviceIdentity_AllowsMultipleDevicesPerUser(t *testing.T) {
	repo, db := newMessagingTestRepo(t)
	ctx := context.Background()
	userID := uuid.New()

	for i := 0; i < 2; i++ {
		identity := &model.UserDeviceIdentity{
			ID:              uuid.New(),
			UserID:          userID,
			DeviceID:        "device-" + string(rune('a'+i)),
			RegistrationID:  uint32(i + 1),
			PublicKey:       []byte("public-" + string(rune('a'+i))),
			SignedPreKey:    []byte("spk-" + string(rune('a'+i))),
			SignedPreKeySig: []byte("sig-" + string(rune('a'+i))),
			SignedPreKeyID:  uint32(i + 10),
		}
		if err := repo.SaveDeviceIdentity(ctx, identity); err != nil {
			t.Fatalf("save identity for device %d: %v", i, err)
		}
	}

	var count int64
	if err := db.Model(&model.UserDeviceIdentity{}).Count(&count).Error; err != nil {
		t.Fatalf("count identities: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected two active identities for same user, got %d", count)
	}
}

func TestMessagingRepositorySaveDeviceIdentity_RestoresSoftDeletedDevice(t *testing.T) {
	repo, db := newMessagingTestRepo(t)
	ctx := context.Background()
	userID := uuid.New()
	deviceID := "device-soft-delete"

	identity := &model.UserDeviceIdentity{
		ID:              uuid.New(),
		UserID:          userID,
		DeviceID:        deviceID,
		RegistrationID:  42,
		PublicKey:       []byte("public-old"),
		SignedPreKey:    []byte("spk-old"),
		SignedPreKeySig: []byte("sig-old"),
		SignedPreKeyID:  11,
	}
	if err := repo.SaveDeviceIdentity(ctx, identity); err != nil {
		t.Fatalf("save original identity: %v", err)
	}

	var stored model.UserDeviceIdentity
	if err := db.Unscoped().Where("user_id = ? AND device_id = ?", userID, deviceID).First(&stored).Error; err != nil {
		t.Fatalf("load identity before delete: %v", err)
	}
	if err := db.Delete(&stored).Error; err != nil {
		t.Fatalf("soft delete identity: %v", err)
	}

	reregistered := &model.UserDeviceIdentity{
		ID:              uuid.New(),
		UserID:          userID,
		DeviceID:        deviceID,
		RegistrationID:  77,
		PublicKey:       []byte("public-new"),
		SignedPreKey:    []byte("spk-new"),
		SignedPreKeySig: []byte("sig-new"),
		SignedPreKeyID:  22,
	}
	if err := repo.SaveDeviceIdentity(ctx, reregistered); err != nil {
		t.Fatalf("re-register same device after soft delete: %v", err)
	}

	var active model.UserDeviceIdentity
	if err := db.Unscoped().Where("user_id = ? AND device_id = ?", userID, deviceID).First(&active).Error; err != nil {
		t.Fatalf("load identity after re-registration: %v", err)
	}
	if active.DeletedAt.Valid {
		t.Fatalf("expected soft-deleted record to be restored as active")
	}
	if active.RegistrationID != reregistered.RegistrationID {
		t.Fatalf("expected re-registered identity to update registration id to %d, got %d", reregistered.RegistrationID, active.RegistrationID)
	}
}
