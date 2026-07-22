package router

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/proxy"
	"github.com/zerodayz7/platform/pkg/constants"
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
			c.Request().Header.Set(constants.HeaderRequestID, ctx.RequestID)
			c.Request().Header.Set(constants.HeaderXForwardedFor, ctx.IP)
			c.Request().Header.Set(constants.HeaderXRealIP, ctx.IP)
		}

		// Wymusić przekazanie User-Agent i Fingerprint,
		// Fiber domyślnie przekazuje większość nagłówków klienta.

		// 4. Wykonujemy Proxy
		return proxy.Do(c, url)
	}
}

// W internal/router/proxy.go popraw ReverseProxy:
// region ReverseProxy
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
			req.Header.Set(constants.HeaderRequestID, ctx.RequestID)
			req.Header.Set(constants.HeaderXForwardedFor, ctx.IP)
			req.Header.Set(constants.HeaderXRealIP, ctx.IP)
			if ctx.DeviceID != "" {
				req.Header.Set(constants.HeaderDeviceFingerprint, ctx.DeviceID)
			}

			// PODPISYWANIE I PRZEKAZYWANIE KONTEKSTU DLA KAZDEGO ZĄDANIA
			payload, err := reqctx.Encode(*ctx)
			if err == nil {
				sig := reqctx.Sign(payload, container.InternalSecret)
				req.Header.Set(constants.HeaderInternalContext, base64.StdEncoding.EncodeToString(payload))
				req.Header.Set(constants.HeaderInternalSignature, sig)
			}
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
		req.Header.Set(constants.HeaderRequestID, ctx.RequestID)
		req.Header.Set(constants.HeaderXForwardedFor, ctx.IP)
		req.Header.Set(constants.HeaderXRealIP, ctx.IP)

		if ctx.UserID != nil {
			req.Header.Set("X-User-Id", ctx.UserID.String())
		}
		if ctx.SessionID != "" {
			req.Header.Set("X-Session-Id", ctx.SessionID)
		}

		if ctx.DeviceID != "" {
			req.Header.Set(constants.HeaderDeviceFingerprint, ctx.DeviceID)
			req.Header.Set("X-Device-Id", ctx.DeviceID)
		}

		// ---  Zero trust: auth-related ---
		req.Header.Del(constants.HeaderAuth)
		req.Header.Del(constants.HeaderCookie)

		// --- podpisany kontekst ---
		payload, err := reqctx.Encode(*ctx)
		if err != nil {
			log.ErrorObj("Failed to encode request context", err)
			return apperr.SendAppError(c, apperr.ErrInternal)
		}
		sig := reqctx.Sign(payload, container.InternalSecret)
		req.Header.Set(constants.HeaderInternalContext, base64.StdEncoding.EncodeToString(payload))
		req.Header.Set(constants.HeaderInternalSignature, sig)

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
			return apperr.SendAppError(c, apperr.ErrUpstreamTimeout)
		}
		log.ErrorObj("Upstream request failed", err)
		return apperr.SendAppError(c, apperr.ErrUpstreamUnreachable)
	}
	// UWAGA: Nie zamykamy resp.Body ręcznie przez defer, jeśli przekazujemy strumień do Fibera!
	// Fiber przejmuje odpowiedzialność za zamknięcie strumienia po zakończeniu wysyłania odpowiedzi do klienta.

	if resp.StatusCode == fiber.StatusForbidden {
		resp.Body.Close() // Tu zamykamy ręcznie, bo przerywamy przepływ błędem
		log.Error("Security Alert: Upstream rejected internal signature or context")
		return apperr.SendAppError(c, apperr.ErrInternal)
	}

	// Przepisywanie nagłówków (wykluczając Hop-by-Hop)
	for k, v := range resp.Header {
		if isHopByHop(k) {
			continue
		}
		for _, vv := range v {
			c.Set(k, vv)
		}
	}

	c.Status(resp.StatusCode)

	// Wysyłanie strumieniowe zero-copy (brak wycieków pamięci i alokacji buforów w pętli)
	c.Response().SetBodyStream(resp.Body, int(resp.ContentLength))
	return nil
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
