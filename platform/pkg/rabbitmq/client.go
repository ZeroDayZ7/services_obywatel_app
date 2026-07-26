package rabbitmq

import (
	"fmt"

	"github.com/zerodayz7/platform/pkg/shared"
)

type Config struct {
	Enabled  bool
	Host     string
	Port     int
	User     string
	Password string
	VHost    string
}

func (c Config) URL() string {
	return fmt.Sprintf("amqp://%s:%s@%s:%d%s", c.User, c.Password, c.Host, c.Port, c.VHost)
}

func NewPublisher(cfg Config) (EventPublisher, error) {
	log := shared.GetLogger()

	if !cfg.Enabled {
		log.Info("[RABBITMQ] Usługa jest wyłączona (RABBITMQ_ENABLED=false). Używanie NoOpPublisher.")
		return NewNoOpPublisher(), nil
	}

	pub, err := NewLivePublisher(cfg.URL())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to rabbitmq: %w", err)
	}

	log.Info("[RABBITMQ] Połączono pomyślnie z brokerem RabbitMQ.")
	return pub, nil
}
