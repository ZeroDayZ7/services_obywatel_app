package server

import (
	stdErrors "errors"

	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/pkg/shared"
	apperrors "github.com/zerodayz7/platform/services/gateway/internal/errors"
)

func ErrorHandler() fiber.ErrorHandler {
	log := shared.GetLogger()
	return func(c *fiber.Ctx, err error) error {
		var appErr *apperrors.AppError
		if stdErrors.As(err, &appErr) {
			status := fiber.StatusBadRequest
			if appErr.Type == apperrors.Internal {
				status = fiber.StatusInternalServerError
			}

			logMap := map[string]any{
				"code": appErr.Code,
			}
			if appErr.Meta != nil {
				logMap["meta"] = appErr.Meta
			}
			if appErr.Err != nil {
				logMap["error"] = appErr.Err.Error()
				log.ErrorMap("AppError occurred", logMap)
			} else {
				log.WarnMap("AppError occurred", logMap)
			}

			response := fiber.Map{
				"code":    appErr.Code,
				"message": appErr.Message,
			}
			if len(appErr.Meta) > 0 {
				response["meta"] = appErr.Meta
			}

			return c.Status(status).JSON(response)
		}

		if e, ok := err.(*fiber.Error); ok {
			log.ErrorMap("HTTP error", map[string]any{
				"error": e.Error(),
			})
			return c.Status(e.Code).JSON(fiber.Map{"error": e.Message})
		}

		log.ErrorMap("Server error", map[string]any{
			"error": err.Error(),
		})
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}
}
