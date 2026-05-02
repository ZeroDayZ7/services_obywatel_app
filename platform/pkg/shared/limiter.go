package shared

import (
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/zerodayz7/platform/pkg/errors"
)

type LimitGroup string

const (
	LimitGlobal        LimitGroup = "global"
	LimitAuth          LimitGroup = "auth"
	LimitUsers         LimitGroup = "users"
	LimitHealth        LimitGroup = "health"
	LimitAudit         LimitGroup = "audit"
	LimitReset         LimitGroup = "reset"
	LimitNotifications LimitGroup = "notifications"
)

var (
	limiters     = make(map[LimitGroup]fiber.Handler)
	limitersLock sync.RWMutex
)

func GetLimiter(group LimitGroup, storage fiber.Storage) fiber.Handler {
	limitersLock.RLock()
	if l, exists := limiters[group]; exists {
		limitersLock.RUnlock()
		return l
	}
	limitersLock.RUnlock()

	limitersLock.Lock()
	defer limitersLock.Unlock()

	if l, exists := limiters[group]; exists {
		return l
	}

	l := createLimiter(group, storage)
	limiters[group] = l
	return l
}

func createLimiter(group LimitGroup, storage fiber.Storage) fiber.Handler {
	presets := map[LimitGroup]struct {
		Max    int
		Window time.Duration
	}{
		LimitGlobal:        {Max: 100, Window: 60 * time.Second},
		LimitAuth:          {Max: 5, Window: 60 * time.Second},
		LimitHealth:        {Max: 50, Window: 30 * time.Second},
		LimitAudit:         {Max: 50, Window: 1 * time.Minute},
		LimitReset:         {Max: 3, Window: 1 * time.Hour},
		LimitNotifications: {Max: 30, Window: 1 * time.Minute},
	}

	cfg, ok := presets[group]
	if !ok {
		cfg = presets[LimitGlobal]
	}

	return limiter.New(limiter.Config{
		Max:        cfg.Max,
		Expiration: cfg.Window,
		Storage:    storage,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return errors.SendAppError(c, errors.ErrTooManyRequests)
		},
	})
}
