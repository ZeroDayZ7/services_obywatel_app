package handler

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/fasthttp/websocket"
	fiber "github.com/gofiber/fiber/v3"
	"github.com/valyala/fasthttp"
	"github.com/zerodayz7/platform/pkg/constants"
	reqctx "github.com/zerodayz7/platform/pkg/context"
	"github.com/zerodayz7/platform/services/websocket-service/internal/hub"
)

type Handler struct {
	hub          *hub.Hub
	sharedSecret string
}

func NewHandler(h *hub.Hub, sharedSecret string) *Handler {
	return &Handler{hub: h, sharedSecret: sharedSecret}
}

func (h *Handler) HandleWS(c fiber.Ctx) error {
	userID, err := h.trustedUserIDFromRequest(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString(err.Error())
	}

	upgrader := websocket.FastHTTPUpgrader{CheckOrigin: func(r *fasthttp.RequestCtx) bool {
		return isAllowedOrigin(r)
	}}
	return upgrader.Upgrade(c.RequestCtx(), func(conn *websocket.Conn) {
		session := h.hub.RegisterSession(userID, conn)
		defer h.hub.UnregisterSession(session)

		for {
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if messageType == websocket.TextMessage || messageType == websocket.BinaryMessage {
				if err := h.hub.PublishEvent("ws:events", map[string]any{
					"user_id": userID,
					"payload": string(message),
					"ts":      time.Now().Unix(),
				}); err != nil {
					_ = conn.WriteMessage(websocket.TextMessage, fmt.Appendf(nil, "event publish failed: %v", err))
				}
			}
		}
	})
}

func (h *Handler) HandleInternalBroadcast(c fiber.Ctx) error {
	providedKey := strings.TrimSpace(c.Get("X-Internal-Key"))
	if providedKey == "" || providedKey != h.sharedSecret {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid internal key"})
	}

	var req struct {
		UserID  string `json:"user_id"`
		Channel string `json:"channel"`
		Message string `json:"message"`
	}
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	payload := []byte(req.Message)
	if req.UserID != "" {
		h.hub.BroadcastToUser(req.UserID, payload)
	}
	if req.Channel != "" {
		h.hub.BroadcastToChannel(req.Channel, payload)
	}

	return c.JSON(fiber.Map{"ok": true, "user_id": req.UserID, "channel": req.Channel})
}

func (h *Handler) trustedUserIDFromRequest(c fiber.Ctx) (string, error) {
	encodedCtx := strings.TrimSpace(c.Get(constants.HeaderInternalContext))
	signature := strings.TrimSpace(c.Get(constants.HeaderInternalSignature))
	if encodedCtx == "" || signature == "" {
		return "", errors.New("missing trusted request context")
	}
	return trustedUserIDFromSignedContext(encodedCtx, signature, h.sharedSecret, strings.TrimSpace(c.Get(constants.HeaderUserID)))
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
