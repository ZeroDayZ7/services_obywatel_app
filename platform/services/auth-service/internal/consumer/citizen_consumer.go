package consumer

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/auth-service/internal/service"
)

type CitizenCreatedPayload struct {
	CitizenID string `json:"citizen_id"`
	Email     string `json:"email"`
}

type CitizenConsumer struct {
	userService service.UserService
	log         *shared.Logger
}

func NewCitizenConsumer(userService service.UserService) *CitizenConsumer {
	return &CitizenConsumer{
		userService: userService,
		log:         shared.GetLogger(),
	}
}

func (c *CitizenConsumer) HandleCitizenCreated(ctx context.Context, headers amqp.Table, body []byte) error {
	c.log.Info("📨 Odebrano zdarzenie utworzenia obywatela przez RabbitMQ", "body", string(body))

	var payload CitizenCreatedPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		c.log.Error("❌ Błąd unmarshalingu payloadu zdarzenia", "error", err)
		return fmt.Errorf("unmarshal event failed: %w", err)
	}

	c.log.Info("👤 Przetwarzanie rejestracji obywatela", "citizen_id", payload.CitizenID, "email", payload.Email)

	// Wywołanie metody w serwisie biznesowym
	// return c.userService.CreateUserFromCitizen(ctx, payload.CitizenID, payload.Email)
	return nil
}
