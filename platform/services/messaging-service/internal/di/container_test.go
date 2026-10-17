package di

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/zerodayz7/platform/pkg/storage"
	"github.com/zerodayz7/platform/services/messaging-service/config"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type testStorage struct{}

func (t *testStorage) Upload(ctx context.Context, filename string, content io.Reader, size int64, contentType string) (string, error) {
	return filename, nil
}
func (t *testStorage) Delete(ctx context.Context, key string) error             { return nil }
func (t *testStorage) Download(ctx context.Context, key string) ([]byte, error) { return nil, nil }
func (t *testStorage) GetPresignedURL(ctx context.Context, key string, expires time.Duration) (string, error) {
	return "https://example.test/" + key, nil
}
func (t *testStorage) GetPresignedUploadURL(ctx context.Context, key string, expires time.Duration) (string, error) {
	return "https://example.test/upload/" + key, nil
}
func (t *testStorage) HeadObject(ctx context.Context, key string) (*storage.ObjectMetadata, error) {
	return nil, nil
}

func TestNewContainer_AssignsStorage(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	storage := &testStorage{}
	container := NewContainer(db, nil, storage, nil, &config.Config{}, nil)
	if container.Storage != storage {
		t.Fatal("expected NewContainer to assign the incoming storage instance to Container.Storage")
	}
}
