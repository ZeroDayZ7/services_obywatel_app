package websocket

import (
	"sync"

	"github.com/zerodayz7/platform/pkg/shared"
)

type Message struct {
	UserID  string
	Payload any
}

type Hub struct {
	clients    map[string]map[*Client]bool
	register   chan *Client
	unregister chan *Client
	Broadcast  chan Message
	mu         sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		Broadcast:  make(chan Message, 256),
	}
}

func (h *Hub) RegisterClient(c *Client) {
	h.register <- c
}

func (h *Hub) UnregisterClient(c *Client) {
	h.unregister <- c
}

func (h *Hub) Run() {
	log := shared.GetLogger()

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if h.clients[client.UserID] == nil {
				h.clients[client.UserID] = make(map[*Client]bool)
			}
			h.clients[client.UserID][client] = true
			total := len(h.clients[client.UserID])
			h.mu.Unlock()

			log.Debug("[WS Hub] Zarejestrowano klienta", "userID", client.UserID, "connections", total)

		case client := <-h.unregister:
			h.mu.Lock()
			if clients, ok := h.clients[client.UserID]; ok {
				if _, exists := clients[client]; exists {
					delete(clients, client)
					close(client.Send)
					if len(clients) == 0 {
						delete(h.clients, client.UserID)
					}
				}
			}
			h.mu.Unlock()

			log.Debug("[WS Hub] Wyrejestrowano klienta", "userID", client.UserID)

		case msg := <-h.Broadcast:
			h.mu.RLock()
			if clients, ok := h.clients[msg.UserID]; ok {
				for client := range clients {
					select {
					case client.Send <- msg.Payload:
					default:
						go h.UnregisterClient(client)
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) BroadcastAll(payload any) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, clients := range h.clients {
		for client := range clients {
			select {
			case client.Send <- payload:
			default:
				go h.UnregisterClient(client)
			}
		}
	}
}
