package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/messaging-service/internal/model"
	"github.com/zerodayz7/platform/services/messaging-service/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newContactTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec(`
		CREATE TABLE contacts (
			id TEXT PRIMARY KEY,
			owner_id TEXT NOT NULL,
			contact_id TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			sync_state TEXT NOT NULL DEFAULT 'synced',
			direction TEXT NOT NULL DEFAULT 'incoming',
			change_seq INTEGER NOT NULL DEFAULT 1,
			deleted_at DATETIME,
			local_alias TEXT,
			encrypted_alias BLOB,
			version INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME,
			updated_at DATETIME
		);
	`).Error; err != nil {
		t.Fatalf("create contacts table: %v", err)
	}
	return db
}

func TestContactsServiceSendRequestCreatesPendingContact(t *testing.T) {
	db := newContactTestDB(t)

	repo := repository.NewContactsRepository(db)
	svc := NewContactsService(repo, shared.InitLogger("development", false))
	ownerID := uuid.New()
	targetID := uuid.New()

	contact, err := svc.SendRequest(context.Background(), ownerID, targetID)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	if contact == nil {
		t.Fatal("expected contact to be created")
	}
	if contact.Status != model.ContactStatusPending {
		t.Fatalf("expected pending status, got %q", contact.Status)
	}

	stored, err := repo.GetContactByOwnerAndTarget(context.Background(), ownerID, targetID)
	if err != nil {
		t.Fatalf("get stored contact: %v", err)
	}
	if stored == nil {
		t.Fatal("expected stored contact row")
	}
	if stored.Status != model.ContactStatusPending {
		t.Fatalf("expected stored contact pending, got %q", stored.Status)
	}
}

func TestContactsServiceRespondToRequestAcceptsRelationship(t *testing.T) {
	db := newContactTestDB(t)

	repo := repository.NewContactsRepository(db)
	svc := NewContactsService(repo, shared.InitLogger("development", false))
	ownerID := uuid.New()
	targetID := uuid.New()

	contact, err := svc.SendRequest(context.Background(), ownerID, targetID)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	if contact == nil {
		t.Fatal("expected contact to be created")
	}

	if err := svc.RespondToRequest(context.Background(), targetID, contact.ID, true); err != nil {
		t.Fatalf("accept request: %v", err)
	}

	accepted, err := repo.GetContactByOwnerAndTarget(context.Background(), targetID, ownerID)
	if err != nil {
		t.Fatalf("get symmetric contact: %v", err)
	}
	if accepted == nil {
		t.Fatal("expected symmetric accepted contact row")
	}
	if accepted.Status != model.ContactStatusAccepted {
		t.Fatalf("expected accepted status, got %q", accepted.Status)
	}
	if accepted.Direction != model.ContactDirectionIncoming {
		t.Fatalf("expected incoming direction, got %q", accepted.Direction)
	}
}

func TestContactsServiceRejectsDuplicateRelationInEitherDirection(t *testing.T) {
	db := newContactTestDB(t)
	repo := repository.NewContactsRepository(db)
	svc := NewContactsService(repo, shared.InitLogger("development", false))
	ownerID := uuid.New()
	targetID := uuid.New()

	if _, err := svc.SendRequest(context.Background(), ownerID, targetID); err != nil {
		t.Fatalf("first send request: %v", err)
	}

	if _, err := svc.SendRequest(context.Background(), targetID, ownerID); !errors.Is(err, ErrContactAlreadyExists) {
		t.Fatalf("expected duplicate conflict, got %v", err)
	}
}
