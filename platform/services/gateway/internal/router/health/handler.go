package health

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

type HealthResponse struct {
	Status  string            `json:"status"`
	Service string            `json:"service"`
	Version string            `json:"version"`
	Time    string            `json:"time"`
	Checks  map[string]string `json:"checks"`
}

func (c *Checker) Handler(ctx *fiber.Ctx) error {
	checks := c.RunChecks()

	status := "ok"
	for _, v := range checks {
		if v != "ok" {
			status = "degraded"
			break
		}
	}

	resp := HealthResponse{
		Status:  status,
		Service: c.Service,
		Version: c.Version,
		Time:    time.Now().UTC().Format(time.RFC3339),
		Checks:  checks,
	}

	return ctx.JSON(resp)
}
