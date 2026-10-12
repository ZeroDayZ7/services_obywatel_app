package handler

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/fasthttp/websocket"
	fiber "github.com/gofiber/fiber/v3"
	"github.com/valyala/fasthttp"
	reqctx "github.com/zerodayz7/platform/pkg/context"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/websocket-service/internal/hub"
)

const (
	eventChannel = "ws:events"
	pingMessage  = "ping"
	pongMessage  = "pong"
)

type Handler struct {
	hub *hub.Hub
}

func NewHandler(h *hub.Hub) *Handler {
	return &Handler{hub: h}
}

func (h *Handler) HandleWS(c fiber.Ctx) error {
	log := shared.GetLogger()

	reqCtx, ok := c.Locals(reqctx.FiberRequestContextKey).(*reqctx.RequestContext)
	if !ok || reqCtx == nil || reqCtx.UserID == nil {
		log.Warn("HandleWS: missing trusted request context or user ID", "path", c.Path())
		return c.Status(fiber.StatusUnauthorized).SendString("missing trusted request context")
	}
	userID := reqCtx.UserID.String()
	log.Info("HandleWS: attempting websocket upgrade for user", "user_id", userID)

	upgrader := websocket.FastHTTPUpgrader{
		CheckOrigin: func(r *fasthttp.RequestCtx) bool {
			allowed := isAllowedOrigin(r)
			if !allowed {
				log.Warn("HandleWS: origin rejected", "origin", string(r.Request.Header.Peek("Origin")), "host", string(r.Host()))
			}
			return allowed
		},
	}

	return upgrader.Upgrade(c.RequestCtx(), func(conn *websocket.Conn) {
		log.Info("HandleWS: websocket connection established", "user_id", userID)
		session := h.hub.RegisterSession(userID, conn)
		defer func() {
			log.Info("HandleWS: websocket session unregistered / closed", "user_id", userID)
			h.hub.UnregisterSession(session)
		}()

		for {
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				log.Warn("HandleWS: client disconnected or read error", "user_id", userID, "error", err)
				return
			}

			if messageType != websocket.TextMessage && messageType != websocket.BinaryMessage {
				continue
			}

			msgStr := string(message)

			// Obsługa heartbeat (ping/pong) - pomijamy Redisa dla optymalizacji
			if msgStr == pingMessage {
				if err := conn.WriteMessage(websocket.TextMessage, []byte(pongMessage)); err != nil {
					log.Warn("HandleWS: failed to write pong response", "user_id", userID, "error", err)
					return
				}
				continue
			}

			log.Debug("HandleWS: ignoring client business payload; backend owns websocket event routing", "user_id", userID, "type", messageType, "size", len(message))
		}
	})
}

func (h *Handler) HandleInternalBroadcast(c fiber.Ctx) error {
	log := shared.GetLogger()

	if _, ok := c.Locals(reqctx.FiberRequestContextKey).(*reqctx.RequestContext); !ok {
		log.Warn("HandleInternalBroadcast: missing trusted request context")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing trusted request context"})
	}

	var req struct {
		UserID  string `json:"user_id"`
		Channel string `json:"channel"`
		Message string `json:"message"`
	}
	if err := c.Bind().Body(&req); err != nil {
		log.Warn("HandleInternalBroadcast: invalid request body", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	log.Info("HandleInternalBroadcast: processing request", "user_id", req.UserID, "channel", req.Channel)

	payload := []byte(req.Message)
	if req.UserID != "" {
		h.hub.BroadcastToUser(req.UserID, payload)
	}
	if req.Channel != "" {
		h.hub.BroadcastToChannel(req.Channel, payload)
	}

	return c.JSON(fiber.Map{"ok": true, "user_id": req.UserID, "channel": req.Channel})
}

func trustedUserIDFromSignedContext(encodedCtx, signature, secret, fallbackUserID string) (string, error) {
	_ = fallbackUserID
	if strings.TrimSpace(secret) == "" {
		return "", errors.New("missing shared secret")
	}
	if strings.TrimSpace(encodedCtx) == "" || strings.TrimSpace(signature) == "" {
		return "", errors.New("missing signed request context")
	}

	payload, err := base64.StdEncoding.DecodeString(encodedCtx)
	if err != nil {
		return "", fmt.Errorf("decode signed context: %w", err)
	}
	if !reqctx.VerifyHMAC(payload, signature, []byte(secret)) {
		return "", errors.New("invalid signed request context")
	}

	ctx, err := reqctx.Decode(payload)
	if err != nil {
		return "", fmt.Errorf("decode request context: %w", err)
	}
	if ctx == nil || ctx.UserID == nil {
		return "", errors.New("missing user_id in trusted context")
	}
	return ctx.UserID.String(), nil
}

func isAllowedOrigin(r *fasthttp.RequestCtx) bool {
	origin := strings.TrimSpace(string(r.Request.Header.Peek("Origin")))
	if origin == "" {
		return true
	}
	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return strings.EqualFold(parsed.Host, string(r.Host()))
}
