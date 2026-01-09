package router

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/gateway/internal/di"
)

// W internal/router/proxy.go popraw ReverseProxy:
func ReverseProxy(container *di.Container, target string) fiber.Handler {
	log := shared.GetLogger()
	return func(c *fiber.Ctx) error {
		req, err := prepareProxyRequest(c, target)
		if err != nil {
			return err
		}

		// JAWNIE kopiujemy fingerprint, bo to nasz krytyczny nagłówek bezpieczeństwa
		fpt := c.Get("X-Device-Fingerprint")
		if fpt == "" {
			// Tutaj możesz nawet zdecydować o odrzuceniu requestu,
			// jeśli Twój system WYMAGA urządzenia we Flutterze
			log.Warn("Request without device fingerprint")
		}
		req.Header.Set("X-Device-Fingerprint", fpt)

		// // Kopiujemy resztę standardowych nagłówków
		// req.Header.Set("Content-Type", c.Get("Content-Type"))
		// req.Header.Set("User-Agent", c.Get("User-Agent"))

		allowedHeaders := []string{
			"Content-Type",
			"User-Agent",
		}
		for _, h := range allowedHeaders {
			if val := c.Get(h); val != "" {
				req.Header.Set(h, val)
			}
		}

		return executeProxyRequest(c, container, req, log)
	}
}

// ReverseProxySecure - przekazuje request, dodaje ID użytkownika i usuwa Auth
func ReverseProxySecure(container *di.Container, target string) fiber.Handler {
	log := shared.GetLogger()
	return func(c *fiber.Ctx) error {
		req, err := prepareProxyRequest(c, target)
		if err != nil {
			return err
		}

		// Kopiujemy nagłówki (Content-Type itp.)
		if ct := c.Get("Content-Type"); ct != "" {
			req.Header.Set("Content-Type", ct)
		}
		req.Header.Set("Accept", c.Get("Accept", "*/*"))

		// Wstrzykujemy tożsamość użytkownika z contextu (Locals)
		if uid := c.Locals("userID"); uid != nil {
			req.Header.Set("X-User-Id", fmt.Sprintf("%v", uid))
		}
		if sid := c.Locals("sessionID"); sid != nil {
			req.Header.Set("X-Session-Id", fmt.Sprintf("%v", sid))
		}

		userIP := c.IP()
		req.Header.Set("X-Forwarded-For", userIP)
		req.Header.Set("X-Real-IP", userIP)

		// Ważne: usuwamy token, backend ufa nagłówkowi X-User-Id
		req.Header.Del("Authorization")

		return executeProxyRequest(c, container, req, log)
	}
}

// --- FUNKCJE POMOCNICZE (DRY) ---

func prepareProxyRequest(c *fiber.Ctx, target string) (*http.Request, error) {

	body := c.Body()

	url := target + c.OriginalURL()

	req, err := http.NewRequestWithContext(
		c.UserContext(),
		string(c.Method()),
		url,
		bytes.NewReader(body),
	)

	if err != nil {
		return nil, err
	}

	return req, nil
}

func executeProxyRequest(c *fiber.Ctx, container *di.Container, req *http.Request, log *shared.Logger) error {
	resp, err := container.HTTPClient.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return c.Status(fiber.StatusGatewayTimeout).JSON(fiber.Map{
				"error": "Upstream service timeout",
			})
		}

		log.ErrorObj("Upstream request failed", err)

		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"error": "Upstream service unreachable",
		})
	}
	defer resp.Body.Close()

	for k, v := range resp.Header {
		if isHopByHop(k) {
			continue
		}
		for _, vv := range v {
			c.Set(k, vv)
		}
	}

	// 3. Przekazanie statusu i body
	c.Status(resp.StatusCode)

	// io.Copy przesyła dane kawałek po kawałku (stream),
	// co jest świetne dla pamięci RAM przy dużych odpowiedziach.
	_, err = io.Copy(c.Response().BodyWriter(), resp.Body)
	return err
}

// Pomocnicza funkcja do filtrowania nagłówków technicznych
func isHopByHop(header string) bool {
	headers := map[string]bool{
		"Connection":          true,
		"Keep-Alive":          true,
		"Proxy-Authenticate":  true,
		"Proxy-Authorization": true,
		"Te":                  true,
		"Trailers":            true,
		"Transfer-Encoding":   true,
		"Upgrade":             true,
	}
	return headers[http.CanonicalHeaderKey(header)]
}
