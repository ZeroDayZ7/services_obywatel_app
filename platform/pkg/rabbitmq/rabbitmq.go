// services_obywatel_app\platform\pkg\rabbitmq\rabbitmq.go
package rabbitmq

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/zerodayz7/platform/pkg/httpserver"
)

type HandlerFunc func(ctx context.Context, headers amqp.Table, body []byte) error

type EventPublisher interface {
	Publish(ctx context.Context, routingKey string, payload []byte) error
	Consume(queueName string, routingKey string) (<-chan amqp.Delivery, error)
	Subscribe(ctx context.Context, queueName string, routingKey string, handler HandlerFunc) error
	SubscribeWithAuth(ctx context.Context, queueName string, routingKey string, keyStore *httpserver.KeyStore, handler HandlerFunc) error

	Close() error
}
