package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/services/messaging-service/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCreateMessageAdvancesGlobalVersionAndDeltaCursor(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	if err := db.Exec(`
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
		t.Fatalf("migrate schema: %v", err)
	}

	repo := NewMessagingRepository(db)
	ctx := context.Background()
	userID := uuid.New()
	conversationID := uuid.New()

	if err := repo.CreateConversation(ctx, &model.Conversation{
		ID:    conversationID,
		Type:  model.ConversationTypeDirect,
		Title: "delta-test",
		Members: []model.ConversationMember{{
			ID:             uuid.New(),
			ConversationID: conversationID,
			UserID:         userID,
			Role:           "member",
		}},
	}); err != nil {
		t.Fatalf("create conversation: %v", err)
	}

	first := &model.Message{
		ID:               uuid.New(),
		ConversationID:   conversationID,
		SenderID:         userID,
		SenderDeviceID:   "device-1",
		Type:             model.MessageTypeText,
		EncryptedPayload: []byte("first-ciphertext"),
	}
	if err := repo.CreateMessage(ctx, first); err != nil {
		t.Fatalf("create first message: %v", err)
	}

	second := &model.Message{
		ID:               uuid.New(),
		ConversationID:   conversationID,
		SenderID:         userID,
		SenderDeviceID:   "device-1",
		Type:             model.MessageTypeText,
		EncryptedPayload: []byte("second-ciphertext"),
	}
	if err := repo.CreateMessage(ctx, second); err != nil {
		t.Fatalf("create second message: %v", err)
	}

	if first.Version == 0 || second.Version == 0 {
		t.Fatalf("message versions should be non-zero, got first=%d second=%d", first.Version, second.Version)
	}
	if second.Version <= first.Version {
		t.Fatalf("second message version should be greater than first: first=%d second=%d", first.Version, second.Version)
	}

	messages, err := repo.GetMessagesSinceVersion(ctx, userID, first.Version)
	if err != nil {
		t.Fatalf("fetch messages since version: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("expected exactly one newer message after version %d, got %d", first.Version, len(messages))
	}
	if messages[0].ID != second.ID {
		t.Fatalf("expected second message to be returned as delta, got %s", messages[0].ID)
	}
}
