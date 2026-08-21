package rabbitmq

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
)

type NoOpPublisher struct{}

func NewNoOpPublisher() *NoOpPublisher { return &NoOpPublisher{} }

func (p *NoOpPublisher) Publish(ctx context.Context, routingKey string, payload []byte) error {
	return nil
}

func (p *NoOpPublisher) Consume(queueName string, routingKey string) (<-chan amqp.Delivery, error) {
	ch := make(chan amqp.Delivery)
	close(ch)
	return ch, nil
}

func (p *NoOpPublisher) Subscribe(ctx context.Context, queueName string, routingKey string, handler HandlerFunc) error {
	<-ctx.Done()
	return ctx.Err()
}

func (p *NoOpPublisher) Close() error { return nil }
