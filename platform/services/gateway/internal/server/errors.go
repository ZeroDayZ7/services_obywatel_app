package server

import (
	stdErrors "errors"

	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/pkg/logger"
	apperrors "github.com/zerodayz7/platform/services/gateway/internal/errors"
	"github.com/zerodayz7/platform/services/gateway/internal/shared/logger"
	"go.uber.org/zap"
)

func ErrorHandler() fiber.ErrorHandler {
	log := logger.GetLogger()
	return func(c *fiber.Ctx, err error) error {
		var appErr *apperrors.AppError
		if stdErrors.As(err, &appErr) {
			status := fiber.StatusBadRequest
			if appErr.Type == apperrors.Internal {
				status = fiber.StatusInternalServerError
			}

			logFields := []zap.Field{zap.String("code", appErr.Code)}
			if appErr.Meta != nil {
				logFields = append(logFields, zap.Any("meta", appErr.Meta))
			}
			if appErr.Err != nil {
				logFields = append(logFields, zap.String("error", appErr.Err.Error()))
				log.Error("AppError occurred", logFields...)
			} else {
				log.Warn("AppError occurred", logFields...)
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
			log.Error("HTTP error", zap.Error(err))
			return c.Status(e.Code).JSON(fiber.Map{"error": e.Message})
		}

		log.Error("Server error", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}
}
