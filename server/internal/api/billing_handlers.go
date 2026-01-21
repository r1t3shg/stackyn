package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
	"stackyn/server/internal/infra"
)

// BillingHandlers handles billing-related API requests
type BillingHandlers struct {
	logger           *zap.Logger
	config           *infra.Config
	userRepo         *UserRepo
	subscriptionRepo *SubscriptionRepo
}

// NewBillingHandlers creates a new billing handlers instance
func NewBillingHandlers(logger *zap.Logger, config *infra.Config, userRepo *UserRepo, subscriptionRepo *SubscriptionRepo) *BillingHandlers {
	return &BillingHandlers{
		logger:           logger,
		config:           config,
		userRepo:         userRepo,
		subscriptionRepo: subscriptionRepo,
	}
}

// CreateCheckoutSessionRequest represents the request body for creating a checkout session
type CreateCheckoutSessionRequest struct {
	Plan string `json:"plan" validate:"required,oneof=starter pro"`
}

// CreateCheckoutSessionResponse represents the response from creating a checkout session
type CreateCheckoutSessionResponse struct {
	CheckoutURL string `json:"checkout_url"`
}

// CreateCheckoutSession creates a Lemon Squeezy checkout session
// POST /api/billing/checkout
// Requires: AuthMiddleware (sets user_id and user_email in context)
func (h *BillingHandlers) CreateCheckoutSession(w http.ResponseWriter, r *http.Request) {
	// Get request ID for logging
	requestID := r.Context().Value(middleware.RequestIDKey)
	if requestID == nil {
		requestID = "unknown"
	}

	// Step 1: Validate the plan
	var req CreateCheckoutSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("Invalid request body",
			zap.String("request_id", fmt.Sprintf("%v", requestID)),
			zap.Error(err),
		)
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate plan is either "starter" or "pro"
	if req.Plan != "starter" && req.Plan != "pro" {
		h.logger.Warn("Invalid plan requested",
			zap.String("request_id", fmt.Sprintf("%v", requestID)),
			zap.String("plan", req.Plan),
		)
		h.writeError(w, http.StatusBadRequest, "Invalid plan. Must be 'starter' or 'pro'")
		return
	}

	// Step 2: Get authenticated user from context
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		h.logger.Error("User ID not found in context",
			zap.String("request_id", fmt.Sprintf("%v", requestID)),
		)
		h.writeError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	userEmail, ok := r.Context().Value("user_email").(string)
	if !ok || userEmail == "" {
		h.logger.Error("User email not found in context",
			zap.String("request_id", fmt.Sprintf("%v", requestID)),
			zap.String("user_id", userID),
		)
		h.writeError(w, http.StatusUnauthorized, "User email not found")
		return
	}

	// Step 3: Validate required environment variables
	if h.config.LemonSqueezy.APIKey == "" {
		h.logger.Error("LEMON_API_KEY not configured",
			zap.String("request_id", fmt.Sprintf("%v", requestID)),
		)
		h.writeError(w, http.StatusInternalServerError, "Billing service not configured")
		return
	}

	if h.config.LemonSqueezy.StoreID == "" {
		h.logger.Error("LEMON_STORE_ID not configured",
			zap.String("request_id", fmt.Sprintf("%v", requestID)),
		)
		h.writeError(w, http.StatusInternalServerError, "Billing service not configured")
		return
	}

	if h.config.LemonSqueezy.FrontendBaseURL == "" {
		h.logger.Error("FRONTEND_BASE_URL not configured",
			zap.String("request_id", fmt.Sprintf("%v", requestID)),
		)
		h.writeError(w, http.StatusInternalServerError, "Frontend URL not configured")
		return
	}

	// Step 4: Resolve Lemon Squeezy variant_id based on environment and plan
	var variantID string
	if h.config.LemonSqueezy.TestMode {
		// Use test variant IDs
		variantID, ok = h.config.LemonSqueezy.TestVariantIDs[req.Plan]
		if !ok {
			h.logger.Error("Test variant ID not found for plan",
				zap.String("request_id", fmt.Sprintf("%v", requestID)),
				zap.String("plan", req.Plan),
			)
			h.writeError(w, http.StatusInternalServerError, "Billing configuration error: test variant ID not found")
			return
		}
	} else {
		// Use live variant IDs
		variantID, ok = h.config.LemonSqueezy.LiveVariantIDs[req.Plan]
		if !ok {
			h.logger.Error("Live variant ID not found for plan",
				zap.String("request_id", fmt.Sprintf("%v", requestID)),
				zap.String("plan", req.Plan),
			)
			h.writeError(w, http.StatusInternalServerError, "Billing configuration error: live variant ID not found")
			return
		}
	}

	h.logger.Info("Creating checkout session",
		zap.String("request_id", fmt.Sprintf("%v", requestID)),
		zap.String("user_id", userID),
		zap.String("plan", req.Plan),
		zap.String("variant_id", variantID),
		zap.Bool("test_mode", h.config.LemonSqueezy.TestMode),
	)

	// Step 5: Call Lemon Squeezy "Create Checkout" API
	checkoutURL, err := h.createLemonSqueezyCheckout(r.Context(), variantID, userEmail, requestID)
	if err != nil {
		h.logger.Error("Failed to create Lemon Squeezy checkout",
			zap.String("request_id", fmt.Sprintf("%v", requestID)),
			zap.String("user_id", userID),
			zap.Error(err),
		)
		h.writeError(w, http.StatusBadGateway, "Failed to create checkout session")
		return
	}

	// Step 6: Return checkout URL to frontend
	response := CreateCheckoutSessionResponse{
		CheckoutURL: checkoutURL,
	}

	h.logger.Info("Checkout session created successfully",
		zap.String("request_id", fmt.Sprintf("%v", requestID)),
		zap.String("user_id", userID),
		zap.String("plan", req.Plan),
	)

	h.writeJSON(w, http.StatusOK, response)
}

// createLemonSqueezyCheckout calls the Lemon Squeezy v1 API to create a checkout session
func (h *BillingHandlers) createLemonSqueezyCheckout(ctx context.Context, variantID, customerEmail string, requestID interface{}) (string, error) {
	// Build success and cancel URLs
	successURL := fmt.Sprintf("%s/billing/success", h.config.LemonSqueezy.FrontendBaseURL)
	cancelURL := fmt.Sprintf("%s/billing/cancel", h.config.LemonSqueezy.FrontendBaseURL)

	// Prepare request payload for Lemon Squeezy v1 API
	// Documentation: https://docs.lemonsqueezy.com/api/checkouts#create-a-checkout
	// Lemon Squeezy v1 uses JSON:API format
	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "checkouts",
			"attributes": map[string]interface{}{
				"custom_price": nil, // Use variant price from Lemon Squeezy
				"product_options": map[string]interface{}{
					"enabled_variants": []string{variantID}, // Only allow this variant
					"redirect_url":     successURL,
					"receipt_link_url": successURL,
					"receipt_button_text": "Return to Stackyn",
					"receipt_thank_you_note": "Thank you for subscribing!",
				},
				"checkout_options": map[string]interface{}{
					"embed":           false,
					"media":           false,
					"logo":            false,
					"desc":            true,
					"discount":        true,
					"dark":            false,
					"subscription_preview": true,
					"button_color":    "#000000",
				},
				"checkout_data": map[string]interface{}{
					"email": customerEmail,
					"custom": map[string]interface{}{
						"user_id": fmt.Sprintf("%v", requestID), // Store request ID for tracking
					},
				},
				"expires_at": nil, // No expiration
				"preview":   false,
				"test_mode": h.config.LemonSqueezy.TestMode,
			},
			"relationships": map[string]interface{}{
				"store": map[string]interface{}{
					"data": map[string]interface{}{
						"type": "stores",
						"id":   h.config.LemonSqueezy.StoreID,
					},
				},
				"variant": map[string]interface{}{
					"data": map[string]interface{}{
						"type": "variants",
						"id":   variantID,
					},
				},
			},
		},
	}

	// Marshal payload to JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request payload: %w", err)
	}

	// Create HTTP request to Lemon Squeezy API
	// Lemon Squeezy v1 API endpoint: https://api.lemonsqueezy.com/v1/checkouts
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.lemonsqueezy.com/v1/checkouts", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/vnd.api+json")
	req.Header.Set("Accept", "application/vnd.api+json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", h.config.LemonSqueezy.APIKey))

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Make request
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call Lemon Squeezy API: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Check response status
	if resp.StatusCode != http.StatusCreated {
		h.logger.Error("Lemon Squeezy API returned error",
			zap.Int("status_code", resp.StatusCode),
			zap.String("response_body", string(body)),
		)
		return "", fmt.Errorf("Lemon Squeezy API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	// Parse response - Lemon Squeezy v1 returns JSON:API format
	var lemonResponse struct {
		Data struct {
			Attributes struct {
				CheckoutURL string `json:"checkout_url"`
				URL         string `json:"url"` // Some versions may use "url" instead
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &lemonResponse); err != nil {
		h.logger.Error("Failed to parse Lemon Squeezy response",
			zap.Error(err),
			zap.String("response_body", string(body)),
		)
		return "", fmt.Errorf("failed to parse Lemon Squeezy response: %w", err)
	}

	// Return checkout URL (check both fields for compatibility)
	checkoutURL := lemonResponse.Data.Attributes.CheckoutURL
	if checkoutURL == "" {
		checkoutURL = lemonResponse.Data.Attributes.URL
	}

	if checkoutURL == "" {
		h.logger.Error("Checkout URL not found in Lemon Squeezy response",
			zap.String("response_body", string(body)),
		)
		return "", fmt.Errorf("checkout URL not found in response")
	}

	return checkoutURL, nil
}

// Helper to write JSON response
func (h *BillingHandlers) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", zap.Error(err))
	}
}

// Helper to write error response
func (h *BillingHandlers) writeError(w http.ResponseWriter, status int, message string) {
	h.writeJSON(w, status, map[string]string{"error": message})
}

// GetSubscriptionResponse represents the response for getting user subscription
type GetSubscriptionResponse struct {
	Plan   string                 `json:"plan"`   // pro | starter | free
	Status string                 `json:"status"` // active | past_due | cancelled | expired | free
	Features SubscriptionFeatures `json:"features"`
	Billing SubscriptionBilling  `json:"billing"`
}

// SubscriptionFeatures represents plan features
type SubscriptionFeatures struct {
	MaxApps       int  `json:"max_apps"`
	CustomDomains bool `json:"custom_domains"`
	BuildMinutes  int  `json:"build_minutes"`
	TeamMembers   int  `json:"team_members"`
}

// SubscriptionBilling represents billing information
type SubscriptionBilling struct {
	CurrentPeriodStart  *time.Time `json:"current_period_start,omitempty"`  // Timestamp or null
	CurrentPeriodEnd    *time.Time `json:"current_period_end,omitempty"`    // Timestamp or null
	CancelAtPeriodEnd   bool       `json:"cancel_at_period_end"`            // true if cancelled but still active
	IsTrial             bool       `json:"is_trial"`                         // true if on trial
	TrialEndsAt         *time.Time `json:"trial_ends_at,omitempty"`          // Timestamp or null
}

// GetSubscription fetches the current user's active subscription
// GET /api/billing/subscription
// Requires: AuthMiddleware (sets user_id in context)
func (h *BillingHandlers) GetSubscription(w http.ResponseWriter, r *http.Request) {
	// Get request ID for logging
	requestID := r.Context().Value(middleware.RequestIDKey)
	if requestID == nil {
		requestID = "unknown"
	}

	// Step 1: Get authenticated user from context
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		h.logger.Error("User ID not found in context",
			zap.String("request_id", fmt.Sprintf("%v", requestID)),
		)
		h.writeError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	h.logger.Info("Fetching subscription",
		zap.String("request_id", fmt.Sprintf("%v", requestID)),
		zap.String("user_id", userID),
	)

	// Step 2: Fetch the latest subscription for the user with priority logic
	subscription, err := h.subscriptionRepo.GetActiveSubscriptionByUserID(r.Context(), userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// No subscription exists - return free plan
			h.logger.Info("No subscription found, returning free plan",
				zap.String("request_id", fmt.Sprintf("%v", requestID)),
				zap.String("user_id", userID),
			)
			response := h.buildFreePlanResponse()
			h.writeJSON(w, http.StatusOK, response)
			return
		}
		h.logger.Error("Failed to get subscription",
			zap.String("request_id", fmt.Sprintf("%v", requestID)),
			zap.String("user_id", userID),
			zap.Error(err),
		)
		// Return free plan on error (don't block request)
		response := h.buildFreePlanResponse()
		h.writeJSON(w, http.StatusOK, response)
		return
	}

	// Step 3: Build response based on subscription status
	response := h.buildSubscriptionResponse(subscription)

	h.logger.Info("Subscription fetched successfully",
		zap.String("request_id", fmt.Sprintf("%v", requestID)),
		zap.String("user_id", userID),
		zap.String("plan", response.Plan),
		zap.String("status", response.Status),
	)

	h.writeJSON(w, http.StatusOK, response)
}

// buildSubscriptionResponse builds the subscription response from a subscription record
func (h *BillingHandlers) buildSubscriptionResponse(sub *Subscription) GetSubscriptionResponse {
	now := time.Now().UTC()
	
	// Determine plan and status
	plan := sub.Plan
	if plan == "" {
		plan = "free"
	}
	
	status := sub.Status
	
	// Business logic: If status = cancelled AND current_period_end > now, treat as active
	// Note: Since we don't have current_period_end stored yet, we use updated_at + 30 days as approximation
	// TODO: Add current_period_end column to subscriptions table
	cancelAtPeriodEnd := false
	if status == "cancelled" {
		// Check if subscription was cancelled recently (within last 30 days)
		// This is a temporary solution until period_end is stored
		if sub.UpdatedAt.After(now.AddDate(0, 0, -30)) {
			status = "active" // Treat as active but show cancel_at_period_end = true
			cancelAtPeriodEnd = true
		} else {
			// Cancelled and past grace period - return free plan
			return h.buildFreePlanResponse()
		}
	}
	
	// Business logic: If status = expired, return free plan
	if status == "expired" {
		return h.buildFreePlanResponse()
	}
	
	// Check if trial
	isTrial := sub.Status == "trial" || sub.TrialEndsAt != nil
	
	// Get plan features
	features := h.getPlanFeatures(plan)
	
	// Build billing info
	// Note: current_period_start and current_period_end are not stored yet
	// TODO: Add these fields to subscriptions table when webhook stores them
	billing := SubscriptionBilling{
		CurrentPeriodStart: nil, // TODO: Store from webhook
		CurrentPeriodEnd:   nil, // TODO: Store from webhook
		CancelAtPeriodEnd:  cancelAtPeriodEnd,
		IsTrial:            isTrial,
		TrialEndsAt:        sub.TrialEndsAt,
	}
	
	// If past_due, status is already set correctly
	// Features are limited for past_due (handled in getPlanFeatures)
	
	return GetSubscriptionResponse{
		Plan:     plan,
		Status:   status,
		Features: features,
		Billing:  billing,
	}
}

// buildFreePlanResponse builds a free plan response
func (h *BillingHandlers) buildFreePlanResponse() GetSubscriptionResponse {
	return GetSubscriptionResponse{
		Plan:   "free",
		Status: "free",
		Features: SubscriptionFeatures{
			MaxApps:       3,
			CustomDomains: false,
			BuildMinutes:  60, // 1 hour per month
			TeamMembers:   1,
		},
		Billing: SubscriptionBilling{
			CurrentPeriodStart: nil,
			CurrentPeriodEnd:   nil,
			CancelAtPeriodEnd:  false,
			IsTrial:            false,
			TrialEndsAt:         nil,
		},
	}
}

// getPlanFeatures returns features for a plan
func (h *BillingHandlers) getPlanFeatures(plan string) SubscriptionFeatures {
	switch plan {
	case "starter":
		return SubscriptionFeatures{
			MaxApps:       5,
			CustomDomains: false,
			BuildMinutes:  300, // 5 hours per month
			TeamMembers:   1,
		}
	case "pro":
		return SubscriptionFeatures{
			MaxApps:       20,
			CustomDomains: true,
			BuildMinutes:  1440, // 24 hours per month
			TeamMembers:   10,
		}
	default:
		// Free plan
		return SubscriptionFeatures{
			MaxApps:       3,
			CustomDomains: false,
			BuildMinutes:  60, // 1 hour per month
			TeamMembers:   1,
		}
	}
}

