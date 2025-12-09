package config

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/citizen-docs/internal/errors"
)

var RateLimitPresets = map[string]struct {
	Max    int
	Window time.Duration
}{
	"global": {Max: 100, Window: 60 * time.Second},
	"auth":   {Max: 10, Window: 60 * time.Second},
	"health": {Max: 20, Window: 60 * time.Second},
	"users":  {Max: 5, Window: 60 * time.Second},
	"visits": {Max: 30, Window: 30 * time.Minute},
}

func NewLimiter(group string) fiber.Handler {
	cfg, ok := RateLimitPresets[group]
	if !ok {
		cfg = RateLimitPresets["global"]
	}

	return limiter.New(limiter.Config{
		Max:        cfg.Max,
		Expiration: cfg.Window,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			log := shared.GetLogger()
			log.WarnMap("Rate limit exceeded", map[string]any{
				"ip":   c.IP(),
				"path": c.Path(),
			})
			return errors.SendAppError(c, errors.ErrTooManyRequests)
		},
	})
}
