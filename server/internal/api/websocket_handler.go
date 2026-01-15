package api

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"stackyn/server/internal/services"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for now - in production, you should validate this
		return true
	},
}

// WebSocketHandler handles WebSocket connections for build status updates
type WebSocketHandler struct {
	hub    *services.Hub
	logger *zap.Logger
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(hub *services.Hub, logger *zap.Logger) *WebSocketHandler {
	return &WebSocketHandler{
		hub:    hub,
		logger: logger,
	}
}

// HandleWebSocket handles WebSocket connections for app build status
func (h *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Get app ID from query parameter
	appID := r.URL.Query().Get("app_id")
	if appID == "" {
		http.Error(w, "app_id query parameter is required", http.StatusBadRequest)
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		// Failed to upgrade connection to WebSocket - no logging
		return
	}

	// Create client
	client := &services.Client{
		ID:     uuid.New().String(),
		AppID:  appID,
		Send:   make(chan []byte, 256),
		Hub:    h.hub,
		Logger: h.logger,
		Conn:   conn,
	}

	// Register client
	h.hub.RegisterClient(client)

	// Start goroutines for reading and writing
	go client.WritePump()
	go client.ReadPump()
}

