package health

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type Checker struct {
	Redis     *redis.Client
	Service   string
	Version   string
	Upstreams []string
}

func (c *Checker) RunChecks() map[string]string {
	checks := make(map[string]string)
	ctx := context.Background()

	if c.Redis != nil {
		if err := c.Redis.Ping(ctx).Err(); err == nil {
			checks["redis"] = "ok"
		} else {
			checks["redis"] = "down"
		}
	}

	for _, url := range c.Upstreams {
		name := "upstream_" + extractName(url)
		client := &http.Client{Timeout: 2 * time.Second}
		if resp, err := client.Get(url); err == nil && resp.StatusCode < 500 {
			checks[name] = "ok"
		} else {
			checks[name] = "down"
		}
	}

	return checks
}

func extractName(url string) string {
	if i := strings.Index(url, "://"); i != -1 {
		url = url[i+3:]
	}
	if i := strings.Index(url, "/"); i != -1 {
		url = url[:i]
	}
	if i := strings.Index(url, ":"); i != -1 {
		url = url[:i]
	}
	return url
}
