package shared

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

func RequestLoggerMiddleware() fiber.Handler {
	allowedHeaders := []string{
		// "Content-Type",
		"User-Agent",
		"X-Device-Fingerprint",
		"Authorization",
		"X-Request-Id",
		"Accept-Language",
		// "Host",
		// "Content-Length",
		"X-Forwarded-For",
		"X-Real-Ip",
	}

	return func(c *fiber.Ctx) error {
		start := time.Now()
		log := GetLogger()
		isDev := log.Core().Enabled(zap.DebugLevel)

		// 1. Wyciąganie Body
		var body any
		if c.Method() == fiber.MethodPost || c.Method() == fiber.MethodPut || c.Method() == fiber.MethodPatch {
			_ = c.BodyParser(&body)
		}

		if isDev {
			fmt.Printf("\n--- [DEBUG] Incoming Request ---\n")
			fmt.Printf("Method: %s\nPath:   %s\n", c.Method(), c.Path())

			if body != nil {
				fmt.Printf("Body:\n")
				if bodyMap, ok := body.(map[string]any); ok {
					for k, v := range bodyMap {
						// MASKOWANIE SEKRETÓW W KONSOLI
						displayValue := v
						if isSensitive(k) {
							displayValue = "********"
						}
						fmt.Printf("  %s: %v\n", k, displayValue)
					}
				} else {
					fmt.Printf("  %+v\n", body)
				}
			}

			fmt.Printf("Headers:\n")
			for _, h := range allowedHeaders {
				val := c.Get(h)

				if h == "X-Request-Id" && val == "" {
					if rid := c.Locals("requestid"); rid != nil {
						val = fmt.Sprintf("%v", rid)
					}
				}

				if val != "" {
					fmt.Printf("  %s: %s\n", h, val)
				}
			}
			fmt.Printf("-------------------------------\n\n")
		}

		// Kontynuacja zapytania
		err := c.Next()

		// Obliczenie czasu trwania zapytania
		latency := time.Since(start)

		// log (Strukturalny)
		if isDev {
			// W DEV logujemy wszystko na DEBUG (z body i czasem)
			log.DebugRequest(
				"Request processed",
				c.Method(),
				c.Path(),
				c.Response().StatusCode(),
				latency.String(),
				body, // Twoja struktura body zostanie automatycznie zdekodowana i zmaskowana
			)
		} else {
			// Na PROD logujemy tylko Info (bez Body dla oszczędności, ale ze statusem i czasem)
			log.Info("Request completed",
				zap.String("method", c.Method()),
				zap.String("path", c.Path()),
				zap.Int("status", c.Response().StatusCode()),
				zap.String("latency", latency.String()),
				zap.Any("request_id", c.Locals("requestid")),
				zap.String("ip", c.IP()),
			)
		}

		return err
	}
}
