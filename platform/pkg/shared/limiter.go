package shared

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"

	"github.com/zerodayz7/platform/pkg/errors"
)

var Presets = map[string]struct {
	Max    int
	Window time.Duration
}{
	"global":        {Max: 100, Window: 60 * time.Second},
	"auth":          {Max: 10, Window: 60 * time.Second},
	"reset":         {Max: 3, Window: 60 * time.Second},
	"notifications": {Max: 30, Window: 60 * time.Second},
	"users":         {Max: 5, Window: 60 * time.Second},
	"health":        {Max: 20, Window: 30 * time.Second},
}

func NewLimiter(group string, storage fiber.Storage) fiber.Handler {
	cfg, ok := Presets[group]
	if !ok {
		cfg = Presets["global"]
	}

	return limiter.New(limiter.Config{
		Max:        cfg.Max,
		Expiration: cfg.Window,
		Storage:    storage,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			log := GetLogger()
			log.WarnMap("Rate limit exceeded", map[string]any{
				"ip":   c.IP(),
				"path": c.Path(),
			})
			return errors.SendAppError(c, errors.ErrTooManyRequests)
		},
	})
}
