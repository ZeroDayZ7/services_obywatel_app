package di

import (
	"net/http"

	"github.com/zerodayz7/platform/pkg/redis"
	"github.com/zerodayz7/platform/pkg/viper"
)

type Container struct {
	Redis          *redis.Client
	Cache          *redis.Cache
	HTTPClient     *http.Client
	InternalSecret []byte
	Config         *viper.Config
}

func NewContainer(redisClient *redis.Client, cfg *viper.Config) *Container {
	cache := redis.NewCache(redisClient, cfg.Session.Prefix, cfg.Session.TTL)

	return &Container{
		Redis: redisClient,
		Cache: cache,
		HTTPClient: &http.Client{
			Timeout: cfg.Proxy.RequestTimeout,
			Transport: &http.Transport{
				MaxIdleConns:        cfg.Proxy.MaxIdleConns,
				MaxIdleConnsPerHost: cfg.Proxy.MaxIdleConnsPerHost,
				IdleConnTimeout:     cfg.Proxy.IdleConnTimeout,
			},
		},
		InternalSecret: []byte(cfg.Internal.HMACSecret),
		Config:         cfg,
	}
}
