package router

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"github.com/zerodayz7/platform/pkg/shared"
)

type HealthResponse struct {
	Status  string                 `json:"status"`
	Service string                 `json:"service"`
	Version string                 `json:"version"`
	Time    string                 `json:"time"`
	Checks  map[string]string      `json:"checks"`
	Details map[string]interface{} `json:"details,omitempty"`
}

type HealthChecker struct {
	Redis     *redis.Client
	Service   string
	Version   string
	Upstreams []string
}

func SetupHealthRoutes(app *fiber.App, redisClient *redis.Client, service, version string, upstreams []string) {
	health := app.Group("/health")
	health.Use(shared.NewLimiter("health"))

	checker := &HealthChecker{
		Redis:     redisClient,
		Service:   service,
		Version:   version,
		Upstreams: upstreams,
	}

	health.Get("/", checker.Handler)
}

// Handler Fiber
func (h *HealthChecker) Handler(c *fiber.Ctx) error {
	checks := make(map[string]string)
	now := time.Now().UTC()

	ctx := context.Background()

	// Redis
	if h.Redis != nil {
		if err := h.Redis.Ping(ctx).Err(); err == nil {
			checks["redis"] = "ok"
		} else {
			checks["redis"] = "down"
		}
	}

	// Upstreams
	for _, url := range h.Upstreams {
		name := "upstream_" + extractName(url)
		client := &http.Client{Timeout: 2 * time.Second}
		if resp, err := client.Get(url); err == nil && resp.StatusCode < 500 {
			checks[name] = "ok"
		} else {
			checks[name] = "down"
		}
	}

	// Ogólny status
	status := "ok"
	for _, v := range checks {
		if v != "ok" {
			status = "degraded"
			break
		}
	}

	resp := HealthResponse{
		Status:  status,
		Service: h.Service,
		Version: h.Version,
		Time:    now.Format(time.RFC3339),
		Checks:  checks,
	}

	return c.JSON(resp)
}

// Pomocnicza funkcja do wyciągania nazwy hosta z URL
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
