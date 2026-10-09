package handler

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	reqctx "github.com/zerodayz7/platform/pkg/context"
	"github.com/zerodayz7/platform/services/messaging-service/internal/model"
)

type stubContactsService struct {
	created *model.Contact
	err     error
}

func (s *stubContactsService) GetContacts(ctx context.Context, userID uuid.UUID) ([]model.Contact, error) {
	return nil, nil
}

func (s *stubContactsService) SendRequest(ctx context.Context, ownerID, targetID uuid.UUID) (*model.Contact, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.created == nil {
		s.created = &model.Contact{OwnerID: ownerID, ContactID: targetID, Status: model.ContactStatusPending}
	}
	return s.created, nil
}

func (s *stubContactsService) RespondToRequest(ctx context.Context, currentUserID, contactID uuid.UUID, accept bool) error {
	return nil
}

func TestRequestContactUsesAuthenticatedUserID(t *testing.T) {
	ownerID := uuid.New()
	targetID := uuid.New()
	service := &stubContactsService{}
	handler := NewContactsHandler(service)

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals(reqctx.FiberRequestContextKey, &reqctx.RequestContext{UserID: &ownerID})
		return c.Next()
	})
	app.Post("/contacts/request", handler.RequestContact)

	payload := bytes.NewBufferString(fmt.Sprintf(`{"target_user_id":"%s"}`, targetID.String()))
	req := httptest.NewRequest(http.MethodPost, "/contacts/request", payload)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app test: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	if service.created == nil {
		t.Fatal("expected service to receive the contact request")
	}
	if service.created.OwnerID != ownerID || service.created.ContactID != targetID {
		t.Fatalf("expected owner=%s target=%s, got owner=%s target=%s", ownerID, targetID, service.created.OwnerID, service.created.ContactID)
	}
}
