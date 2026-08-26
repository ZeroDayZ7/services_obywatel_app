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
	SenderID string
	HMACKey  []byte
}

//#region URL
func (c Config) URL() string {
	vhost := c.VHost
	if vhost != "" && vhost[0] != '/' {
		vhost = "/" + vhost
	}
	return fmt.Sprintf("amqp://%s:%s@%s:%d%s", c.User, c.Password, c.Host, c.Port, vhost)
}

//#region NewPublisher
func NewPublisher(cfg Config) (EventPublisher, error) {
	log := shared.GetLogger()

	if !cfg.Enabled {
		log.Info("[RABBITMQ] Usługa jest wyłączona (RABBITMQ_ENABLED=false). Używanie NoOpPublisher.")
		return NewNoOpPublisher(), nil
	}

	pub, err := NewLivePublisher(cfg.URL(), cfg.SenderID, cfg.HMACKey)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to rabbitmq: %w", err)
	}

	log.Info("[RABBITMQ] Połączono pomyślnie z brokerem RabbitMQ.")
	return pub, nil
}