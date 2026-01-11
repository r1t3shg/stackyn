package api

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"go.uber.org/zap"
	"stackyn/server/internal/services"
)

// WebhookHandlers handles webhook requests from external services (e.g., Lemon Squeezy)
type WebhookHandlers struct {
	logger              *zap.Logger
	subscriptionService *services.SubscriptionService
	userRepo            UserRepository
	webhookSecret       string // Lemon Squeezy webhook signing secret
}

// NewWebhookHandlers creates a new webhook handlers instance
func NewWebhookHandlers(logger *zap.Logger, subscriptionService *services.SubscriptionService, userRepo UserRepository, webhookSecret string) *WebhookHandlers {
	return &WebhookHandlers{
		logger:              logger,
		subscriptionService: subscriptionService,
		userRepo:            userRepo,
		webhookSecret:       webhookSecret,
	}
}

// LemonSqueezyWebhook handles webhook events from Lemon Squeezy
// POST /api/webhooks/lemon-squeezy
func (h *WebhookHandlers) LemonSqueezyWebhook(w http.ResponseWriter, r *http.Request) {
	// Read raw body for signature verification
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("Failed to read webhook body", zap.Error(err))
		h.writeError(w, http.StatusBadRequest, "Failed to read request body")
		return
	}

	// Verify webhook signature (Lemon Squeezy sends signature in X-Signature header)
	signature := r.Header.Get("X-Signature")
	if signature == "" {
		h.logger.Warn("Webhook request missing signature")
		h.writeError(w, http.StatusUnauthorized, "Missing signature")
		return
	}

	// Verify signature
	if !h.verifyLemonSqueezySignature(body, signature) {
		h.logger.Warn("Invalid webhook signature")
		h.writeError(w, http.StatusUnauthorized, "Invalid signature")
		return
	}

	// Parse webhook payload
	var payload LemonSqueezyWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		h.logger.Error("Failed to parse webhook payload", zap.Error(err))
		h.writeError(w, http.StatusBadRequest, "Invalid payload")
		return
	}

	h.logger.Info("Received Lemon Squeezy webhook",
		zap.String("event", payload.Meta.EventName),
		zap.String("type", payload.Data.Type),
	)

	// Process webhook event based on type
	ctx := r.Context()
	switch payload.Meta.EventName {
	case "subscription_created", "subscription_updated":
		if err := h.handleSubscriptionEvent(ctx, payload); err != nil {
			h.logger.Error("Failed to handle subscription event",
				zap.Error(err),
				zap.String("event", payload.Meta.EventName),
			)
			h.writeError(w, http.StatusInternalServerError, "Failed to process webhook")
			return
		}
	case "subscription_cancelled", "subscription_expired":
		if err := h.handleSubscriptionCancellation(ctx, payload); err != nil {
			h.logger.Error("Failed to handle subscription cancellation",
				zap.Error(err),
				zap.String("event", payload.Meta.EventName),
			)
			h.writeError(w, http.StatusInternalServerError, "Failed to process webhook")
			return
		}
	default:
		h.logger.Info("Unhandled webhook event",
			zap.String("event", payload.Meta.EventName),
		)
		// Return 200 OK even for unhandled events (to avoid webhook retries)
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// verifyLemonSqueezySignature verifies the webhook signature using HMAC-SHA256
// Lemon Squeezy signs webhooks with the webhook secret
func (h *WebhookHandlers) verifyLemonSqueezySignature(body []byte, signature string) bool {
	if h.webhookSecret == "" {
		h.logger.Warn("Webhook secret not configured - skipping signature verification")
		return true // Allow in development if secret not set
	}

	mac := hmac.New(sha256.New, []byte(h.webhookSecret))
	mac.Write(body)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	// Lemon Squeezy sends signature in format: sha256=<hex>
	// Extract hex part if present
	if len(signature) > 7 && signature[:7] == "sha256=" {
		signature = signature[7:]
	}

	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// handleSubscriptionEvent handles subscription created/updated events
func (h *WebhookHandlers) handleSubscriptionEvent(ctx context.Context, payload LemonSqueezyWebhookPayload) error {
	// Extract subscription data from payload
	// Note: Lemon Squeezy webhook payload structure may vary - adjust as needed
	subscriptionID := payload.Data.Attributes.SubscriptionID
	customerID := payload.Data.Attributes.CustomerID
	planName := payload.Data.Attributes.PlanName // e.g., "starter", "pro"
	status := payload.Data.Attributes.Status     // e.g., "active", "cancelled"

	h.logger.Info("Processing subscription event",
		zap.String("subscription_id", subscriptionID),
		zap.String("customer_id", customerID),
		zap.String("plan", planName),
		zap.String("status", status),
	)

	// Get user by customer ID (Lemon Squeezy customer ID should be stored in user metadata)
	// For MVP, we'll use customer ID as email or lookup by external ID
	// TODO: Add customer_id field to users table or create mapping table
	user, err := h.userRepo.GetUserByEmail(customerID) // Assuming customerID is email for MVP
	if err != nil {
		return fmt.Errorf("failed to find user for customer ID %s: %w", customerID, err)
	}

	// Get plan limits based on plan name
	ramLimitMB, diskLimitGB := services.GetPlanLimits(planName)

	// Activate subscription
	if status == "active" {
		if err := h.subscriptionService.ActivateSubscription(
			ctx,
			user.ID,
			planName,
			subscriptionID,
			ramLimitMB,
			diskLimitGB,
			user.Email,
		); err != nil {
			return fmt.Errorf("failed to activate subscription: %w", err)
		}
	} else if status == "cancelled" {
		// Handle cancellation (set status to cancelled)
		if err := h.subscriptionService.CancelSubscription(ctx, user.ID); err != nil {
			return fmt.Errorf("failed to cancel subscription: %w", err)
		}
	}

	return nil
}

// handleSubscriptionCancellation handles subscription cancellation/expiration events
func (h *WebhookHandlers) handleSubscriptionCancellation(ctx context.Context, payload LemonSqueezyWebhookPayload) error {
	customerID := payload.Data.Attributes.CustomerID

	// Get user by customer ID
	user, err := h.userRepo.GetUserByEmail(customerID) // Assuming customerID is email for MVP
	if err != nil {
		return fmt.Errorf("failed to find user for customer ID %s: %w", customerID, err)
	}

	// Cancel subscription
	if err := h.subscriptionService.CancelSubscription(ctx, user.ID); err != nil {
		return fmt.Errorf("failed to cancel subscription: %w", err)
	}

	return nil
}

// LemonSqueezyWebhookPayload represents the structure of a Lemon Squeezy webhook payload
// Adjust fields based on actual Lemon Squeezy webhook format
type LemonSqueezyWebhookPayload struct {
	Meta struct {
		EventName string `json:"event_name"`
	} `json:"meta"`
	Data struct {
		Type       string `json:"type"`
		Attributes struct {
			SubscriptionID string `json:"subscription_id,omitempty"`
			CustomerID     string `json:"customer_id,omitempty"`
			PlanName       string `json:"plan_name,omitempty"`
			Status         string `json:"status,omitempty"`
		} `json:"attributes"`
	} `json:"data"`
}

// CancelSubscription cancels a subscription
func (h *WebhookHandlers) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", zap.Error(err))
	}
}

func (h *WebhookHandlers) writeError(w http.ResponseWriter, status int, message string) {
	h.writeJSON(w, status, map[string]string{"error": message})
}

