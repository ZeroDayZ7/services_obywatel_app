// platform/services/auth-service/internal/consumer/citizen_consumer.go
package consumer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/auth-service/internal/service"
)

type CitizenCreatedPayload struct {
	UserID          string `json:"user_id"`
	AgreementNumber string `json:"agreement_number"`
	SignedAt        string `json:"signed_at"`
}

type OutboxEnvelope struct {
	MessageID     string                `json:"message_id"`
	AggregateID   string                `json:"aggregate_id"`
	AggregateType string                `json:"aggregate_type"`
	EventType     string                `json:"event_type"`
	Payload       CitizenCreatedPayload `json:"payload"`
}

type CitizenConsumer struct {
	consumerService service.ConsumerService
	log             *shared.Logger
}

//#region NewCitizenConsumer
func NewCitizenConsumer(consumerService service.ConsumerService) *CitizenConsumer {
	return &CitizenConsumer{
		consumerService: consumerService,
		log:             shared.GetLogger(),
	}
}

//#region HandleCitizenCreated
func (c *CitizenConsumer) HandleCitizenCreated(ctx context.Context, headers amqp.Table, body []byte) error {
	c.log.Info("📨 Odebrano zdarzenie utworzenia obywatela przez RabbitMQ")

	var envelope OutboxEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		c.log.Error("❌ Błąd unmarshalingu koperty Outbox", "error", err)
		return fmt.Errorf("unmarshal envelope failed: %w", err)
	}

	// Konwersja ID z stringa na UUID
	citizenUUID, err := uuid.Parse(envelope.Payload.UserID)
	if err != nil {
		c.log.Error("❌ Niepoprawny format UUID obywatela w payloadzie", "user_id", envelope.Payload.UserID, "error", err)
		return fmt.Errorf("invalid citizen uuid: %w", err)
	}

	// Wywołanie dedykowanego serwisu dla konsumera
	err = c.consumerService.CreateCitizenAccountFromEvent(ctx, citizenUUID, envelope.Payload.AgreementNumber)
	if err != nil {
		return err // Zwrócenie błędu spowoduje Nack / ponowienie w RabbitMQ
	}

	return nil
}
