package middleware

import (
	"github.com/gofiber/fiber/v2"
)

func ValidateParams[T any]() fiber.Handler {
	return func(c *fiber.Ctx) error {
		params := new(T)
		if err := c.ParamsParser(params); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "INVALID_PARAMS")
		}

		if errs := ValidateStruct(params); len(errs) > 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"code":   "VALIDATION_FAILED",
				"errors": errs,
			})
		}

		c.Locals("validatedParams", params)
		return c.Next()
	}
}
