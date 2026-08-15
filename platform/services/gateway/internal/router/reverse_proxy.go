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
		ctx, _ := c.Locals("requestContext").(*reqctx.RequestContext)

		url := target + c.Path()

		if ctx != nil {
			c.Request().Header.Set(constants.HeaderRequestID, ctx.RequestID)
			c.Request().Header.Set(constants.HeaderXForwardedFor, ctx.IP)
			c.Request().Header.Set(constants.HeaderXRealIP, ctx.IP)
		}

		return proxy.Do(c, url)
	}
}

// region ReverseProxy
func ReverseProxy(container *di.Container, serviceID string, target string) fiber.Handler {
	log := shared.GetLogger()
	return func(c *fiber.Ctx) error {
		ctx, _ := c.Locals("requestContext").(*reqctx.RequestContext)

		req, err := prepareProxyRequest(c, target)
		if err != nil {
			return err
		}

		clientHeaders := []string{
			constants.HeaderContentType,
			constants.HeaderAccept,
			constants.HeaderUserAgent,
			constants.HeaderDeviceFingerprint,
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

			// Pobranie dedykowanego sekretu HMAC dla konkretnego serwisu
			hmacSecret, _, ok := container.GetHMACKey(serviceID)
			if !ok {
				log.Error("Missing HMAC secret for target service", "service_id", serviceID)
				return apperr.SendAppError(c, apperr.ErrInternal)
			}

			// PODPISYWANIE I PRZEKAZYWANIE KONTEKSTU DLA KAŻDEGO ŻĄDANIA
			payload, err := reqctx.Encode(*ctx)
			if err == nil {
				sig := reqctx.SignHMAC(payload, hmacSecret)
				req.Header.Set(constants.HeaderInternalContext, base64.StdEncoding.EncodeToString(payload))
				req.Header.Set(constants.HeaderInternalSignature, sig)
			}
		}

		return executeProxyRequest(c, container, req, log)
	}
}

func ReverseProxySecure(container *di.Container, serviceID string, target string) fiber.Handler {
	log := shared.GetLogger()

	return func(c *fiber.Ctx) error {
		// --- Pobieramy RequestContext (JEDYNE źródło prawdy) ---
		ctx, ok := c.Locals("requestContext").(*reqctx.RequestContext)
		if !ok || ctx == nil {
			log.Warn("Missing request context")
			return fiber.ErrUnauthorized
		}

		// --- Budujemy request do upstream ---
		req, err := prepareProxyRequest(c, target)
		if err != nil {
			return err
		}

		// --- Whitelist nagłówków z klienta (MINIMUM) ---
		clientHeaders := []string{
			constants.HeaderContentType,
			constants.HeaderAccept,
			constants.HeaderUserAgent,
			constants.HeaderDeviceFingerprint,
		}

		for _, h := range clientHeaders {
			if v := c.Get(h); v != "" {
				req.Header.Set(h, v)
			}
		}

		// --- Nagłówki kontrolowane ---
		req.Header.Set(constants.HeaderRequestID, ctx.RequestID)
		req.Header.Set(constants.HeaderXForwardedFor, ctx.IP)
		req.Header.Set(constants.HeaderXRealIP, ctx.IP)

		if ctx.UserID != nil {
			req.Header.Set(constants.HeaderUserID, ctx.UserID.String())
		}
		if ctx.SessionID != "" {
			req.Header.Set(constants.HeaderSessionID, ctx.SessionID)
		}

		if ctx.DeviceID != "" {
			req.Header.Set(constants.HeaderDeviceFingerprint, ctx.DeviceID)
			req.Header.Set(constants.HeaderDeviceID, ctx.DeviceID)
		}

		// --- Zero trust: auth-related ---
		req.Header.Del(constants.HeaderAuth)
		req.Header.Del(constants.HeaderCookie)

		// Pobranie dedykowanego sekretu HMAC dla konkretnego serwisu
		hmacSecret, _, ok := container.GetHMACKey(serviceID)
		if !ok {
			log.Error("Missing HMAC secret for target service", "service_id", serviceID)
			return apperr.SendAppError(c, apperr.ErrInternal)
		}

		// --- Podpisany kontekst ---
		payload, err := reqctx.Encode(*ctx)
		if err != nil {
			log.ErrorObj("Failed to encode request context", err)
			return apperr.SendAppError(c, apperr.ErrInternal)
		}
		sig := reqctx.SignHMAC(payload, hmacSecret)
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

	if resp.StatusCode == fiber.StatusForbidden {
		resp.Body.Close()
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

	// Wysyłanie strumieniowe zero-copy
	c.Response().SetBodyStream(resp.Body, int(resp.ContentLength))
	return nil
}

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
