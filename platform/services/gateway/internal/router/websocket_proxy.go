package router

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/fasthttp/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"
	"github.com/zerodayz7/platform/pkg/constants"
	reqctx "github.com/zerodayz7/platform/pkg/context"
	apperr "github.com/zerodayz7/platform/pkg/errors"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/gateway/internal/di"
)

func websocketProxy(container *di.Container, target string) fiber.Handler {
	log := shared.GetLogger()

	return func(c *fiber.Ctx) error {
		ctx, ok := c.Locals("requestContext").(*reqctx.RequestContext)
		if !ok || ctx == nil || ctx.UserID == nil {
			return fiber.ErrUnauthorized
		}

		hmacSecret, _, ok := container.GetHMACKey("websocket-service")
		if !ok {
			log.Error(
				"Missing HMAC secret for websocket-service",
				"service_id", "websocket-service",
			)
			return apperr.SendAppError(c, apperr.ErrInternal)
		}

		payload, err := reqctx.Encode(*ctx)
		if err != nil {
			log.ErrorObj(
				"Failed to encode request context for websocket proxy",
				err,
			)
			return apperr.SendAppError(c, apperr.ErrInternal)
		}

		upstreamURL := buildUpstreamWebsocketURL(
			target,
			c.OriginalURL(),
		)

		upgrader := websocket.FastHTTPUpgrader{
			HandshakeTimeout: 10 * time.Second,
			CheckOrigin: func(r *fasthttp.RequestCtx) bool {
				origin := strings.TrimSpace(
					string(r.Request.Header.Peek("Origin")),
				)

				if origin == "" {
					return true
				}

				parsed, err := url.Parse(origin)
				if err != nil {
					return false
				}

				return strings.EqualFold(
					parsed.Host,
					string(r.Host()),
				)
			},
		}

		return upgrader.Upgrade(c.Context(), func(clientConn *websocket.Conn) {
			headers := make(map[string][]string)

			headers[constants.HeaderUserID] = []string{
				ctx.UserID.String(),
			}
			headers[constants.HeaderInternalContext] = []string{
				base64.StdEncoding.EncodeToString(payload),
			}
			headers[constants.HeaderInternalSignature] = []string{
				reqctx.SignHMAC(payload, hmacSecret),
			}

			if ctx.SessionID != nil {
				headers[constants.HeaderSessionID] = []string{
					ctx.SessionID.String(),
				}
			}

			if ctx.RequestID != "" {
				headers[constants.HeaderRequestID] = []string{
					ctx.RequestID,
				}
			}

			if ctx.Fingerprint != "" {
				headers[constants.HeaderDeviceFingerprint] = []string{
					ctx.Fingerprint,
				}
			}

			remoteConn, _, err := websocket.DefaultDialer.Dial(
				upstreamURL,
				headers,
			)
			if err != nil {
				_ = clientConn.WriteMessage(
					websocket.CloseMessage,
					websocket.FormatCloseMessage(
						websocket.ClosePolicyViolation,
						fmt.Sprintf(
							"upstream websocket unavailable: %v",
							err,
						),
					),
				)
				return
			}

			defer func() {
				_ = remoteConn.Close()
				_ = clientConn.Close()
			}()

			go copyWebSocketFrames(clientConn, remoteConn)
			copyWebSocketFrames(remoteConn, clientConn)
		})
	}
}

func buildUpstreamWebsocketURL(
	baseURL string,
	requestPath string,
) string {
	trimmed := strings.TrimRight(baseURL, "/")

	if requestPath == "" || requestPath == "/" {
		return trimmed
	}

	path := requestPath
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	return trimmed + path
}

func copyWebSocketFrames(
	dst *websocket.Conn,
	src *websocket.Conn,
) {
	for {
		messageType, message, err := src.ReadMessage()
		if err != nil {
			return
		}

		if err := dst.WriteMessage(messageType, message); err != nil {
			return
		}
	}
}
