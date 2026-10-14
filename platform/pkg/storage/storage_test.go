package storage

import (
	"context"
	"testing"
	"time"

	viperpkg "github.com/zerodayz7/platform/pkg/viper"
)

func TestS3ConfigResolvesInternalAndExternalEndpoints(t *testing.T) {
	cfg := viperpkg.S3Config{InternalEndpoint: "minio:9000", ExternalEndpoint: "localhost:9000"}
	if got := cfg.InternalAddress(); got != "minio:9000" {
		t.Fatalf("internal endpoint mismatch: got %q", got)
	}
	if got := cfg.ExternalAddress(); got != "localhost:9000" {
		t.Fatalf("external endpoint mismatch: got %q", got)
	}

	legacy := viperpkg.S3Config{Endpoint: "minio:9000"}
	if got := legacy.InternalAddress(); got != "minio:9000" {
		t.Fatalf("legacy internal endpoint mismatch: got %q", got)
	}
	if got := legacy.ExternalAddress(); got != "minio:9000" {
		t.Fatalf("legacy external endpoint fallback mismatch: got %q", got)
	}
}

func TestNoOpStoragePresignedURLsFailClosed(t *testing.T) {
	storage := &NoOpStorage{}

	if _, err := storage.GetPresignedUploadURL(context.Background(), "demo.txt", 5*time.Minute); err == nil {
		t.Fatal("expected presigned upload URL to fail when storage is disabled")
	}
	if _, err := storage.GetPresignedURL(context.Background(), "demo.txt", 5*time.Minute); err == nil {
		t.Fatal("expected presigned download URL to fail when storage is disabled")
	}
}
