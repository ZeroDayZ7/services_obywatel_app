package di

import (
	consumer "github.com/zerodayz7/platform/services/notification-service/internal/consumer"
)

type Consumers struct {
    CitizenConsumer *consumer.CitizenConsumer
}

func NewConsumers(services *Services) *Consumers {
    return &Consumers{
        CitizenConsumer: consumer.NewCitizenConsumer(services.NotificationSvc),
    }
}
