package errors

import (
	"maps"

	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/pkg/shared"
)

func SendAppError(c *fiber.Ctx, err *AppError) error {
	log := shared.GetLogger()

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

	// Logowanie
	logFields := map[string]any{
		"code": err.Code,
		"type": string(err.Type),
	}
	maps.Copy(logFields, err.Meta)
	if err.Err != nil {
		logFields["error"] = err.Err.Error()
		log.ErrorMap("AppError occurred", logFields)
	} else {
		log.WarnMap("AppError occurred", logFields)
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
