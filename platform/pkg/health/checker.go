// pkg/health/checker.go
package health

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Checker struct {
	DB        *gorm.DB
	Redis     *redis.Client
	Upstreams []string // np. []string{"http://auth-service:8082/health"}
	Service   string
	Version   string
}

type Result struct {
	Status  string            `json:"status"`
	Service string            `json:"service"`
	Version string            `json:"version"`
	Time    string            `json:"time"`
	Checks  map[string]string `json:"checks"`
}

func (c *Checker) Check(ctx context.Context) Result {
	checks := make(map[string]string)
	now := time.Now().UTC()

	// DB
	if c.DB != nil {
		sqlDB, _ := c.DB.DB()
		if sqlDB.Ping() == nil {
			checks["database"] = "ok"
		} else {
			checks["database"] = "failed"
		}
	}

	// Redis
	if c.Redis != nil {
		if c.Redis.Ping(ctx).Err() == nil {
			checks["redis"] = "ok"
		} else {
			checks["redis"] = "failed"
		}
	}

	// Upstreams
	for _, url := range c.Upstreams {
		name := "upstream_" + extractName(url)
		client := &http.Client{Timeout: 2 * time.Second}
		if resp, err := client.Get(url); err == nil && resp.StatusCode < 500 {
			checks[name] = "ok"
		} else {
			checks[name] = "down"
		}
	}

	status := "ok"
	for _, v := range checks {
		if v != "ok" {
			status = "degraded"
			break
		}
	}

	return Result{
		Status:  status,
		Service: c.Service,
		Version: c.Version,
		Time:    now.Format(time.RFC3339),
		Checks:  checks,
	}
}

func extractName(url string) string {
	// http://auth-service:8082/health → auth-service
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
