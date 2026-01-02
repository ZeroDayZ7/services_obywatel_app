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

// Przyjmujemy surowe wartości, aby di nie musiało importować configu
func NewContainer(
	redisClient *redis.Client,
	cache *redis.Cache,
	requestTimeout time.Duration,
	maxIdleConns int,
	maxIdlePerHost int,
) *Container {
	return &Container{
		Redis: redisClient,
		Cache: cache,
		HTTPClient: &http.Client{
			Timeout: requestTimeout,
			Transport: &http.Transport{
				MaxIdleConns:        maxIdleConns,
				MaxIdleConnsPerHost: maxIdlePerHost,
				IdleConnTimeout:     90 * time.Second,
				DisableCompression:  false,
			},
		},
	}
}
