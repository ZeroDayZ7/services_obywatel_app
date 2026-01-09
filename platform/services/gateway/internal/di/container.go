package di

import (
	"net/http"
	"time"

	"github.com/zerodayz7/platform/pkg/redis"
)

type Container struct {
	Redis      *redis.Client
	Cache      *redis.Cache
	HTTPClient *http.Client
}

func NewContainer(
	redisClient *redis.Client,
	sessionPrefix string,
	sessionTTL time.Duration,
	requestTimeout time.Duration,
	maxIdleConns int,
	maxIdlePerHost int,
) *Container {

	cache := redis.NewCache(redisClient, sessionPrefix, sessionTTL)

	return &Container{
		Redis: redisClient,
		Cache: cache,
		HTTPClient: &http.Client{
			Timeout: requestTimeout,
			Transport: &http.Transport{
				MaxIdleConns:        maxIdleConns,
				MaxIdleConnsPerHost: maxIdlePerHost,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
}
