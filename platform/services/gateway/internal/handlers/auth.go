package handlers

import (
	"github.com/gofiber/fiber/v2"
	apperr "github.com/zerodayz7/platform/pkg/errors"
	rdy "github.com/zerodayz7/platform/pkg/redis"
)

// #region GetMeHandler
func GetMeHandler(c *fiber.Ctx) error {
	sess, ok := c.Locals("userSession").(*rdy.UserSession)
	if !ok || sess == nil {
		return apperr.SendAppError(c, apperr.ErrUnauthorized)
	}

	response := fiber.Map{
		"user_id":      sess.UserID,
		"username":     sess.Username,
		"email":        sess.Email,
		"role":         sess.Role,
		"permissions":  sess.Permissions,
		"is_read_only": sess.IsReadOnly,
	}

	if sess.Employee != nil {
		response["employee"] = sess.Employee
	}

	return c.Status(fiber.StatusOK).JSON(response)
}
