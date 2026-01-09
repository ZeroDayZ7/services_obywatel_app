package errors

import (
	"github.com/gofiber/fiber/v2"
)

func SendAppError(c *fiber.Ctx, err *AppError) error {

	if err.Type == Internal {
		c.Context().Logger().Printf("[ERROR] Internal: %v", err.Message)
	} else {
		c.Context().Logger().Printf("[WARN] %s: %v", err.Type, err.Message)
	}
	statusMap := map[ErrorType]int{
		Validation:   fiber.StatusBadRequest,
		Unauthorized: fiber.StatusUnauthorized,
		NotFound:     fiber.StatusNotFound,
		Internal:     fiber.StatusInternalServerError,
		BadRequest:   fiber.StatusBadRequest,
		Timeout:      fiber.StatusGatewayTimeout,
	}

	status, ok := statusMap[err.Type]
	if !ok {
		status = fiber.StatusInternalServerError
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
