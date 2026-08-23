package consumer

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/zerodayz7/platform/services/auth-service/internal/service"
)

type CitizenCreatedPayload struct {
	CitizenID string `json:"citizen_id"`
	Email     string `json:"email"`
}

type CitizenConsumer struct {
	userService service.UserService
}

func NewCitizenConsumer(userService service.UserService) *CitizenConsumer {
	return &CitizenConsumer{
		userService: userService,
	}
}

func (c *CitizenConsumer) HandleCitizenCreated(ctx context.Context, headers amqp.Table, body []byte) error {
	var payload CitizenCreatedPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return fmt.Errorf("unmarshal event failed: %w", err)
	}

	// Wywołanie metody w serwisie biznesowym
	// return c.userService.CreateUserFromCitizen(ctx, payload.CitizenID, payload.Email)
	return nil
}
