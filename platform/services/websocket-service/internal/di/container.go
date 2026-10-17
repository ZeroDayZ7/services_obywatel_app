package di

import (
	"github.com/redis/go-redis/v9"
	"github.com/zerodayz7/platform/pkg/httpserver"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/websocket-service/internal/config"
	"github.com/zerodayz7/platform/services/websocket-service/internal/hub"
)

type Container struct {
	Redis    *redis.Client
	Hub      *hub.Hub
	Config   *config.Config
	Logger   *shared.Logger
	KeyStore *httpserver.KeyStore
}

func NewContainer(redisClient *redis.Client, hubManager *hub.Hub, logger *shared.Logger, cfg *config.Config, keyStore *httpserver.KeyStore) *Container {
	return &Container{
		Redis:    redisClient,
		Hub:      hubManager,
		Config:   cfg,
		Logger:   logger,
		KeyStore: keyStore,
	}
}
