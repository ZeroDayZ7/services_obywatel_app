package publisher

import (
	"context"
	"encoding/json"

	pkgredis "github.com/zerodayz7/platform/pkg/redis"
	"github.com/zerodayz7/platform/pkg/shared"
)

type RedisPublisher struct {
	client  *pkgredis.Client
	logger  *shared.Logger
	channel string
}

func NewRedisPublisher(client *pkgredis.Client, logger *shared.Logger) EventPublisher {
	return &RedisPublisher{
		client:  client,
		logger:  logger,
		channel: WebsocketChannel,
	}
}

func (p *RedisPublisher) Publish(ctx context.Context, event Event) error {
	if p == nil || p.client == nil || len(event) == 0 {
		return nil
	}

	payload, err := json.Marshal(event)
	if err != nil {
		if p.logger != nil {
			p.logger.Warn("failed to marshal websocket event payload", "event", event, "error", err)
		}
		return err
	}

	if err := p.client.Publish(ctx, p.channel, payload).Err(); err != nil {
		if p.logger != nil {
			p.logger.Warn("websocket event publication failed", "channel", p.channel, "event", event, "error", err)
		}
		return err
	}

	return nil
}
