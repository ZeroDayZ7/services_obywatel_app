package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/services/messaging-service/internal/model"
	"github.com/zerodayz7/platform/services/messaging-service/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestMessagingActivationLifecycle(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	if err := db.Exec(`
		CREATE TABLE messaging_activations (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			device_id TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'not_started',
			consent_accepted INTEGER NOT NULL DEFAULT 0,
			terms_version TEXT NOT NULL DEFAULT '',
			activated_at DATETIME,
			last_seen_at DATETIME,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		);
	`).Error; err != nil {
		t.Fatalf("migrate activation table: %v", err)
	}

	repo := repository.NewMessagingRepository(db)
	service := NewMessagingService(repo, nil, nil)
	userID := uuid.New()
	ctx := context.Background()

	status, err := service.GetActivationStatus(ctx, userID)
	if err != nil {
		t.Fatalf("get activation status: %v", err)
	}
	if status.Status != model.ActivationStatusNotStarted {
		t.Fatalf("expected initial status %q, got %q", model.ActivationStatusNotStarted, status.Status)
	}

	_, err = service.ActivateMessaging(ctx, userID, model.ActivateMessagingRequest{
		DeviceID: "device-1",
	})
	if err == nil {
		t.Fatal("expected activation without explicit consent to fail")
	}

	accepted, err := service.AcceptTerms(ctx, userID, model.AcceptTermsRequest{DeviceID: "device-1", TermsVersion: model.CurrentMessagingTerms().Version})
	if err != nil {
		t.Fatalf("accept terms: %v", err)
	}
	if accepted.Status != model.ActivationStatusActive {
		t.Fatalf("expected active status after accepted terms, got %q", accepted.Status)
	}
	if !accepted.ConsentAccepted {
		t.Fatal("expected consent accepted flag to be true")
	}
	if accepted.TermsVersion != model.CurrentMessagingTerms().Version {
		t.Fatalf("expected terms version %q, got %q", model.CurrentMessagingTerms().Version, accepted.TermsVersion)
	}
	if accepted.ActivatedAt == nil || accepted.ActivatedAt.IsZero() {
		t.Fatal("expected activated timestamp to be set")
	}

	activated, err := service.ActivateMessaging(ctx, userID, model.ActivateMessagingRequest{
		DeviceID:     "device-2",
		TermsVersion: model.CurrentMessagingTerms().Version,
		Consent:      true,
	})
	if err != nil {
		t.Fatalf("activate messaging after explicit consent: %v", err)
	}
	if activated.Status != model.ActivationStatusActive {
		t.Fatalf("expected active status after activation, got %q", activated.Status)
	}
	if activated.ConsentAccepted != true {
		t.Fatal("expected consent accepted flag to be true")
	}
	if time.Since(activated.UpdatedAt) > time.Minute {
		t.Fatal("updated_at should be recent")
	}
}
