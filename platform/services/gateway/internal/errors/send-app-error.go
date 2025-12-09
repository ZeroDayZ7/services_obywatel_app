package errors

import (
	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/services/gateway/internal/shared/logger"
	"go.uber.org/zap"
)

func SendAppError(c *fiber.Ctx, err *AppError) error {
	log := logger.GetLogger()
	statusMap := map[ErrorType]int{
		Validation:   fiber.StatusBadRequest,
		Unauthorized: fiber.StatusUnauthorized,
		NotFound:     fiber.StatusNotFound,
		Internal:     fiber.StatusInternalServerError,
		BadRequest:   fiber.StatusBadRequest,
	}

	status, ok := statusMap[err.Type]
	if !ok {
		status = fiber.StatusInternalServerError
	}

	fields := []zap.Field{
		zap.String("code", err.Code),
		zap.String("type", string(err.Type)),
	}
	for k, v := range err.Meta {
		fields = append(fields, zap.Any(k, v))
	}
	if err.Err != nil {
		fields = append(fields, zap.String("error", err.Err.Error()))
		log.Error("AppError occurred", fields...)
	} else {
		log.Warn("AppError occurred", fields...)
	}

	response := fiber.Map{
		"code":    err.Code,
		"message": err.Message,
	}
	if len(err.Meta) > 0 {
		response["meta"] = err.Meta
	}

	return c.Status(status).JSON(response)
}
