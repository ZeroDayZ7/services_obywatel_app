package storage

import (
	"context"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	viperpkg "github.com/zerodayz7/platform/pkg/viper"
)

func TestS3ConfigRequiresExplicitEndpointsWhenEnabled(t *testing.T) {
	validate := validator.New()

	t.Run("valid endpoints pass", func(t *testing.T) {
		cfg := viperpkg.S3Config{
			Enabled:          true,
			InternalEndpoint: "minio:9000",
			ExternalEndpoint: "localhost:9000",
			AccessKey:        "minioadmin",
			SecretKey:        "minioadminpassword",
			Bucket:           "citizens-data",
		}
		if err := validate.Struct(cfg); err != nil {
			t.Fatalf("expected valid S3 config to pass validation: %v", err)
		}
	})

	t.Run("missing external endpoint fails", func(t *testing.T) {
		cfg := viperpkg.S3Config{
			Enabled:          true,
			InternalEndpoint: "minio:9000",
			AccessKey:        "minioadmin",
			SecretKey:        "minioadminpassword",
			Bucket:           "citizens-data",
		}
		if err := validate.Struct(cfg); err == nil {
			t.Fatal("expected validation to fail when external endpoint is missing")
		}
	})
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
