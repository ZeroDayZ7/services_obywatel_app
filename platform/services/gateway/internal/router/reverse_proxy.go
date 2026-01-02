package router

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/services/gateway/internal/di"
)

// ReverseProxy - podstawowe przekierowanie z użyciem klienta z DI
func ReverseProxy(container *di.Container, target string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		req, err := prepareProxyRequest(c, target)
		if err != nil {
			return err
		}

		// Kopiujemy wszystkie nagłówki od klienta
		c.Request().Header.VisitAll(func(key, value []byte) {
			req.Header.Set(string(key), string(value))
		})

		return executeProxyRequest(c, container, req)
	}
}

// ReverseProxySecure - przekazuje request, dodaje ID użytkownika i usuwa Auth
func ReverseProxySecure(container *di.Container, target string) fiber.Handler {
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

		// Ważne: usuwamy token, backend ufa nagłówkowi X-User-Id
		req.Header.Del("Authorization")

		return executeProxyRequest(c, container, req)
	}
}

// --- FUNKCJE POMOCNICZE (DRY) ---

func prepareProxyRequest(c *fiber.Ctx, target string) (*http.Request, error) {
	body := c.Body()
	// Sklejamy bazowy adres mikroserwisu z oryginalną ścieżką requestu
	url := target + c.OriginalURL()

	return http.NewRequest(string(c.Method()), url, bytes.NewReader(body))
}

func executeProxyRequest(c *fiber.Ctx, container *di.Container, req *http.Request) error {
	// UŻYWAMY WSPÓŁDZIELONEGO KLIENTA Z KONTENERA
	resp, err := container.HTTPClient.Do(req)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"error": "Upstream service unreachable",
		})
	}
	defer resp.Body.Close()

	// Kopiujemy nagłówki odpowiedzi z mikroserwisu do klienta (Fluttera)
	for k, v := range resp.Header {
		for _, vv := range v {
			c.Set(k, vv)
		}
	}

	c.Status(resp.StatusCode)
	_, err = io.Copy(c, resp.Body)
	return err
}
