package router

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/proxy"
	reqctx "github.com/zerodayz7/platform/pkg/context"
	apperr "github.com/zerodayz7/platform/pkg/errors"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/gateway/internal/di"
)

func ReverseProxyFiber(container *di.Container, target string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 1. Pobieramy Twój kontekst (ID requestu itp.)
		ctx, _ := c.Locals("requestContext").(*reqctx.RequestContext)

		// 2. Przygotowujemy URL docelowy (np. http://auth-service:8080/auth/login)
		// c.Path() zawiera pełną ścieżkę
		url := target + c.Path()

		// 3. Dodajemy Twoje customowe nagłówki przed wysłaniem
		if ctx != nil {
			c.Request().Header.Set("X-Request-Id", ctx.RequestID)
			c.Request().Header.Set("X-Forwarded-For", ctx.IP)
			c.Request().Header.Set("X-Real-IP", ctx.IP)
		}

		// Wymusić przekazanie User-Agent i Fingerprint,
		// Fiber domyślnie przekazuje większość nagłówków klienta.

		// 4. Wykonujemy Proxy
		return proxy.Do(c, url)
	}
}

// W internal/router/proxy.go popraw ReverseProxy:
func ReverseProxy(container *di.Container, target string) fiber.Handler {
	log := shared.GetLogger()
	return func(c *fiber.Ctx) error {
		ctx, _ := c.Locals("requestContext").(*reqctx.RequestContext)

		req, err := prepareProxyRequest(c, target)
		if err != nil {
			return err
		}

		clientHeaders := []string{
			"Content-Type",
			"Accept",
			"User-Agent",
			"X-Device-Fingerprint",
		}

		for _, h := range clientHeaders {
			if v := c.Get(h); v != "" {
				req.Header.Set(h, v)
			}
		}

		if ctx != nil {
			req.Header.Set("X-Request-Id", ctx.RequestID)
			req.Header.Set("X-Forwarded-For", ctx.IP)
			req.Header.Set("X-Real-IP", ctx.IP)
		}

		return executeProxyRequest(c, container, req, log)
	}
}

func ReverseProxySecure(container *di.Container, target string) fiber.Handler {
	log := shared.GetLogger()

	return func(c *fiber.Ctx) error {
		// --- Pobieramy RequestContext (JEDYNE źródło prawdy) ---
		ctx, ok := c.Locals("requestContext").(*reqctx.RequestContext)
		if !ok || ctx == nil {
			log.Warn("Missing request context")
			return fiber.ErrUnauthorized
		}

		// ---  Budujemy request do upstream ---
		req, err := prepareProxyRequest(c, target)
		if err != nil {
			return err
		}

		// ---  Whitelist nagłówków z klienta (MINIMUM) ---
		clientHeaders := []string{
			"Content-Type",
			"Accept",
			"User-Agent",
			"X-Device-Fingerprint",
		}

		for _, h := range clientHeaders {
			if v := c.Get(h); v != "" {
				req.Header.Set(h, v)
			}
		}

		// --- Nagłówki kontrolowane  ---
		req.Header.Set("X-Request-Id", ctx.RequestID)
		req.Header.Set("X-Forwarded-For", ctx.IP)
		req.Header.Set("X-Real-IP", ctx.IP)

		if ctx.UserID != nil {
			req.Header.Set("X-User-Id", ctx.UserID.String())
		}
		if ctx.SessionID != "" {
			req.Header.Set("X-Session-Id", ctx.SessionID)
		}

		if ctx.DeviceID != "" {
			req.Header.Set("X-Device-Id", ctx.DeviceID)
		}

		// ---  Zero trust: auth-related ---
		req.Header.Del("Authorization")
		req.Header.Del("Cookie")

		// --- podpisany kontekst ---
		payload, err := reqctx.Encode(*ctx)
		if err != nil {
			log.ErrorObj("Failed to encode request context", err)
			return apperr.SendAppError(c, apperr.ErrInternal)
		}
		sig := reqctx.Sign(payload, container.InternalSecret)
		req.Header.Set("X-Internal-Context", base64.StdEncoding.EncodeToString(payload))
		req.Header.Set("X-Internal-Signature", sig)

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
