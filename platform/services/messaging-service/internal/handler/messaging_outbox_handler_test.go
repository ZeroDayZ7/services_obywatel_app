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

type stubMessagingService struct {
	processOutbox func(ctx context.Context, userID uuid.UUID, req model.OutboxBatchRequest) (*model.OutboxBatchResponse, error)
}

func (s *stubMessagingService) GetDeltaSync(ctx context.Context, userID uuid.UUID, req model.SyncDeltaRequest) (*model.SyncDeltaResponse, error) {
	return nil, nil
}

func (s *stubMessagingService) ProcessOutbox(ctx context.Context, userID uuid.UUID, req model.OutboxBatchRequest) (*model.OutboxBatchResponse, error) {
	if s.processOutbox != nil {
		return s.processOutbox(ctx, userID, req)
	}
	return &model.OutboxBatchResponse{ProcessedCount: len(req.Messages)}, nil
}

func (s *stubMessagingService) SendMessage(ctx context.Context, senderID uuid.UUID, msg *model.Message) error {
	return nil
}

func (s *stubMessagingService) GetContacts(ctx context.Context, ownerID uuid.UUID, sinceVersion uint64) ([]model.Contact, error) {
	return nil, nil
}

func (s *stubMessagingService) GetMessages(ctx context.Context, userID uuid.UUID, conversationID string, limit, offset int) ([]model.Message, error) {
	return nil, nil
}

func (s *stubMessagingService) GetConversations(ctx context.Context, userID uuid.UUID) ([]model.Conversation, error) {
	return nil, nil
}

func (s *stubMessagingService) CreateConversation(ctx context.Context, userID uuid.UUID, req model.CreateConversationRequest) (*model.Conversation, error) {
	return nil, nil
}

func (s *stubMessagingService) GetConversationByID(ctx context.Context, userID uuid.UUID, conversationID string) (*model.Conversation, error) {
	return nil, nil
}

func (s *stubMessagingService) MarkAsRead(ctx context.Context, userID uuid.UUID, conversationID string) error {
	return nil
}

func (s *stubMessagingService) GetActivationStatus(ctx context.Context, userID uuid.UUID) (*model.MessagingActivationStatusResponse, error) {
	return nil, nil
}

func (s *stubMessagingService) GetCurrentTerms(ctx context.Context) (*model.MessagingTermsDocument, error) {
	return nil, nil
}

func (s *stubMessagingService) ActivateMessaging(ctx context.Context, userID uuid.UUID, req model.ActivateMessagingRequest) (*model.MessagingActivationStatusResponse, error) {
	return nil, nil
}

func (s *stubMessagingService) AcceptTerms(ctx context.Context, userID uuid.UUID, req model.AcceptTermsRequest) (*model.MessagingActivationStatusResponse, error) {
	return nil, nil
}

func (s *stubMessagingService) UploadDeviceKeys(ctx context.Context, userID uuid.UUID, req model.UploadDeviceKeysRequest) error {
	return nil
}

func (s *stubMessagingService) GetUserKeyBundle(ctx context.Context, targetUserID string) (*model.PreKeyBundleDto, error) {
	return nil, nil
}

func (s *stubMessagingService) GetUserPreKeys(ctx context.Context, targetUserID string) (*model.UserPreKeysResponse, error) {
	return nil, nil
}

func TestMessagingHandlerProcessOutbox_AcceptsWrappedMessagesPayload(t *testing.T) {
	userID := uuid.New()
	service := &stubMessagingService{
		processOutbox: func(ctx context.Context, gotUserID uuid.UUID, req model.OutboxBatchRequest) (*model.OutboxBatchResponse, error) {
			if gotUserID != userID {
				t.Fatalf("expected userID %s, got %s", userID, gotUserID)
			}
			if len(req.Messages) != 1 {
				t.Fatalf("expected 1 messages, got %d", len(req.Messages))
			}
			return &model.OutboxBatchResponse{ProcessedCount: 1}, nil
		},
	}

	handler := NewMessagingHandler(service)
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals(reqctx.FiberRequestContextKey, &reqctx.RequestContext{UserID: &userID})
		return c.Next()
	})
	app.Post("/sync/outbox", handler.ProcessOutbox)

	payload := fmt.Sprintf(`{"messages":[{"event_id":"%s","event_type":"SEND_MESSAGE","idempotency_key":"%s","conversation_id":"%s","payload":{"message_id":"msg-1"}}]}`,
		uuid.NewString(),
		uuid.NewString(),
		uuid.NewString(),
	)

	req := httptest.NewRequest(http.MethodPost, "/sync/outbox", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app test: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestMessagingHandlerProcessOutbox_AcceptsDirectMessageEnvelopePayload(t *testing.T) {
	userID := uuid.New()
	conversationID := uuid.New()
	key := uuid.NewString()
	service := &stubMessagingService{
		processOutbox: func(ctx context.Context, gotUserID uuid.UUID, req model.OutboxBatchRequest) (*model.OutboxBatchResponse, error) {
			if gotUserID != userID {
				t.Fatalf("expected userID %s, got %s", userID, gotUserID)
			}
			if len(req.Messages) != 1 {
				t.Fatalf("expected 1 messages, got %d", len(req.Messages))
			}
			if req.Messages[0].Ciphertext == "" {
				t.Fatal("expected ciphertext in direct outbox envelope")
			}
			if req.Messages[0].ConversationID == nil || req.Messages[0].ConversationID.String() != conversationID.String() {
				t.Fatalf("expected conversation %s, got %v", conversationID, req.Messages[0].ConversationID)
			}
			return &model.OutboxBatchResponse{ProcessedCount: 1}, nil
		},
	}

	handler := NewMessagingHandler(service)
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals(reqctx.FiberRequestContextKey, &reqctx.RequestContext{UserID: &userID})
		return c.Next()
	})
	app.Post("/sync/outbox", handler.ProcessOutbox)

	payload := fmt.Sprintf(`{"messages":[{"conversation_id":"%s","sender_device_id":"device-1","ciphertext":"Zm9v","type":1,"idempotency_key":"%s"}]}`,
		conversationID,
		key,
	)
	req := httptest.NewRequest(http.MethodPost, "/sync/outbox", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app test: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestMessagingHandlerProcessOutbox_IgnoresEmptyConversationID(t *testing.T) {
	userID := uuid.New()
	service := &stubMessagingService{
		processOutbox: func(ctx context.Context, gotUserID uuid.UUID, req model.OutboxBatchRequest) (*model.OutboxBatchResponse, error) {
			if gotUserID != userID {
				t.Fatalf("expected userID %s, got %s", userID, gotUserID)
			}
			if len(req.Messages) != 1 {
				t.Fatalf("expected 1 messages, got %d", len(req.Messages))
			}
			if req.Messages[0].ConversationID != nil {
				t.Fatalf("expected empty conversation_id to be ignored, got %v", req.Messages[0].ConversationID)
			}
			return &model.OutboxBatchResponse{ProcessedCount: 0}, nil
		},
	}

	handler := NewMessagingHandler(service)
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals(reqctx.FiberRequestContextKey, &reqctx.RequestContext{UserID: &userID})
		return c.Next()
	})
	app.Post("/sync/outbox", handler.ProcessOutbox)

	payload := `{"messages":[{"event_id":"01a1221b-0a84-71be-9f4f-14afafc0b841","idempotency_key":"01a1221b-0a84-71be-9f4f-14afafc0b841","message_id":"01a1221b-0a84-71be-9f4f-14afafc0b841","event_type":"SEND_MESSAGE","conversation_id":"","sender_device_id":"8e681e7c-6ccc-4854-8fc6-b6dd5074f39e","ciphertext":"","type":1,"content":"","created_at":"2026-10-09T19:19:19.933148Z","outbox_event_id":"01a1221b-0a84-71be-9f4f-14afafc0b841"}]}`
	req := httptest.NewRequest(http.MethodPost, "/sync/outbox", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app test: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestMessagingHandlerProcessOutbox_RejectsRootArrayPayload(t *testing.T) {
	userID := uuid.New()
	handler := NewMessagingHandler(&stubMessagingService{})
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals(reqctx.FiberRequestContextKey, &reqctx.RequestContext{UserID: &userID})
		return c.Next()
	})
	app.Post("/sync/outbox", handler.ProcessOutbox)

	payload := `[{"event_id":"11111111-1111-4111-8111-111111111111","event_type":"SEND_MESSAGE"}]`
	req := httptest.NewRequest(http.MethodPost, "/sync/outbox", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app test: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}
