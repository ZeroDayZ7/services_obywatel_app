package health

import (
	"context"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
)

type Checker struct {
	Redis     *redis.Client
	Service   string
	Version   string
	Upstreams map[string]string
}

func (c *Checker) RunChecks(ctx context.Context) map[string]string {
	checks := make(map[string]string)

	if c.Redis != nil {
		if err := c.Redis.Ping(ctx).Err(); err == nil {
			checks["redis"] = "ok"
		} else {
			checks["redis"] = "down"
		}
	}

	// Wspólny HTTP client zamiast alokacji w pętli
	client := &http.Client{Timeout: 2 * time.Second}

	for name, url := range c.Upstreams {
		key := "upstream_" + name

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			checks[key] = "down"
			continue
		}

		resp, err := client.Do(req)
		if err == nil && resp.StatusCode < 500 {
			checks[key] = "ok"
			_ = resp.Body.Close()
		} else {
			checks[key] = "down"
		}
	}

	return checks
}

func (c *Checker) Handler(ctx *fiber.Ctx) error {
	checks := c.RunChecks(ctx.UserContext())
	resp := NewResponse(c.Service, c.Version, checks)

	return ctx.JSON(resp)
}
