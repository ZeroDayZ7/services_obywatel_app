// cmdr: cmdr: middleware\validator.go

package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/pkg/errors"
	"github.com/zerodayz7/platform/pkg/validator"
)

// Klucze w kontekście Fibera, do których serwisy mogą się odwołać przez c.Locals()
const (
	CtxValidatedBody   = "platform_validated_body"
	CtxValidatedQuery  = "platform_validated_query"
	CtxValidatedParams = "platform_validated_params"
)

// ValidateBody parsuje i waliduje strukturę z body żądania (JSON)
func ValidateBody[T any]() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var body T
		if err := c.BodyParser(&body); err != nil {
			return errors.SendAppError(c, errors.ErrInvalidJSON)
		}

		if errs := validator.Validate(body); len(errs) > 0 {
			return errors.SendAppError(c, formatValidationError(errs))
		}

		c.Locals(CtxValidatedBody, body)
		return c.Next()
	}
}

// ValidateQuery parsuje i waliduje parametry z query string URL
func ValidateQuery[T any]() fiber.Handler {
	return func(c *fiber.Ctx) error {
		query := new(T)
		if err := c.QueryParser(query); err != nil {
			return errors.SendAppError(c, errors.ErrInvalidQuery)
		}

		if errs := validator.Validate(query); len(errs) > 0 {
			return errors.SendAppError(c, formatValidationError(errs))
		}

		c.Locals(CtxValidatedQuery, *query)
		return c.Next()
	}
}

// ValidateParams parsuje i waliduje parametry ze ścieżki URL (path params)
func ValidateParams[T any]() fiber.Handler {
	return func(c *fiber.Ctx) error {
		params := new(T)
		if err := c.ParamsParser(params); err != nil {
			return errors.SendAppError(c, errors.ErrInvalidParams)
		}

		if errs := validator.Validate(params); len(errs) > 0 {
			return errors.SendAppError(c, formatValidationError(errs))
		}

		c.Locals(CtxValidatedParams, *params)
		return c.Next()
	}
}

// GetBody bezpiecznie wyciąga i rzutuje zwalidowane body z kontekstu
func GetBody[T any](c *fiber.Ctx) T {
	val, ok := c.Locals(CtxValidatedBody).(T)
	if !ok {
		var zero T
		return zero
	}
	return val
}

// GetQuery bezpiecznie wyciąga i rzutuje zwalidowane query z kontekstu
func GetQuery[T any](c *fiber.Ctx) T {
	val, ok := c.Locals(CtxValidatedQuery).(T)
	if !ok {
		var zero T
		return zero
	}
	return val
}

// GetParams bezpiecznie wyciąga i rzutuje zwalidowane path params z kontekstu
func GetParams[T any](c *fiber.Ctx) T {
	val, ok := c.Locals(CtxValidatedParams).(T)
	if !ok {
		var zero T
		return zero
	}
	return val
}

// Pomocnicza funkcja tworząca bezpieczną kopię błędu walidacji (Thread-Safe)
//#region formatValidationError
func formatValidationError(errs map[string]string) *errors.AppError {
	meta := make(map[string]any, len(errs))
	for k, v := range errs {
		meta[k] = v
	}
	return &errors.AppError{
		Code:    errors.ErrValidationFailed.Code,
		Type:    errors.ErrValidationFailed.Type,
		Message: errors.ErrValidationFailed.Message,
		Meta:    meta,
	}
}
