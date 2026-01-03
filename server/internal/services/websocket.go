package services

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512
)

// BuildStatusUpdate represents a build status update message
type BuildStatusUpdate struct {
	Type      string                 `json:"type"`      // "build_status"
	AppID     string                 `json:"app_id"`
	BuildJobID string                `json:"build_job_id"`
	Status    string                 `json:"status"`    // "building", "completed", "failed"
	Progress  *BuildProgress         `json:"progress,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// BuildProgress represents build progress information
type BuildProgress struct {
	Stage   string `json:"stage"`   // "cloning", "detecting", "building", "deploying"
	Message string `json:"message"` // Human-readable progress message
	Percent int    `json:"percent"` // 0-100
}

// Client represents a WebSocket client connection
type Client struct {
	ID     string
	AppID  string // The app ID this client is subscribed to
	Send   chan []byte
	Hub    *Hub
	Logger *zap.Logger
	Conn   *websocket.Conn
}

// Hub maintains the set of active clients and broadcasts messages to them
type Hub struct {
	// Registered clients by app ID
	clients map[string]map[*Client]bool

	// Inbound messages from the clients
	broadcast chan []byte

	// Register requests from the clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Mutex for thread-safe access
	mu sync.RWMutex

	logger *zap.Logger
}

// NewHub creates a new Hub instance
func NewHub(logger *zap.Logger) *Hub {
	return &Hub{
		clients:    make(map[string]map[*Client]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		logger:     logger,
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if h.clients[client.AppID] == nil {
				h.clients[client.AppID] = make(map[*Client]bool)
			}
			h.clients[client.AppID][client] = true
			h.mu.Unlock()
			// WebSocket client registered - no logging

		case client := <-h.unregister:
			h.mu.Lock()
			if clients, ok := h.clients[client.AppID]; ok {
				if _, ok := clients[client]; ok {
					delete(clients, client)
					close(client.Send)
					if len(clients) == 0 {
						delete(h.clients, client.AppID)
					}
				}
			}
			h.mu.Unlock()
			// WebSocket client unregistered - no logging

		case message := <-h.broadcast:
			// Parse message to determine which app ID to broadcast to
			var update BuildStatusUpdate
			if err := json.Unmarshal(message, &update); err != nil {
				// Failed to unmarshal broadcast message - no logging
				continue
			}

			h.mu.RLock()
			clients, ok := h.clients[update.AppID]
			if !ok {
				h.mu.RUnlock()
				continue
			}
			// Create a copy of clients to iterate safely
			clientList := make([]*Client, 0, len(clients))
			for client := range clients {
				clientList = append(clientList, client)
			}
			h.mu.RUnlock()

			// Send message to all clients subscribed to this app
			for _, client := range clientList {
				select {
				case client.Send <- message:
				default:
					// Client's send buffer is full, remove client
					h.mu.Lock()
					if clients, ok := h.clients[client.AppID]; ok {
						if _, ok := clients[client]; ok {
							delete(clients, client)
							close(client.Send)
							if len(clients) == 0 {
								delete(h.clients, client.AppID)
							}
						}
					}
					h.mu.Unlock()
				}
			}
		}
	}
}

// BroadcastBuildStatus broadcasts a build status update to all clients subscribed to the app
func (h *Hub) BroadcastBuildStatus(update BuildStatusUpdate) {
	message, err := json.Marshal(update)
	if err != nil {
		// Failed to marshal build status update - no logging
		return
	}
	
	select {
	case h.broadcast <- message:
	default:
		// Broadcast channel is full, dropping message - no logging
	}
}

// RegisterClient registers a client with the hub
func (h *Hub) RegisterClient(client *Client) {
	h.register <- client
}

// ReadPump pumps messages from the websocket connection to the hub
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.unregister <- c
		if c.Conn != nil {
			c.Conn.Close()
		}
	}()

	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

		for {
			_, _, err := c.Conn.ReadMessage()
			if err != nil {
				// WebSocket read error - no logging
				break
			}
		}
}

// WritePump pumps messages from the hub to the websocket connection
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		if c.Conn != nil {
			c.Conn.Close()
		}
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel
				if c.Conn != nil {
					c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				}
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued chat messages to the current websocket message
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

