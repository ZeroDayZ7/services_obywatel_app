package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/messaging-service/internal/model"
	pub "github.com/zerodayz7/platform/services/messaging-service/internal/publisher"
)

type capturePublisher struct {
	event pub.Event
}

func (p *capturePublisher) Publish(_ context.Context, event pub.Event) error {
	p.event = event
	return nil
}

type stubContactsRepo struct {
	created *model.Contact
}

type stubTargetValidator struct{}

func (s *stubTargetValidator) GetActivationStatus(ctx context.Context, userID uuid.UUID) (*model.MessagingActivation, error) {
	return &model.MessagingActivation{
		UserID:          userID,
		Status:          model.ActivationStatusActive,
		ConsentAccepted: true,
		TermsVersion:    model.CurrentMessagingTerms().Version,
	}, nil
}

func (s *stubTargetValidator) GetLatestDeviceIdentityForUser(ctx context.Context, userID uuid.UUID) (*model.UserDeviceIdentity, error) {
	return &model.UserDeviceIdentity{
		UserID:    userID,
		DeviceID:  "device-1",
		PublicKey: []byte{0x01, 0x02, 0x03},
	}, nil
}

func (r *stubContactsRepo) GetContactsByUserID(ctx context.Context, userID uuid.UUID) ([]model.Contact, error) {
	return nil, nil
}

func (r *stubContactsRepo) GetContactByOwnerAndTarget(ctx context.Context, ownerID, targetID uuid.UUID) (*model.Contact, error) {
	return nil, nil
}

func (r *stubContactsRepo) GetContactByID(ctx context.Context, id uuid.UUID) (*model.Contact, error) {
	return nil, nil
}

func (r *stubContactsRepo) CreateContact(ctx context.Context, contact *model.Contact) error {
	r.created = contact
	return nil
}

func (r *stubContactsRepo) UpdateContactStatus(ctx context.Context, contactID uuid.UUID, status model.ContactStatus) error {
	return nil
}

func (r *stubContactsRepo) CreateSymmetricContact(ctx context.Context, ownerID, targetID uuid.UUID, status model.ContactStatus) error {
	return nil
}

func TestContactsServiceUsesInjectedPublisherForContactEvents(t *testing.T) {
	ownerID := uuid.New()
	targetID := uuid.New()
	pubs := &capturePublisher{}
	svc := NewContactsService(&stubContactsRepo{}, &stubTargetValidator{}, shared.GetLogger(), pubs)

	_, err := svc.SendRequest(context.Background(), ownerID, targetID)
	if err != nil {
		t.Fatalf("SendRequest returned error: %v", err)
	}
	if pubs.event == nil {
		t.Fatal("expected publisher to receive an event")
	}

	payload := map[string]any(pubs.event)
	if payload["type"] != "contact_updated" {
		t.Fatalf("expected event type contact_updated, got %v", payload["type"])
	}
	if payload["source_service"] != "messaging-service" {
		t.Fatalf("expected source_service messaging-service, got %v", payload["source_service"])
	}

	targetUsers, ok := payload["target_user_ids"].([]string)
	if !ok || len(targetUsers) != 2 {
		t.Fatalf("expected 2 target users, got %#v", payload["target_user_ids"])
	}
	if targetUsers[0] != ownerID.String() && targetUsers[1] != ownerID.String() {
		t.Fatalf("expected owner user to be included in fan-out, got %#v", targetUsers)
	}
}
