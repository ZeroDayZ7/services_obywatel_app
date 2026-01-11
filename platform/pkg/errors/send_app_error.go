package errors

import (
	"github.com/gofiber/fiber/v2"
)

// SendAppError teraz przyjmuje interfejs error, co rozwiązuje błędy kompilacji w handlerach
func SendAppError(c *fiber.Ctx, err error) error {
	// 1. Sprawdzamy czy błąd jest typu *AppError (Type Assertion)
	appErr, ok := err.(*AppError)
	if !ok {
		// Jeśli to błąd nieznany (np. z bazy danych lub sieci),
		// traktujemy go jako błąd wewnętrzny, aby nie wyciekły wrażliwe dane.
		appErr = ErrInternal
	}

	// 2. Logowanie
	if appErr.Type == Internal {
		c.Context().Logger().Printf("[ERROR] Internal: %v", appErr.Message)
	} else {
		c.Context().Logger().Printf("[WARN] %s: %v", appErr.Type, appErr.Message)
	}

	// 3. Mapowanie typów na statusy HTTP
	statusMap := map[ErrorType]int{
		Validation:   fiber.StatusBadRequest,
		Unauthorized: fiber.StatusUnauthorized,
		NotFound:     fiber.StatusNotFound,
		Internal:     fiber.StatusInternalServerError,
		BadRequest:   fiber.StatusBadRequest,
		Timeout:      fiber.StatusGatewayTimeout,
		Conflict:     fiber.StatusConflict,
	}

	status, exists := statusMap[appErr.Type]
	if !exists {
		status = fiber.StatusInternalServerError
	}

	// 4. Budowa odpowiedzi
	response := fiber.Map{
		"code":    appErr.Code,
		"message": appErr.Message,
	}

	if len(appErr.Meta) > 0 {
		response["meta"] = appErr.Meta
	}

	return c.Status(status).JSON(response)
}
