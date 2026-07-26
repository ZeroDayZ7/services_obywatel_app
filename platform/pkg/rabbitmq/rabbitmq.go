package rabbitmq

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
)

type EventPublisher interface {
	Publish(ctx context.Context, routingKey string, payload []byte) error
	Consume(queueName string, routingKey string) (<-chan amqp.Delivery, error)
	Close() error
}
