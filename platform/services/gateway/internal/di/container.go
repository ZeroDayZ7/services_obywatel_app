package di

import (
	"net/http"

	"github.com/zerodayz7/platform/pkg/httpserver"
	"github.com/zerodayz7/platform/pkg/rabbitmq"
	"github.com/zerodayz7/platform/pkg/redis"
	"github.com/zerodayz7/platform/services/gateway/config"
)

type Container struct {
	Redis          *redis.Client
	Cache          *redis.Cache
	EventPublisher rabbitmq.EventPublisher
	HTTPClient     *http.Client
	Config         *config.Config
	KeyStore       *httpserver.KeyStore
}

//#region NewContainer
func NewContainer(
	redisClient *redis.Client,
	eventPublisher rabbitmq.EventPublisher,
	cfg *config.Config,
	keyStore *httpserver.KeyStore,
) *Container {
	cache := redis.NewCache(redisClient, cfg.Session.TTL)

	return &Container{
		Redis:          redisClient,
		Cache:          cache,
		EventPublisher: eventPublisher,
		HTTPClient: &http.Client{
			Timeout: cfg.Proxy.RequestTimeout,
			Transport: &http.Transport{
				MaxIdleConns:        cfg.Proxy.MaxIdleConns,
				MaxIdleConnsPerHost: cfg.Proxy.MaxIdleConnsPerHost,
				IdleConnTimeout:     cfg.Proxy.IdleConnTimeout,
			},
		},
		Config:   cfg,
		KeyStore: keyStore,
	}
}

//#region GetHMACKey
func (c *Container) GetHMACKey(serviceID string) ([]byte, uint32, bool) {
	if c.KeyStore == nil {
		return nil, 0, false
	}
	return c.KeyStore.GetKey(serviceID)
}
