package rabbitmq

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/zerodayz7/platform/pkg/crypto"
	"github.com/zerodayz7/platform/pkg/shared"
)

type RabbitMQPublisher struct {
	conn     *amqp.Connection
	ch       *amqp.Channel
	senderID string
	hmacKey  []byte
}

// #region NewLivePublisher
func NewLivePublisher(url string, senderID string, hmacKey []byte) (*RabbitMQPublisher, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	return &RabbitMQPublisher{
		conn:     conn,
		ch:       ch,
		senderID: senderID,
		hmacKey:  hmacKey,
	}, nil
}

// #region Publish
func (p *RabbitMQPublisher) Publish(ctx context.Context, routingKey string, payload []byte) error {
	headers := amqp.Table{
		"X-Sender-ID": p.senderID,
	}

	if len(p.hmacKey) > 0 {
		headers["X-Signature"] = crypto.ComputeHMAC256Hex(payload, p.hmacKey)
	}

	return p.ch.PublishWithContext(ctx,
		"amq.topic",
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Headers:     headers,
			Body:        payload,
		},
	)
}

// #region Consume
func (p *RabbitMQPublisher) Consume(queueName string, routingKey string) (<-chan amqp.Delivery, error) {
	q, err := p.ch.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}

	err = p.ch.QueueBind(
		q.Name,
		routingKey,
		"amq.topic",
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}

	err = p.ch.Qos(1, 0, false)
	if err != nil {
		return nil, err
	}

	return p.ch.Consume(
		q.Name,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
}

// #region Subscribe
func (p *RabbitMQPublisher) Subscribe(ctx context.Context, queueName string, routingKey string, handler HandlerFunc) error {
	log := shared.GetLogger()

	msgs, err := p.Consume(queueName, routingKey)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case d, ok := <-msgs:
			if !ok {
				return nil
			}

			if err := handler(ctx, d.Headers, d.Body); err != nil {
				log.Error("[RABBITMQ] Błąd przetwarzania wiadomości", "queue", queueName, "error", err)
				_ = d.Nack(false, false)
				continue
			}

			_ = d.Ack(false)
		}
	}
}

// #region Close
func (p *RabbitMQPublisher) Close() error {
	if p.ch != nil {
		_ = p.ch.Close()
	}
	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}
