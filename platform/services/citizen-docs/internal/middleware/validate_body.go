package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/services/citizen-docs/internal/errors"
)

func ValidateBody[T any]() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var body T
		if err := c.BodyParser(&body); err != nil {
			return errors.SendAppError(c, errors.ErrInvalidJSON)
		}

		if errs := ValidateStruct(body); len(errs) > 0 {

			meta := make(map[string]any)
			for k, v := range errs {
				meta[k] = v
			}

			appErr := errors.ErrValidationFailed
			appErr.Meta = meta
			return errors.SendAppError(c, appErr)
		}

		c.Locals("validatedBody", body)
		return c.Next()
	}
}
