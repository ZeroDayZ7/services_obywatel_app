package shared

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// #region RequestLoggerMiddleware
func RequestLoggerMiddleware() fiber.Handler {
	allowedHeaders := []string{
		"User-Agent",
		"X-Device-Fingerprint",
		"Authorization",
		"X-Request-Id",
		"X-Operation-Id",
		"Accept-Language",
		"X-Forwarded-For",
		"X-Real-Ip",
	}

	return func(c *fiber.Ctx) error {
		start := time.Now()
		log := GetLogger()
		isDev := log.Core().Enabled(zap.DebugLevel)

		// 1. Wykonanie żądania w handlerach
		err := c.Next()

		latency := time.Since(start)
		status := c.Response().StatusCode()
		requestID := c.Locals("requestid")

		// 2. Podsumowanie DEV (Wejście + Wyjście w jednym bloku)
		if isDev {
			fmt.Printf("\n=== [DEBUG HTTP TRANSACTION] ===\n")
			fmt.Printf("Method: %s | Path: %s | Status: %d | Latency: %s\n", c.Method(), c.Path(), status, latency)

			// Log only headers and request identifiers; never print incoming payloads or E2EE key material.
			// Nagłówki wejściowe
			fmt.Printf("Headers:\n")
			for _, h := range allowedHeaders {
				val := c.Get(h)
				if h == "X-Request-Id" && val == "" && requestID != nil {
					val = fmt.Sprintf("%v", requestID)
				}
				if val != "" {
					fmt.Printf("  %s: %s\n", h, val)
				}
			}
			fmt.Printf("================================\n\n")
		}

		// 3. Log strukturalny (Zap)
		log.Info("Request completed",
			zap.String("method", c.Method()),
			zap.String("path", c.Path()),
			zap.Int("status", status),
			zap.String("latency", latency.String()),
			zap.Any("request_id", requestID),
			zap.String("ip", c.IP()),
		)

		return err
	}
}
