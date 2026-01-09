package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/pkg/errors"
)

func ValidateBody[T any]() fiber.Handler {
	return func(c *fiber.Ctx) error {
		body := new(T)

		if err := c.BodyParser(body); err != nil {
			return errors.SendAppError(c, errors.ErrInvalidJSON)
		}

		if errs := ValidateStruct(*body); len(errs) > 0 {
			appErr := errors.ErrValidationFailed
			finalErr := &errors.AppError{
				Code:    appErr.Code,
				Type:    appErr.Type,
				Message: appErr.Message,
				Meta:    make(map[string]any),
			}

			for k, v := range errs {
				finalErr.Meta[k] = v
			}

			return errors.SendAppError(c, finalErr)
		}

		c.Locals("validatedBody", *body)
		return c.Next()
	}
}
