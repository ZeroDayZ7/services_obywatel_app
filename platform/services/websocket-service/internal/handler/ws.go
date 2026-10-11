package handler

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/fasthttp/websocket"
	fiber "github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/valyala/fasthttp"
	"github.com/zerodayz7/platform/services/websocket-service/internal/hub"
)

type userClaims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

type Handler struct {
	hub          *hub.Hub
	sharedSecret string
}

func NewHandler(h *hub.Hub, sharedSecret string) *Handler {
	return &Handler{hub: h, sharedSecret: sharedSecret}
}

func (h *Handler) HandleWS(c fiber.Ctx) error {
	userID := strings.TrimSpace(c.Query("userId"))
	if userID == "" {
		userID = strings.TrimSpace(c.Query("user_id"))
	}
	if userID == "" {
		userID = strings.TrimSpace(c.Get("X-User-Id"))
	}

	token := strings.TrimSpace(c.Get("Authorization"))
	if token == "" {
		token = strings.TrimSpace(c.Query("token"))
	}
	if token != "" {
		token = strings.TrimPrefix(token, "Bearer ")
		parsedUserID, err := h.validateToken(token)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).SendString("invalid token")
		}
		if userID == "" {
			userID = parsedUserID
		}
	}

	if userID == "" {
		return c.Status(fiber.StatusUnauthorized).SendString("missing user_id")
	}

	upgrader := websocket.FastHTTPUpgrader{CheckOrigin: func(*fasthttp.RequestCtx) bool { return true }}
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

func (h *Handler) validateToken(token string) (string, error) {
	if h.sharedSecret == "" {
		return "", errors.New("missing shared secret")
	}
	parsed, err := jwt.ParseWithClaims(token, &userClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(h.sharedSecret), nil
	})
	if err != nil {
		return "", err
	}
	claims, ok := parsed.Claims.(*userClaims)
	if !ok || !parsed.Valid {
		return "", errors.New("invalid claims")
	}
	if claims.UserID == "" {
		return "", errors.New("missing user_id in token")
	}
	return claims.UserID, nil
}
