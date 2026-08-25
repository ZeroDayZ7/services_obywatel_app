package handlers

import (
	"github.com/gofiber/fiber/v2"
	apperr "github.com/zerodayz7/platform/pkg/errors"
	rdy "github.com/zerodayz7/platform/pkg/redis"
)

//#region GetMeHandler
func GetMeHandler(c *fiber.Ctx) error {
	sess, ok := c.Locals("userSession").(rdy.UserSession)
	if !ok {
		return apperr.SendAppError(c, apperr.ErrUnauthorized)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"user_id":        sess.UserID,
		"username":       sess.Username,
		"email":          sess.Email,
		"role":           sess.Role,
		"institution_id": sess.InstitutionID,
		"department_id":  sess.DepartmentID,
		"permissions":    sess.Permissions,
	})
}
