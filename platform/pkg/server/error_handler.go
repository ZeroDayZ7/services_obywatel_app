package server

import (
	stdErrors "errors"

	"github.com/gofiber/fiber/v2"
	appErrors "github.com/zerodayz7/platform/pkg/errors"
	"github.com/zerodayz7/platform/pkg/shared"
)

func ErrorHandler() fiber.ErrorHandler {
	log := shared.GetLogger()
	return func(c *fiber.Ctx, err error) error {
		var appErr *appErrors.AppError
		if stdErrors.As(err, &appErr) {
			return appErrors.SendAppError(c, appErr)
		}

		// Fiber error
		if e, ok := err.(*fiber.Error); ok {
			log.ErrorMap("HTTP error", map[string]any{"error": e.Error()})
			return c.Status(e.Code).JSON(fiber.Map{"error": e.Message})
		}

		log.ErrorMap("Server error", map[string]any{"error": err.Error()})
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}
}
