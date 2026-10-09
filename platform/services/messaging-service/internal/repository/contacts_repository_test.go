package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/services/messaging-service/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestContactsRepositoryCreateAndLookup(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
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

	repo := NewContactsRepository(db)
	ownerID := uuid.New()
	targetID := uuid.New()
	input := &model.Contact{
		OwnerID:   ownerID,
		ContactID: targetID,
		Status:    model.ContactStatusPending,
		Direction: model.ContactDirectionOutgoing,
		Version:   1,
	}

	if err := repo.CreateContact(context.Background(), input); err != nil {
		t.Fatalf("create contact: %v", err)
	}

	stored, err := repo.GetContactByOwnerAndTarget(context.Background(), ownerID, targetID)
	if err != nil {
		t.Fatalf("get by owner and target: %v", err)
	}
	if stored == nil {
		t.Fatal("expected contact was not persisted")
	}
	if stored.OwnerID != ownerID || stored.ContactID != targetID {
		t.Fatalf("persisted contact mismatch: got %s -> %s", stored.OwnerID, stored.ContactID)
	}
	if stored.Direction != model.ContactDirectionOutgoing {
		t.Fatalf("expected outgoing direction, got %q", stored.Direction)
	}

	forUser, err := repo.GetContactsByUserID(context.Background(), targetID)
	if err != nil {
		t.Fatalf("get by user id: %v", err)
	}
	if len(forUser) != 1 {
		t.Fatalf("expected target user to see exactly one incoming contact, got %d", len(forUser))
	}
	if forUser[0].OwnerID != ownerID || forUser[0].ContactID != targetID {
		t.Fatalf("expected incoming relation owner=%s contact=%s, got owner=%s contact=%s", ownerID, targetID, forUser[0].OwnerID, forUser[0].ContactID)
	}

	ownerVisible, err := repo.GetContactsByUserID(context.Background(), ownerID)
	if err != nil {
		t.Fatalf("get owner contacts: %v", err)
	}
	if len(ownerVisible) != 1 {
		t.Fatalf("expected owner to see exactly one contact, got %d", len(ownerVisible))
	}
}
