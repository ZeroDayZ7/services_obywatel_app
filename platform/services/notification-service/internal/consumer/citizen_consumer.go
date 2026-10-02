package consumer

import (
	"context"
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/notification-service/internal/model"
	notificationService "github.com/zerodayz7/platform/services/notification-service/internal/service"
)

type CitizenCreatedPayload struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
}

type OutboxEnvelope struct {
	MessageID     string                `json:"message_id"`
	AggregateID   string                `json:"aggregate_id"`
	AggregateType string                `json:"aggregate_type"`
	EventType     string                `json:"event_type"`
	Payload       CitizenCreatedPayload `json:"payload"`
}

type CitizenConsumer struct {
	notificationSvc *notificationService.NotificationService
	log             *shared.Logger
}

func NewCitizenConsumer(svc *notificationService.NotificationService) *CitizenConsumer {
	return &CitizenConsumer{
		notificationSvc: svc,
		log:             shared.GetLogger(),
	}
}

func (c *CitizenConsumer) HandleCitizenCreated(ctx context.Context, headers amqp.Table, body []byte) error {
	c.log.Info("📨 NotificationService: received citizen.created event")

	var envelope OutboxEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		c.log.Error("❌ NotificationService: failed to unmarshal envelope", "error", err)
		return err
	}

	// Build notification
	n := &model.Notification{
		// Depending on identity payload, parse UserID -> UUID in service layer if needed
		Title:    "Aktywacja konta w systemie Obywatel Plus",
		Content:  "Witaj! Twój profil cyfrowy został pomyślnie utworzony. Pamiętaj o włączeniu 2FA.",
		Priority: "normal",
		Category: "SYSTEM_WELCOME",
	}

	if err := c.notificationSvc.Send(ctx, n); err != nil {
		c.log.ErrorObj("NotificationService: failed to persist notification", err)
		return err
	}

	return nil
}
