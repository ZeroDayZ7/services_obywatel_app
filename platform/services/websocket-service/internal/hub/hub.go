package hub

import (
	"context"
	"encoding/json"
	"strings"
	"sync"

	"github.com/fasthttp/websocket"
	"github.com/redis/go-redis/v9"
)

type Session struct {
	UserID string
	Conn   *websocket.Conn
	send   chan []byte
	done   chan struct{}
}

type Hub struct {
	mu       sync.RWMutex
	sessions map[string]map[*Session]struct{}
	redis    *redis.Client
}

func NewHub(redisClient *redis.Client) *Hub {
	return &Hub{
		sessions: make(map[string]map[*Session]struct{}),
		redis:    redisClient,
	}
}

func (h *Hub) RegisterSession(userID string, conn *websocket.Conn) *Session {
	session := &Session{
		UserID: userID,
		Conn:   conn,
		send:   make(chan []byte, 32),
		done:   make(chan struct{}),
	}

	h.mu.Lock()
	if _, exists := h.sessions[userID]; !exists {
		h.sessions[userID] = make(map[*Session]struct{})
	}
	h.sessions[userID][session] = struct{}{}
	h.mu.Unlock()

	go session.writePump()
	return session
}

func (h *Hub) UnregisterSession(session *Session) {
	if session == nil {
		return
	}

	h.mu.Lock()
	delete(h.sessions[session.UserID], session)
	if len(h.sessions[session.UserID]) == 0 {
		delete(h.sessions, session.UserID)
	}
	h.mu.Unlock()

	close(session.done)
	close(session.send)
	_ = session.Conn.Close()
}

func (h *Hub) BroadcastToUser(userID string, payload []byte) {
	h.mu.RLock()
	clients, ok := h.sessions[userID]
	h.mu.RUnlock()
	if !ok {
		return
	}

	for session := range clients {
		select {
		case session.send <- payload:
		default:
			_ = session.Conn.WriteMessage(websocket.CloseMessage, []byte("channel full"))
			go h.UnregisterSession(session)
		}
	}
}

func (h *Hub) BroadcastToChannel(channel string, payload []byte) {
	if strings.TrimSpace(channel) == "" {
		return
	}

	if strings.HasPrefix(channel, "ws:user:") {
		userID := strings.TrimPrefix(channel, "ws:user:")
		h.BroadcastToUser(userID, payload)
		return
	}

	if strings.HasPrefix(channel, "ws:conversation:") {
		// Konwersacyjne kanały są prywatne i nie wolno rozsyłać do wszystkich sesji.
		return
	}
}

func (h *Hub) PublishEvent(channel string, payload any) error {
	if h.redis == nil {
		return nil
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return h.redis.Publish(context.Background(), channel, body).Err()
}

func (h *Hub) BroadcastToUsers(userIDs []string, payload []byte) {
	if len(userIDs) == 0 {
		return
	}
	seen := make(map[string]struct{}, len(userIDs))
	for _, userID := range userIDs {
		if strings.TrimSpace(userID) == "" {
			continue
		}
		if _, exists := seen[userID]; exists {
			continue
		}
		seen[userID] = struct{}{}
		h.BroadcastToUser(userID, payload)
	}
}

func (h *Hub) handleRedisEvent(payload []byte) {
	if len(payload) == 0 {
		return
	}

	var event map[string]any
	if err := json.Unmarshal(payload, &event); err != nil {
		return
	}

	body, err := json.Marshal(event)
	if err != nil {
		return
	}

	if targetUserIDs, ok := event["targetUserIds"].([]any); ok {
		users := make([]string, 0, len(targetUserIDs))
		for _, item := range targetUserIDs {
			if userID, ok := item.(string); ok && strings.TrimSpace(userID) != "" {
				users = append(users, userID)
			}
		}
		if len(users) > 0 {
			h.BroadcastToUsers(users, body)
			return
		}
	}

	if userID, ok := event["userId"].(string); ok && strings.TrimSpace(userID) != "" {
		h.BroadcastToUser(userID, body)
	}
}

func (h *Hub) RunRedisSubscriber(ctx context.Context) {
	if h.redis == nil {
		return
	}

	pubsub := h.redis.PSubscribe(ctx, "ws:*")
	defer func() { _ = pubsub.Close() }()

	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			if msg == nil || len(msg.Payload) == 0 {
				continue
			}
			if msg.Channel == "ws:internal" {
				continue
			}
			if msg.Channel == "ws:events" {
				h.handleRedisEvent([]byte(msg.Payload))
				continue
			}
			h.BroadcastToChannel(msg.Channel, []byte(msg.Payload))
		}
	}
}

func (s *Session) writePump() {
	for {
		select {
		case msg, ok := <-s.send:
			if !ok {
				return
			}
			if err := s.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-s.done:
			return
		}
	}
}
