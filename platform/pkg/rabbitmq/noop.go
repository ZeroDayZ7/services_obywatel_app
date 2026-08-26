package rabbitmq

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/zerodayz7/platform/pkg/httpserver"
)

type NoOpPublisher struct{}

//#region NewNoOpPublisher
func NewNoOpPublisher() *NoOpPublisher { return &NoOpPublisher{} }

//#region Publish
func (p *NoOpPublisher) Publish(ctx context.Context, routingKey string, payload []byte) error {
	return nil
}

//#region Consume
func (p *NoOpPublisher) Consume(queueName string, routingKey string) (<-chan amqp.Delivery, error) {
	ch := make(chan amqp.Delivery)
	close(ch)
	return ch, nil
}

//#region Subscribe
func (p *NoOpPublisher) Subscribe(ctx context.Context, queueName string, routingKey string, handler HandlerFunc) error {
	<-ctx.Done()
	return ctx.Err()
}

//#region SubscribeWithAuth
func (p *NoOpPublisher) SubscribeWithAuth(
	ctx context.Context,
	queueName string,
	routingKey string,
	keyStore *httpserver.KeyStore,
	handler HandlerFunc,
) error {
	<-ctx.Done()
	return ctx.Err()
}

//#region Close
func (p *NoOpPublisher) Close() error { return nil }
