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
		FileName:      "invoice.pdf",
		ContentType:   "application/pdf",
		Size:          128,
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

func TestAttachmentServiceGetDownloadURL_RequiresOwnership(t *testing.T) {
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
	ownerID := uuid.New()
	otherUserID := uuid.New()
	attachmentID := uuid.New()

	if err := repo.CreateAttachment(context.Background(), &model.Attachment{
		ID:           attachmentID,
		UserID:       ownerID,
		FileKey:      "messenger/attachments/test/attachment/file.pdf",
		OriginalName: "test-file.txt",
		ContentType:  "text/plain",
		Size:         12,
		Status:       model.AttachmentStatusUploaded,
	}); err != nil {
		t.Fatalf("create attachment: %v", err)
	}

	download, err := service.GetAttachmentDownloadURL(context.Background(), otherUserID, attachmentID)
	if err == nil {
		t.Fatal("expected ownership check to fail")
	}
	if download != nil {
		t.Fatal("expected nil download response on unauthorized access")
	}
}
