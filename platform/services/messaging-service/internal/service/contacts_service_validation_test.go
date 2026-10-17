package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/messaging-service/internal/model"
	"github.com/zerodayz7/platform/services/messaging-service/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newContactsServiceTestDeps(t *testing.T) (repository.ContactsRepository, repository.MessagingRepository, *capturePublisher, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec(`
		DROP TABLE IF EXISTS contacts;
		DROP TABLE IF EXISTS messaging_activations;
		DROP TABLE IF EXISTS user_device_identities;
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
		CREATE TABLE user_device_identities (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			device_id TEXT NOT NULL,
			signal_device_id INTEGER NOT NULL DEFAULT 1,
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
		t.Fatalf("create sqlite tables for contact validation tests: %v", err)
	}

	contactsRepo := repository.NewContactsRepository(db)
	messagingRepo := repository.NewMessagingRepository(db)
	publisher := &capturePublisher{}
	return contactsRepo, messagingRepo, publisher, db
}

func seedActiveTarget(t *testing.T, db *gorm.DB, targetID uuid.UUID) {
	t.Helper()
	terms := model.CurrentMessagingTerms()
	if err := db.Create(&model.MessagingActivation{
		UserID:          targetID,
		DeviceID:        "device-1",
		Status:          model.ActivationStatusActive,
		ConsentAccepted: true,
		TermsVersion:    terms.Version,
	}).Error; err != nil {
		t.Fatalf("seed activation: %v", err)
	}
	if err := db.Create(&model.UserDeviceIdentity{
		UserID:          targetID,
		DeviceID:        "device-1",
		RegistrationID:  77,
		PublicKey:       bytesRepeat(33, 0x11),
		SignedPreKey:    bytesRepeat(33, 0x22),
		SignedPreKeySig: bytesRepeat(64, 0x33),
		SignedPreKeyID:  7,
	}).Error; err != nil {
		t.Fatalf("seed device identity: %v", err)
	}
}

func bytesRepeat(n int, b byte) []byte {
	out := make([]byte, n)
	for i := range out {
		out[i] = b
	}
	return out
}

func TestContactsServiceSendRequest_AllowsActiveEligibleTarget(t *testing.T) {
	contactsRepo, messagingRepo, publisher, db := newContactsServiceTestDeps(t)
	ownerID := uuid.New()
	targetID := uuid.New()
	seedActiveTarget(t, db, targetID)

	svc := NewContactsService(contactsRepo, messagingRepo, shared.GetLogger(), publisher)
	contact, err := svc.SendRequest(context.Background(), ownerID, targetID)
	if err != nil {
		t.Fatalf("SendRequest returned error for eligible target: %v", err)
	}
	if contact == nil {
		t.Fatal("expected contact to be created")
	}
	if publisher.event == nil {
		t.Fatal("expected contact_updated event to be published")
	}

	var count int64
	if err := db.Model(&model.Contact{}).Count(&count).Error; err != nil {
		t.Fatalf("count contacts: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly one contact row, got %d", count)
	}
}

func TestContactsServiceSendRequest_RejectsUnknownUser(t *testing.T) {
	contactsRepo, messagingRepo, publisher, db := newContactsServiceTestDeps(t)
	ownerID := uuid.New()
	targetID := uuid.New()

	svc := NewContactsService(contactsRepo, messagingRepo, shared.GetLogger(), publisher)
	if _, err := svc.SendRequest(context.Background(), ownerID, targetID); err == nil {
		t.Fatal("expected unknown target user to be rejected")
	}
	if publisher.event != nil {
		t.Fatal("expected no contact_updated event for unknown target")
	}
	var count int64
	if err := db.Model(&model.Contact{}).Count(&count).Error; err != nil {
		t.Fatalf("count contacts: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no contact rows for unknown target, got %d", count)
	}
}

func TestContactsServiceSendRequest_RejectsInactiveTarget(t *testing.T) {
	contactsRepo, messagingRepo, publisher, db := newContactsServiceTestDeps(t)
	ownerID := uuid.New()
	targetID := uuid.New()
	if err := db.Create(&model.MessagingActivation{UserID: targetID, Status: model.ActivationStatusPending}).Error; err != nil {
		t.Fatalf("seed pending activation: %v", err)
	}

	svc := NewContactsService(contactsRepo, messagingRepo, shared.GetLogger(), publisher)
	if _, err := svc.SendRequest(context.Background(), ownerID, targetID); err == nil {
		t.Fatal("expected inactive target to be rejected")
	}
	if publisher.event != nil {
		t.Fatal("expected no contact_updated event for inactive target")
	}

	var count int64
	if err := db.Model(&model.Contact{}).Count(&count).Error; err != nil {
		t.Fatalf("count contacts: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no contact rows for inactive target, got %d", count)
	}
}

func TestContactsServiceSendRequest_RejectsMissingDeviceIdentity(t *testing.T) {
	contactsRepo, messagingRepo, publisher, db := newContactsServiceTestDeps(t)
	ownerID := uuid.New()
	targetID := uuid.New()
	terms := model.CurrentMessagingTerms()
	if err := db.Create(&model.MessagingActivation{
		UserID:          targetID,
		Status:          model.ActivationStatusActive,
		ConsentAccepted: true,
		TermsVersion:    terms.Version,
	}).Error; err != nil {
		t.Fatalf("seed active activation: %v", err)
	}

	svc := NewContactsService(contactsRepo, messagingRepo, shared.GetLogger(), publisher)
	if _, err := svc.SendRequest(context.Background(), ownerID, targetID); err == nil {
		t.Fatal("expected target without device identity to be rejected")
	}
	if publisher.event != nil {
		t.Fatal("expected no contact_updated event without device identity")
	}
	var count int64
	if err := db.Model(&model.Contact{}).Count(&count).Error; err != nil {
		t.Fatalf("count contacts: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no contact rows without device identity, got %d", count)
	}
}

func TestContactsServiceSendRequest_RejectsSelfInvite(t *testing.T) {
	contactsRepo, messagingRepo, publisher, _ := newContactsServiceTestDeps(t)
	userID := uuid.New()

	svc := NewContactsService(contactsRepo, messagingRepo, shared.GetLogger(), publisher)
	if _, err := svc.SendRequest(context.Background(), userID, userID); err == nil {
		t.Fatal("expected self invite to be rejected")
	}
	if publisher.event != nil {
		t.Fatal("expected no event for self invite")
	}
}

func TestContactsServiceSendRequest_RejectsExistingInvitationInEitherDirection(t *testing.T) {
	contactsRepo, messagingRepo, publisher, db := newContactsServiceTestDeps(t)
	ownerID := uuid.New()
	targetID := uuid.New()
	seedActiveTarget(t, db, targetID)

	if err := db.Create(&model.Contact{
		OwnerID:   targetID,
		ContactID: ownerID,
		Status:    model.ContactStatusPending,
		Direction: model.ContactDirectionIncoming,
		Version:   1,
	}).Error; err != nil {
		t.Fatalf("seed reverse invite: %v", err)
	}

	svc := NewContactsService(contactsRepo, messagingRepo, shared.GetLogger(), publisher)
	if _, err := svc.SendRequest(context.Background(), ownerID, targetID); err == nil {
		t.Fatal("expected duplicate contact request in reverse direction to be rejected")
	}
	if publisher.event != nil {
		t.Fatal("expected no additional event for duplicate request")
	}

	var count int64
	if err := db.Model(&model.Contact{}).Count(&count).Error; err != nil {
		t.Fatalf("count contacts: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one row to remain after duplicate rejection, got %d", count)
	}
}
