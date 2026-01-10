package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/pkg/errors"
	"github.com/zerodayz7/platform/pkg/validator"
)

func ValidateParams[T any]() fiber.Handler {
	return func(c *fiber.Ctx) error {
		params := new(T)
		if err := c.ParamsParser(params); err != nil {
			return errors.SendAppError(c, errors.ErrInvalidParams)
		}

		if errs := validator.Validate(params); len(errs) > 0 {
			meta := make(map[string]any)
			for k, v := range errs {
				meta[k] = v
			}

			appErr := errors.ErrValidationFailed
			appErr.Meta = meta
			return errors.SendAppError(c, appErr)
		}

		c.Locals("validatedParams", params)
		return c.Next()
	}
}
