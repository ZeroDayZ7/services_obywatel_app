// internal/di/consumers.go
package di

import (
	"github.com/zerodayz7/platform/services/auth-service/internal/consumer"
)

type Consumers struct {
	CitizenConsumer *consumer.CitizenConsumer
}

//#region NewConsumers
func NewConsumers(services *Services) *Consumers {
	return &Consumers{
		CitizenConsumer: consumer.NewCitizenConsumer(services.ConsumerService),
	}
}
