package api

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"stackyn/server/internal/infra"
	"stackyn/server/internal/services"
)

// WebhookHandlers handles webhook requests from external services (e.g., Lemon Squeezy)
type WebhookHandlers struct {
	logger              *zap.Logger
	subscriptionService *services.SubscriptionService
	subscriptionRepo    *SubscriptionRepo
	userRepo            *UserRepo
	config              *infra.Config
	pool                *pgxpool.Pool
}

// NewWebhookHandlers creates a new webhook handlers instance
func NewWebhookHandlers(logger *zap.Logger, subscriptionService *services.SubscriptionService, subscriptionRepo *SubscriptionRepo, userRepo *UserRepo, config *infra.Config, pool *pgxpool.Pool) *WebhookHandlers {
	return &WebhookHandlers{
		logger:              logger,
		subscriptionService: subscriptionService,
		subscriptionRepo:    subscriptionRepo,
		userRepo:            userRepo,
		config:              config,
		pool:                pool,
	}
}

// LemonSqueezyWebhook handles webhook events from Lemon Squeezy
// POST /api/billing/webhook
func (h *WebhookHandlers) LemonSqueezyWebhook(w http.ResponseWriter, r *http.Request) {
	// Log incoming webhook request for debugging
	h.logger.Info("Received webhook request",
		zap.String("method", r.Method),
		zap.String("path", r.URL.Path),
		zap.String("remote_addr", r.RemoteAddr),
		zap.String("user_agent", r.UserAgent()),
	)

	// Step 1: Read the raw request body (must be done before parsing JSON)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("Failed to read webhook body", zap.Error(err))
		h.writeError(w, http.StatusBadRequest, "Failed to read request body")
		return
	}

	h.logger.Info("Webhook body received",
		zap.Int("body_size", len(body)),
		zap.String("body_preview", string(body[:min(len(body), 500)])), // Log first 500 chars for debugging
	)

	// Step 2: Verify Lemon Squeezy webhook signature BEFORE parsing JSON
	// Lemon Squeezy sends signature in X-Signature header
	signature := r.Header.Get("X-Signature")
	if signature == "" {
		h.logger.Warn("Webhook request missing X-Signature header",
			zap.Strings("available_headers", getHeaderNames(r)),
		)
		h.writeError(w, http.StatusUnauthorized, "Missing signature")
		return
	}

	h.logger.Info("Webhook signature header found",
		zap.String("signature_preview", signature[:min(len(signature), 20)]+"..."),
		zap.Bool("webhook_secret_configured", h.config.LemonSqueezy.WebhookSecret != ""),
	)

	// Verify signature using HMAC-SHA256
	if !h.verifyLemonSqueezySignature(body, signature) {
		h.logger.Warn("Invalid webhook signature",
			zap.String("signature_received", signature),
			zap.Bool("webhook_secret_set", h.config.LemonSqueezy.WebhookSecret != ""),
		)
		h.writeError(w, http.StatusUnauthorized, "Invalid signature")
		return
	}

	h.logger.Info("Webhook signature verified successfully")

	// Step 3: Parse JSON payload after verification
	var payload LemonSqueezyWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		h.logger.Error("Failed to parse webhook payload", zap.Error(err))
		h.writeError(w, http.StatusBadRequest, "Invalid payload")
		return
	}

	// Extract event name and subscription ID for logging
	eventName := payload.Meta.EventName
	subscriptionID := payload.Data.ID
	customDataUserID := payload.Meta.CustomData.UserID

	h.logger.Info("Received Lemon Squeezy webhook",
		zap.String("event_name", eventName),
		zap.String("subscription_id", subscriptionID),
		zap.String("data_type", payload.Data.Type),
		zap.String("custom_data_user_id", customDataUserID),
		zap.String("customer_id", payload.Data.Attributes.CustomerID.String()),
		zap.String("variant_id", payload.Data.Attributes.VariantID.String()),
		zap.String("status", payload.Data.Attributes.Status),
	)

	// Step 4: Process webhook event based on type
	ctx := r.Context()
	if err := h.processWebhookEvent(ctx, payload); err != nil {
		h.logger.Error("Failed to process webhook event",
			zap.Error(err),
			zap.String("event_name", eventName),
			zap.String("subscription_id", subscriptionID),
		)
		// Still return 200 to prevent webhook retries
		// Lemon Squeezy will retry on non-2xx responses
		h.writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
		return
	}

	// Step 5: Return HTTP 200 on successful processing
	h.writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// verifyLemonSqueezySignature verifies the webhook signature using HMAC-SHA256
// Lemon Squeezy signs webhooks with the webhook secret
func (h *WebhookHandlers) verifyLemonSqueezySignature(body []byte, signature string) bool {
	webhookSecret := h.config.LemonSqueezy.WebhookSecret
	if webhookSecret == "" {
		h.logger.Warn("Webhook secret not configured - rejecting request for security")
		return false // Reject in production if secret not set
	}

	// Compute HMAC-SHA256 signature
	mac := hmac.New(sha256.New, []byte(webhookSecret))
	mac.Write(body)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	// Lemon Squeezy sends signature in format: sha256=<hex> or just <hex>
	// Extract hex part if present
	receivedSignature := signature
	if len(signature) > 7 && signature[:7] == "sha256=" {
		receivedSignature = signature[7:]
	}

	// Log signature details for debugging (without exposing full secrets)
	h.logger.Debug("Verifying webhook signature",
		zap.Int("body_length", len(body)),
		zap.Int("received_sig_length", len(receivedSignature)),
		zap.Int("expected_sig_length", len(expectedSignature)),
		zap.String("received_sig_prefix", receivedSignature[:min(len(receivedSignature), 8)]),
		zap.String("expected_sig_prefix", expectedSignature[:min(len(expectedSignature), 8)]),
	)

	// Use constant-time comparison to prevent timing attacks
	isValid := hmac.Equal([]byte(receivedSignature), []byte(expectedSignature))
	
	if !isValid {
		h.logger.Warn("Signature verification failed",
			zap.String("received_sig_prefix", receivedSignature[:min(len(receivedSignature), 16)]),
			zap.String("expected_sig_prefix", expectedSignature[:min(len(expectedSignature), 16)]),
		)
	}
	
	return isValid
}

// processWebhookEvent processes webhook events based on event_name
func (h *WebhookHandlers) processWebhookEvent(ctx context.Context, payload LemonSqueezyWebhookPayload) error {
	eventName := payload.Meta.EventName

	switch eventName {
	case "subscription_created":
		return h.handleSubscriptionCreated(ctx, payload)
	case "subscription_updated":
		return h.handleSubscriptionUpdated(ctx, payload)
	case "subscription_plan_changed":
		return h.handleSubscriptionPlanChanged(ctx, payload)
	case "subscription_payment_success":
		return h.handleSubscriptionPaymentSuccess(ctx, payload)
	case "subscription_payment_failed":
		return h.handleSubscriptionPaymentFailed(ctx, payload)
	case "subscription_cancelled":
		return h.handleSubscriptionCancelled(ctx, payload)
	case "subscription_expired":
		return h.handleSubscriptionExpired(ctx, payload)
	default:
		// Gracefully ignore unknown events
		h.logger.Info("Unhandled webhook event (ignoring)",
			zap.String("event_name", eventName),
		)
		// TODO: Add handlers for future events:
		// - subscription_resumed
		// - subscription_paused
		// - order_created
		// - order_refunded
		return nil
	}
}

// handleSubscriptionCreated handles subscription_created event
// Creates subscription record with status = active
func (h *WebhookHandlers) handleSubscriptionCreated(ctx context.Context, payload LemonSqueezyWebhookPayload) error {
	attrs := payload.Data.Attributes
	subscriptionID := payload.Data.ID

	// Extract data from payload
	lemonSubscriptionID := subscriptionID
	lemonCustomerID := attrs.CustomerID.String() // Extract customer ID for portal access
	planName := h.extractPlanName(attrs)
	status := "active" // New subscriptions are always active
	// Note: currentPeriodEnd (RenewsAt) is available but not stored in current schema
	// TODO: Add current_period_end column to subscriptions table if needed

	// Get user_id from custom_data (set during checkout)
	// Lemon Squeezy stores custom_data from checkout in meta.custom_data
	userID := payload.Meta.CustomData.UserID
	if userID == "" {
		h.logger.Warn("user_id not found in custom_data, attempting fallback lookup",
			zap.String("subscription_id", lemonSubscriptionID),
			zap.String("customer_id", lemonCustomerID),
		)
		
		// Fallback: try to get from customer email
		// First, try to get customer email from Lemon Squeezy customer API
		// For now, we'll try to find user by email if we have it in the payload
		// Note: Lemon Squeezy webhook doesn't include customer email directly
		// We need to look it up or use a different approach
		
		// Alternative: Look up subscription by lemon_subscription_id to get user_id
		existingSub, err := h.subscriptionRepo.GetSubscriptionByLemonSubscriptionID(ctx, lemonSubscriptionID)
		if err == nil && existingSub != nil {
			userID = existingSub.UserID
			h.logger.Info("Found user_id from existing subscription",
				zap.String("user_id", userID),
				zap.String("subscription_id", lemonSubscriptionID),
			)
		} else {
			// Last resort: return error - we need user_id to create subscription
			return fmt.Errorf("user_id not found in custom_data and no existing subscription found for lemon_subscription_id: %s", lemonSubscriptionID)
		}
	} else {
		h.logger.Info("Found user_id in custom_data",
			zap.String("user_id", userID),
		)
	}

	h.logger.Info("Processing subscription_created",
		zap.String("subscription_id", lemonSubscriptionID),
		zap.String("user_id", userID),
		zap.String("plan", planName),
	)

	// Get plan limits based on plan name
	ramLimitMB, diskLimitGB := services.GetPlanLimits(planName)

	// Use UPSERT logic for idempotency
	// Check if subscription already exists
	existingSub, err := h.subscriptionRepo.GetSubscriptionByLemonSubscriptionID(ctx, lemonSubscriptionID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to check existing subscription: %w", err)
	}

	if existingSub != nil {
		// Subscription already exists - update it (idempotent)
		h.logger.Info("Subscription already exists, updating",
			zap.String("subscription_id", lemonSubscriptionID),
		)
		// Update existing subscription
		if err := h.subscriptionRepo.UpdateSubscription(
			ctx,
			existingSub.ID,
			planName,
			status,
			&ramLimitMB,
			&diskLimitGB,
			&lemonSubscriptionID,
			&lemonCustomerID, // Update customer ID if changed
		); err != nil {
			return fmt.Errorf("failed to update subscription: %w", err)
		}
		return nil
	}

	// Create new subscription
	_, err = h.subscriptionRepo.CreateSubscription(
		ctx,
		userID,
		lemonSubscriptionID,
		lemonCustomerID, // Store customer ID for portal access
		planName,
		status,
		nil, // No trial for paid subscriptions
		nil,
		ramLimitMB,
		diskLimitGB,
	)
	if err != nil {
		return fmt.Errorf("failed to create subscription: %w", err)
	}

	// Sync billing fields to users table
	if err := h.subscriptionService.ActivateSubscription(
		ctx,
		userID,
		planName,
		lemonSubscriptionID,
		ramLimitMB,
		diskLimitGB,
		"", // Email not needed for sync
	); err != nil {
		h.logger.Warn("Failed to sync billing fields to users table",
			zap.Error(err),
			zap.String("user_id", userID),
		)
		// Non-critical - subscription table is source of truth
	}

	return nil
}

// handleSubscriptionUpdated handles subscription_updated event
// Updates subscription fields and syncs status, renew date, variant
func (h *WebhookHandlers) handleSubscriptionUpdated(ctx context.Context, payload LemonSqueezyWebhookPayload) error {
	subscriptionID := payload.Data.ID

	// Extract data from payload
	lemonSubscriptionID := subscriptionID
	lemonCustomerID := payload.Data.Attributes.CustomerID.String() // Extract customer ID
	planName := h.extractPlanName(payload.Data.Attributes)
	status := h.mapLemonStatusToInternal(payload.Data.Attributes.Status)
	// Note: currentPeriodEnd (RenewsAt) is available but not stored in current schema
	// TODO: Add current_period_end column to subscriptions table if needed

	h.logger.Info("Processing subscription_updated",
		zap.String("subscription_id", lemonSubscriptionID),
		zap.String("plan", planName),
		zap.String("status", status),
	)

	// Get existing subscription
	existingSub, err := h.subscriptionRepo.GetSubscriptionByLemonSubscriptionID(ctx, lemonSubscriptionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Subscription doesn't exist - create it (handles race conditions)
			return h.handleSubscriptionCreated(ctx, payload)
		}
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	// Get plan limits
	ramLimitMB, diskLimitGB := services.GetPlanLimits(planName)

	// Update subscription
	return h.subscriptionRepo.UpdateSubscription(
		ctx,
		existingSub.ID,
		planName,
		status,
		&ramLimitMB,
		&diskLimitGB,
		&lemonSubscriptionID,
		&lemonCustomerID, // Update customer ID if changed
	)
}

// handleSubscriptionPlanChanged handles subscription_plan_changed event
// Updates plan + variant, keeps subscription active
func (h *WebhookHandlers) handleSubscriptionPlanChanged(ctx context.Context, payload LemonSqueezyWebhookPayload) error {
	subscriptionID := payload.Data.ID

	lemonSubscriptionID := subscriptionID
	planName := h.extractPlanName(payload.Data.Attributes)
	status := "active" // Keep subscription active when plan changes

	h.logger.Info("Processing subscription_plan_changed",
		zap.String("subscription_id", lemonSubscriptionID),
		zap.String("new_plan", planName),
	)

	// Get existing subscription
	existingSub, err := h.subscriptionRepo.GetSubscriptionByLemonSubscriptionID(ctx, lemonSubscriptionID)
	if err != nil {
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	// Get plan limits for new plan
	ramLimitMB, diskLimitGB := services.GetPlanLimits(planName)

	// Update plan and limits, keep status active
	lemonCustomerIDStr := payload.Data.Attributes.CustomerID.String()
	lemonCustomerID := &lemonCustomerIDStr
	return h.subscriptionRepo.UpdateSubscription(
		ctx,
		existingSub.ID,
		planName,
		status,
		&ramLimitMB,
		&diskLimitGB,
		&lemonSubscriptionID,
		lemonCustomerID, // Update customer ID if changed
	)
}

// handleSubscriptionPaymentSuccess handles subscription_payment_success event
// Marks subscription as active, updates last_payment_at, clears past_due flags
func (h *WebhookHandlers) handleSubscriptionPaymentSuccess(ctx context.Context, payload LemonSqueezyWebhookPayload) error {
	subscriptionID := payload.Data.ID

	lemonSubscriptionID := subscriptionID
	status := "active" // Payment succeeded - mark as active

	h.logger.Info("Processing subscription_payment_success",
		zap.String("subscription_id", lemonSubscriptionID),
	)

	// Get existing subscription
	existingSub, err := h.subscriptionRepo.GetSubscriptionByLemonSubscriptionID(ctx, lemonSubscriptionID)
	if err != nil {
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	// Update status to active (clears any past_due flags)
	// Customer ID shouldn't change on payment success, but update if present
	lemonCustomerIDStr := payload.Data.Attributes.CustomerID.String()
	lemonCustomerID := &lemonCustomerIDStr
	return h.subscriptionRepo.UpdateSubscription(
		ctx,
		existingSub.ID,
		existingSub.Plan, // Keep existing plan
		status,
		nil, // Don't change limits
		nil,
		&lemonSubscriptionID,
		lemonCustomerID, // Update customer ID if changed
	)
}

// handleSubscriptionPaymentFailed handles subscription_payment_failed event
// Marks subscription as past_due, does NOT immediately disable services
func (h *WebhookHandlers) handleSubscriptionPaymentFailed(ctx context.Context, payload LemonSqueezyWebhookPayload) error {
	subscriptionID := payload.Data.ID

	lemonSubscriptionID := subscriptionID
	status := "past_due" // Payment failed - mark as past_due

	h.logger.Info("Processing subscription_payment_failed",
		zap.String("subscription_id", lemonSubscriptionID),
	)

	// Get existing subscription
	existingSub, err := h.subscriptionRepo.GetSubscriptionByLemonSubscriptionID(ctx, lemonSubscriptionID)
	if err != nil {
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	// Update status to past_due (don't disable services yet)
	// Customer ID shouldn't change on payment failure, but update if present
	lemonCustomerIDStr := payload.Data.Attributes.CustomerID.String()
	lemonCustomerID := &lemonCustomerIDStr
	return h.subscriptionRepo.UpdateSubscription(
		ctx,
		existingSub.ID,
		existingSub.Plan, // Keep existing plan
		status,
		nil, // Don't change limits
		nil,
		&lemonSubscriptionID,
		lemonCustomerID, // Update customer ID if changed
	)
}

// handleSubscriptionCancelled handles subscription_cancelled event
// Marks subscription as cancelled, keeps access until period_end
func (h *WebhookHandlers) handleSubscriptionCancelled(ctx context.Context, payload LemonSqueezyWebhookPayload) error {
	subscriptionID := payload.Data.ID

	lemonSubscriptionID := subscriptionID
	status := "cancelled" // User cancelled - mark as cancelled

	h.logger.Info("Processing subscription_cancelled",
		zap.String("subscription_id", lemonSubscriptionID),
	)

	// Get existing subscription
	existingSub, err := h.subscriptionRepo.GetSubscriptionByLemonSubscriptionID(ctx, lemonSubscriptionID)
	if err != nil {
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	// Update status to cancelled (access continues until period_end)
	// Customer ID shouldn't change on cancellation, but update if present
	lemonCustomerIDStr := payload.Data.Attributes.CustomerID.String()
	lemonCustomerID := &lemonCustomerIDStr
	return h.subscriptionRepo.UpdateSubscription(
		ctx,
		existingSub.ID,
		existingSub.Plan, // Keep existing plan
		status,
		nil, // Don't change limits
		nil,
		&lemonSubscriptionID,
		lemonCustomerID, // Update customer ID if changed
	)
}

// handleSubscriptionExpired handles subscription_expired event
// Marks subscription as expired, disables paid features immediately
func (h *WebhookHandlers) handleSubscriptionExpired(ctx context.Context, payload LemonSqueezyWebhookPayload) error {
	subscriptionID := payload.Data.ID

	lemonSubscriptionID := subscriptionID
	status := "expired" // Subscription expired - disable immediately

	h.logger.Info("Processing subscription_expired",
		zap.String("subscription_id", lemonSubscriptionID),
	)

	// Get existing subscription
	existingSub, err := h.subscriptionRepo.GetSubscriptionByLemonSubscriptionID(ctx, lemonSubscriptionID)
	if err != nil {
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	// Update status to expired (disable paid features immediately)
	// Customer ID shouldn't change on expiration, but update if present
	lemonCustomerIDStr := payload.Data.Attributes.CustomerID.String()
	lemonCustomerID := &lemonCustomerIDStr
	if err := h.subscriptionRepo.UpdateSubscription(
		ctx,
		existingSub.ID,
		existingSub.Plan, // Keep existing plan
		status,
		nil, // Don't change limits
		nil,
		&lemonSubscriptionID,
		lemonCustomerID, // Update customer ID if changed
	); err != nil {
		return err
	}

	// Expire subscription in subscription service (may stop apps)
	if err := h.subscriptionService.ExpireSubscription(ctx, existingSub.UserID, ""); err != nil {
		h.logger.Warn("Failed to expire subscription in service",
			zap.Error(err),
			zap.String("user_id", existingSub.UserID),
		)
		// Non-critical - status is already updated
	}

	return nil
}

// extractPlanName extracts plan name from variant_id or product name
// Maps Lemon Squeezy variant/product names to internal plan names
func (h *WebhookHandlers) extractPlanName(attrs LemonSqueezyAttributes) string {
	// Try to extract from variant_id by checking test/live variant maps
	variantID := attrs.VariantID.String()
	if variantID != "" {
		// Check test variant IDs
		if h.config.LemonSqueezy.TestMode {
			for plan, vid := range h.config.LemonSqueezy.TestVariantIDs {
				if vid == variantID {
					return plan
				}
			}
		} else {
			// Check live variant IDs
			for plan, vid := range h.config.LemonSqueezy.LiveVariantIDs {
				if vid == variantID {
					return plan
				}
			}
		}
	}

	// Fallback: try to extract from product name or variant name
	productName := attrs.ProductName
	if productName != "" {
		// Normalize product name to plan name
		if productName == "Starter" || productName == "starter" {
			return "starter"
		}
		if productName == "Pro" || productName == "pro" {
			return "pro"
		}
	}

	// Default fallback
	h.logger.Warn("Could not determine plan name from webhook payload",
		zap.String("variant_id", variantID),
		zap.String("product_name", productName),
	)
	return "starter" // Default fallback
}

// mapLemonStatusToInternal maps Lemon Squeezy status to internal status
func (h *WebhookHandlers) mapLemonStatusToInternal(lemonStatus string) string {
	switch lemonStatus {
	case "active":
		return "active"
	case "on_trial":
		return "trial"
	case "past_due":
		return "past_due"
	case "cancelled":
		return "cancelled"
	case "expired":
		return "expired"
	case "paused":
		return "cancelled" // Treat paused as cancelled
	default:
		h.logger.Warn("Unknown Lemon Squeezy status, defaulting to active",
			zap.String("lemon_status", lemonStatus),
		)
		return "active"
	}
}

// LemonSqueezyWebhookPayload represents the structure of a Lemon Squeezy webhook payload
// Based on Lemon Squeezy v1 webhook format
type LemonSqueezyWebhookPayload struct {
	Meta struct {
		EventName  string `json:"event_name"`
		CustomData struct {
			UserID string `json:"user_id"` // Set during checkout via custom_data
		} `json:"custom_data"`
	} `json:"meta"`
	Data struct {
		Type       string                 `json:"type"`
		ID         string                 `json:"id"` // Lemon subscription ID
		Attributes LemonSqueezyAttributes `json:"attributes"`
	} `json:"data"`
}

// FlexibleString can unmarshal from both string and number JSON types
// Lemon Squeezy sometimes sends IDs as numbers, sometimes as strings
type FlexibleString string

// UnmarshalJSON implements custom JSON unmarshaling for FlexibleString
func (fs *FlexibleString) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as string first
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		*fs = FlexibleString(str)
		return nil
	}
	
	// If that fails, try as number
	var num float64
	if err := json.Unmarshal(data, &num); err == nil {
		*fs = FlexibleString(fmt.Sprintf("%.0f", num)) // Convert to string without decimals
		return nil
	}
	
	return fmt.Errorf("cannot unmarshal %s into FlexibleString", string(data))
}

// String returns the string value
func (fs FlexibleString) String() string {
	return string(fs)
}

// LemonSqueezyAttributes represents subscription attributes from Lemon Squeezy
type LemonSqueezyAttributes struct {
	Status      string         `json:"status"`       // active, on_trial, past_due, cancelled, expired, paused
	CustomerID  FlexibleString `json:"customer_id"`  // Lemon customer ID (can be number or string)
	VariantID   FlexibleString `json:"variant_id"`   // Lemon variant ID (can be number or string)
	ProductName string         `json:"product_name"` // Product name (e.g., "Starter", "Pro")
	RenewsAt    string         `json:"renews_at"`     // Next renewal date (current_period_end) - ISO 8601 string
	EndsAt      string         `json:"ends_at"`       // When subscription ends (if cancelled) - ISO 8601 string
}

// Helper to write JSON response
func (h *WebhookHandlers) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", zap.Error(err))
	}
}

// Helper to write error response
func (h *WebhookHandlers) writeError(w http.ResponseWriter, status int, message string) {
	h.writeJSON(w, status, map[string]string{"error": message})
}

// Helper function to get header names for logging
func getHeaderNames(r *http.Request) []string {
	headers := make([]string, 0, len(r.Header))
	for name := range r.Header {
		headers = append(headers, name)
	}
	return headers
}

// Helper function for min (since Go 1.21+ has it, but for compatibility)
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
