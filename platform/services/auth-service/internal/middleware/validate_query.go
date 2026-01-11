package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/pkg/errors"
	"github.com/zerodayz7/platform/pkg/validator"
)

func ValidateQuery[T any]() fiber.Handler {
	return func(c *fiber.Ctx) error {
		query := new(T)
		if err := c.QueryParser(query); err != nil {
			return errors.SendAppError(c, errors.ErrInvalidQuery)
		}

		if errs := validator.Validate(query); len(errs) > 0 {
			meta := make(map[string]any)
			for k, v := range errs {
				meta[k] = v
			}

			appErr := errors.ErrValidationFailed
			appErr.Meta = meta
			return errors.SendAppError(c, appErr)
		}

		c.Locals("validatedQuery", query)
		return c.Next()
	}
}
