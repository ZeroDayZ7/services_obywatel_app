package di

import (
	"net/http"
	"time"

	"github.com/zerodayz7/platform/pkg/redis"
	"github.com/zerodayz7/platform/pkg/types"
)

type Container struct {
	Redis          *redis.Client
	Cache          *redis.Cache
	HTTPClient     *http.Client
	InternalSecret []byte
	Services       types.ServicesConfig
	Config         *types.Config
}

func NewContainer(redisClient *redis.Client, cfg *types.Config) *Container {
	cache := redis.NewCache(redisClient, cfg.Session.Prefix, cfg.Session.TTL)

	return &Container{
		Redis: redisClient,
		Cache: cache,
		HTTPClient: &http.Client{
			Timeout: cfg.Proxy.RequestTimeout,
			Transport: &http.Transport{
				MaxIdleConns:        cfg.Proxy.MaxIdleConns,
				MaxIdleConnsPerHost: cfg.Proxy.MaxIdleConnsPerHost,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		InternalSecret: []byte(cfg.Internal.HMACSecret),
		Services:       cfg.Services,
		Config:         cfg,
	}
}
