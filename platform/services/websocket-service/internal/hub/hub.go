package hub

import (
	"context"
	"encoding/json"
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
	mu      sync.RWMutex
	sessions map[string]map[*Session]struct{}
	redis   *redis.Client
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
	for _, clients := range h.sessions {
		for session := range clients {
			if session == nil {
				continue
			}
			select {
			case session.send <- payload:
			default:
				go h.UnregisterSession(session)
			}
		}
	}

	_ = channel
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
			if msg.Channel == "ws:events" {
				continue
			}
			if msg.Channel == "ws:internal" {
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
