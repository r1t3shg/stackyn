package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// WebSocketBroadcastClient is a client for broadcasting build status updates to the API server
type WebSocketBroadcastClient struct {
	apiServerURL string
	httpClient   *http.Client
	logger       *zap.Logger
}

// NewWebSocketBroadcastClient creates a new WebSocket broadcast client
// apiServerURL should be the base URL of the API server (e.g., "http://localhost:8080")
func NewWebSocketBroadcastClient(apiServerURL string, logger *zap.Logger) *WebSocketBroadcastClient {
	if apiServerURL == "" {
		apiServerURL = "http://localhost:8080" // Default to localhost
	}

	return &WebSocketBroadcastClient{
		apiServerURL: apiServerURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		logger: logger,
	}
}

// BroadcastStatus sends a build status update to the API server for WebSocket broadcasting
func (c *WebSocketBroadcastClient) BroadcastStatus(ctx context.Context, update BuildStatusUpdate) error {
	url := fmt.Sprintf("%s/api/internal/build-status", c.apiServerURL)

	body, err := json.Marshal(update)
	if err != nil {
		return fmt.Errorf("failed to marshal update: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Warn("Failed to broadcast build status", zap.Error(err))
		return fmt.Errorf("failed to send broadcast request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.logger.Warn("Broadcast endpoint returned non-OK status",
			zap.Int("status_code", resp.StatusCode),
		)
		return fmt.Errorf("broadcast endpoint returned status %d", resp.StatusCode)
	}

	return nil
}

