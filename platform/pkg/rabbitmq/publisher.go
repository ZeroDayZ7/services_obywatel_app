package rabbitmq

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQPublisher struct {
	conn *amqp.Connection
	ch   *amqp.Channel
}

func NewLivePublisher(url string) (*RabbitMQPublisher, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	return &RabbitMQPublisher{conn: conn, ch: ch}, nil
}

func (p *RabbitMQPublisher) Publish(ctx context.Context, routingKey string, payload []byte) error {
	return p.ch.PublishWithContext(ctx,
		"amq.topic", // Default exchange
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        payload,
		},
	)
}

// Consume automatycznie deklaruje i konfiguruje kolejke, a nastepnie zwraca kanal z wiadomosciami
func (p *RabbitMQPublisher) Consume(queueName string, routingKey string) (<-chan amqp.Delivery, error) {
	// Deklarujemy trwała kolejke (durable: true)
	q, err := p.ch.QueueDeclare(
		queueName,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return nil, err
	}

	// Poniewaz uzywamy default exchange "", powiazanie z routingKey jako nazwa kolejki
	// nastepuje automatycznie, ale dla zachowania standardow zrobimy jawne bindowanie
	// na wypadek, gdybys w przyszlosci zmienil exchange na amq.topic
	err = p.ch.QueueBind(
		q.Name,
		routingKey,
		"amq.topic", // exchange name (pusty dla default exchange)
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}

	// Ograniczamy pobieranie (Prefetch: 1), zeby jeden worker nie blokowal wiadomosci,
	// gdy przetwarza ciezki task przez LLM (Pattern: Fair Dispatch)
	err = p.ch.Qos(1, 0, false)
	if err != nil {
		return nil, err
	}

	return p.ch.Consume(
		q.Name,
		"",    // consumer tag
		false, // auto-ack: FALSE - potwierdzamy recznie dopieru po przetworzeniu przez LLM!
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
}

func (p *RabbitMQPublisher) Close() error {
	if p.ch != nil {
		_ = p.ch.Close()
	}
	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}
