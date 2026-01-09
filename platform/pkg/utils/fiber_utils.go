package utils

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/zerodayz7/platform/pkg/errors"
)

func GetUserID(c *fiber.Ctx) (uuid.UUID, error) {
	idStr := c.Get("X-User-Id")
	if idStr == "" {
		idStr = c.Get("X-User-ID")
	}

	if idStr == "" {
		return uuid.Nil, errors.ErrInvalidToken
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return uuid.Nil, errors.ErrInvalidToken
	}

	return id, nil
}

// ParseUUID extract and parses a UUID from a specific path parameter (e.g. /:id)
func ParseUUID(c *fiber.Ctx, paramName string) (uuid.UUID, error) {
	idStr := c.Params(paramName)
	id, err := uuid.Parse(idStr)
	if err != nil {
		return uuid.Nil, errors.ErrInvalidRequest
	}
	return id, nil
}
