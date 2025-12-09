// internal/shared/cookies.go
package shared

import "github.com/gofiber/fiber/v2"

// SetSessionCookie ustawia ciasteczko sesji
func SetSessionCookie(c *fiber.Ctx, sessionID string) {
	c.Cookie(&fiber.Cookie{
		Name:     "__Host-session_",
		Value:    sessionID,
		HTTPOnly: true,
		Secure:   true, // włączone w produkcji
		SameSite: "Strict",
		Path:     "/",
	})
}

// ClearSessionCookie usuwa ciasteczko sesji (wylogowanie)
func ClearSessionCookie(c *fiber.Ctx) {
	c.Cookie(&fiber.Cookie{
		Name:     "__Host-session_",
		Value:    "",
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Strict",
		Path:     "/",
		MaxAge:   -1,
	})
}
