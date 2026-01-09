package health

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
)

type Checker struct {
	Redis     *redis.Client
	Service   string
	Version   string
	Upstreams []string
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

	for _, url := range c.Upstreams {
		name := "upstream_" + extractName(url)

		// Tworzymy request z kontekstem
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			checks[name] = "down"
			continue
		}

		client := &http.Client{Timeout: 2 * time.Second}
		resp, err := client.Do(req)

		if err == nil && resp.StatusCode < 500 {
			checks[name] = "ok"
			resp.Body.Close()
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

func (c *Checker) Handler(ctx *fiber.Ctx) error {
	// Używamy kontekstu z Fibera i Twojej nowej logiki
	checks := c.RunChecks(ctx.UserContext())

	// Używamy Twojego NewResponse
	resp := NewResponse(c.Service, c.Version, checks)

	return ctx.JSON(resp)
}
