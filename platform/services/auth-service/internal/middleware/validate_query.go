package middleware

import (
	"github.com/gofiber/fiber/v2"
)

func ValidateQuery[T any]() fiber.Handler {
	return func(c *fiber.Ctx) error {
		query := new(T)
		if err := c.QueryParser(query); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "INVALID_QUERY")
		}

		if errs := ValidateStruct(query); len(errs) > 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"code":   "VALIDATION_FAILED",
				"errors": errs,
			})
		}

		c.Locals("validatedQuery", query)
		return c.Next()
	}
}
