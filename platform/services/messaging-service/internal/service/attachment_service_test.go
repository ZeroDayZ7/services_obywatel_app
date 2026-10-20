package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/pkg/storage"
	"github.com/zerodayz7/platform/services/messaging-service/internal/model"
	"github.com/zerodayz7/platform/services/messaging-service/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type fakeStorage struct {
	data map[string][]byte
}

func (f *fakeStorage) Upload(ctx context.Context, filename string, content io.Reader, size int64, contentType string) (string, error) {
	payload, err := io.ReadAll(content)
	if err != nil {
		return "", err
	}
	f.data[filename] = payload
	return filename, nil
}

func (f *fakeStorage) Delete(ctx context.Context, key string) error {
	delete(f.data, key)
	return nil
}

func (f *fakeStorage) Download(ctx context.Context, key string) ([]byte, error) {
	payload, ok := f.data[key]
	if !ok {
		return nil, errors.New("not found")
	}
	return payload, nil
}

func (f *fakeStorage) GetPresignedURL(ctx context.Context, key string, expires time.Duration) (string, error) {
	return "https://example.test/" + key, nil
}

func (f *fakeStorage) GetPresignedUploadURL(ctx context.Context, key string, expires time.Duration) (string, error) {
	return "https://example.test/upload/" + key, nil
}

func (f *fakeStorage) HeadObject(ctx context.Context, key string) (*storage.ObjectMetadata, error) {
	payload, ok := f.data[key]
	if !ok {
		return nil, errors.New("not found")
	}
	return &storage.ObjectMetadata{ContentType: "application/octet-stream", Size: int64(len(payload))}, nil
}

func TestAttachmentServiceFinalizeUpload_ValidatesSizeAndHash(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec(`
		DROP TABLE IF EXISTS attachments;
		CREATE TABLE attachments (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			conversation_id TEXT,
			message_id TEXT,
			file_key TEXT NOT NULL,
			original_name TEXT NOT NULL,
			content_type TEXT NOT NULL,
			size INTEGER NOT NULL DEFAULT 0,
			sha256 TEXT,
			status TEXT NOT NULL DEFAULT 'pending',
			metadata TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		);
	`).Error; err != nil {
		t.Fatalf("migrate attachment schema: %v", err)
	}
	store := &fakeStorage{data: make(map[string][]byte)}
	repo := repository.NewMessagingRepository(db)
	service := NewAttachmentService(repo, store, nil, 25*1024*1024)
	userID := uuid.New()
	conversationID := uuid.New()

	prepareResp, err := service.PrepareUpload(context.Background(), userID, model.AttachmentUploadRequest{
		ConversationID: &conversationID,
		FileName:       "hello.txt",
		ContentType:    "text/plain",
		Size:           int64(len("hello")),
	})
	if err != nil {
		t.Fatalf("prepare upload: %v", err)
	}
	payload := []byte("hello")
	if _, err := store.Upload(context.Background(), prepareResp.Attachment.FileKey, io.NopCloser(bytes.NewReader(payload)), int64(len(payload)), "text/plain"); err != nil {
		t.Fatalf("upload fake file: %v", err)
	}

	hash := sha256.Sum256(payload)
	finalized, err := service.FinalizeUpload(context.Background(), userID, model.AttachmentFinalizeRequest{
		AttachmentID: prepareResp.Attachment.ID,
		Size:         int64(len(payload)),
		Sha256:       hex.EncodeToString(hash[:]),
	})
	if err != nil {
		t.Fatalf("finalize upload: %v", err)
	}
	if finalized.Status != model.AttachmentStatusUploaded {
		t.Fatalf("expected uploaded status, got %s", finalized.Status)
	}

	_, err = service.FinalizeUpload(context.Background(), userID, model.AttachmentFinalizeRequest{
		AttachmentID: prepareResp.Attachment.ID,
		Size:         int64(len(payload)) + 1,
		Sha256:       hex.EncodeToString(hash[:]),
	})
	if err == nil {
		t.Fatal("expected size-mismatch validation to fail")
	}
}

func TestAttachmentServicePrepareUpload_UsesMessengerObjectKeyPrefix(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec(`
		DROP TABLE IF EXISTS attachments;
		CREATE TABLE attachments (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			conversation_id TEXT,
			message_id TEXT,
			file_key TEXT NOT NULL,
			original_name TEXT NOT NULL,
			content_type TEXT NOT NULL,
			size INTEGER NOT NULL DEFAULT 0,
			sha256 TEXT,
			status TEXT NOT NULL DEFAULT 'pending',
			metadata TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		);
	`).Error; err != nil {
		t.Fatalf("migrate attachment schema: %v", err)
	}

	store := &fakeStorage{data: make(map[string][]byte)}
	repo := repository.NewMessagingRepository(db)
	service := NewAttachmentService(repo, store, nil, 25*1024*1024)
	userID := uuid.New()
	conversationID := uuid.New()

	prepareResp, err := service.PrepareUpload(context.Background(), userID, model.AttachmentUploadRequest{
		ConversationID: &conversationID,
		FileName:       "invoice.pdf",
		ContentType:    "application/pdf",
		Size:           128,
	})
	if err != nil {
		t.Fatalf("prepare upload: %v", err)
	}

	key := prepareResp.Attachment.FileKey
	if !strings.Contains(key, "messenger/attachments/") {
		t.Fatalf("expected messenger attachment prefix, got %q", key)
	}
	if !strings.Contains(key, conversationID.String()) {
		t.Fatalf("expected conversation id in key, got %q", key)
	}
	if !strings.Contains(key, prepareResp.Attachment.ID.String()) {
		t.Fatalf("expected attachment id in key, got %q", key)
	}
	if !strings.HasSuffix(key, ".pdf") {
		t.Fatalf("expected original extension retained in the storage key, got %q", key)
	}
	if strings.Contains(key, "invoice") {
		t.Fatalf("expected original file name not to be used as storage identifier, got %q", key)
	}
}

func setupAttachmentPermissionTestDB(t *testing.T) (*gorm.DB, repository.MessagingRepository, *fakeStorage) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec(`
		DROP TABLE IF EXISTS attachments;
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
			role TEXT DEFAULT 'member',
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
			attachment_id TEXT,
			type TEXT NOT NULL DEFAULT 'text',
			sequence INTEGER NOT NULL DEFAULT 0,
			encrypted_payload BLOB NOT NULL,
			media_header BLOB,
			version INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		);
		CREATE TABLE attachments (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			conversation_id TEXT,
			message_id TEXT,
			file_key TEXT NOT NULL,
			original_name TEXT NOT NULL,
			content_type TEXT NOT NULL,
			size INTEGER NOT NULL DEFAULT 0,
			sha256 TEXT,
			status TEXT NOT NULL DEFAULT 'pending',
			metadata TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		);
	`).Error; err != nil {
		t.Fatalf("create sqlite schema: %v", err)
	}
	store := &fakeStorage{data: make(map[string][]byte)}
	return db, repository.NewMessagingRepository(db), store
}

func TestAttachmentServiceGetDownloadURL_AuthorizesConversationMembers(t *testing.T) {
	db, repo, store := setupAttachmentPermissionTestDB(t)
	service := NewAttachmentService(repo, store, nil, 25*1024*1024)

	ownerID := uuid.New()
	memberID := uuid.New()
	outsiderID := uuid.New()
	conversationID := uuid.New()
	messageID := uuid.New()
	attachmentID := uuid.New()

	if err := db.Create(&model.Conversation{
		ID:   conversationID,
		Type: model.ConversationTypeDirect,
	}).Error; err != nil {
		t.Fatalf("create conversation: %v", err)
	}
	for _, userID := range []uuid.UUID{ownerID, memberID} {
		if err := db.Create(&model.ConversationMember{
			ID:             uuid.New(),
			ConversationID: conversationID,
			UserID:         userID,
			Role:           "member",
		}).Error; err != nil {
			t.Fatalf("create conversation member %s: %v", userID, err)
		}
	}
	if err := db.Create(&model.Message{
		ID:               messageID,
		ConversationID:   conversationID,
		SenderID:         ownerID,
		SenderDeviceID:   "device-1",
		Type:             model.MessageTypeMedia,
		Sequence:         1,
		EncryptedPayload: []byte("payload"),
	}).Error; err != nil {
		t.Fatalf("create message: %v", err)
	}
	if err := repo.CreateAttachment(context.Background(), &model.Attachment{
		ID:             attachmentID,
		UserID:         ownerID,
		ConversationID: &conversationID,
		MessageID:      &messageID,
		FileKey:        "messenger/attachments/test/attachment/file.pdf",
		OriginalName:   "file.pdf",
		ContentType:    "application/pdf",
		Size:           128,
		Status:         model.AttachmentStatusUploaded,
	}); err != nil {
		t.Fatalf("create attachment: %v", err)
	}

	t.Run("owner can download", func(t *testing.T) {
		download, err := service.GetAttachmentDownloadURL(context.Background(), ownerID, attachmentID)
		if err != nil {
			t.Fatalf("owner should be authorized: %v", err)
		}
		if download == nil || download.AttachmentID != attachmentID {
			t.Fatalf("expected attachment download response for owner, got %#v", download)
		}
	})

	t.Run("conversation member can download", func(t *testing.T) {
		download, err := service.GetAttachmentDownloadURL(context.Background(), memberID, attachmentID)
		if err != nil {
			t.Fatalf("conversation member should be authorized: %v", err)
		}
		if download == nil || download.AttachmentID != attachmentID {
			t.Fatalf("expected attachment download response for member, got %#v", download)
		}
	})

	t.Run("outsider is forbidden", func(t *testing.T) {
		download, err := service.GetAttachmentDownloadURL(context.Background(), outsiderID, attachmentID)
		if err == nil {
			t.Fatal("expected outsider to be denied")
		}
		if download != nil {
			t.Fatalf("expected nil response for outsider, got %#v", download)
		}
	})

	t.Run("user without membership is forbidden", func(t *testing.T) {
		otherUserID := uuid.New()
		download, err := service.GetAttachmentDownloadURL(context.Background(), otherUserID, attachmentID)
		if err == nil {
			t.Fatal("expected non-member to be denied")
		}
		if download != nil {
			t.Fatalf("expected nil response for non-member, got %#v", download)
		}
	})
}

func TestAttachmentServiceGetAttachment_RejectsMissingAttachmentForUnauthorizedUser(t *testing.T) {
	_, repo, store := setupAttachmentPermissionTestDB(t)
	service := NewAttachmentService(repo, store, nil, 25*1024*1024)

	userID := uuid.New()
	missingID := uuid.New()
	attachment, err := service.GetAttachment(context.Background(), userID, missingID)
	if err == nil {
		t.Fatal("expected not found for missing attachment")
	}
	if attachment != nil {
		t.Fatalf("expected nil attachment for missing resource, got %#v", attachment)
	}
}
