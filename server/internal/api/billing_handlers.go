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
		// Log available test variant IDs for debugging
		h.logger.Info("Looking up test variant ID",
			zap.String("request_id", fmt.Sprintf("%v", requestID)),
			zap.String("plan", req.Plan),
			zap.Any("available_plans", getMapKeys(h.config.LemonSqueezy.TestVariantIDs)),
		)
		variantID, ok = h.config.LemonSqueezy.TestVariantIDs[req.Plan]
		if !ok {
			h.logger.Error("Test variant ID not found for plan",
				zap.String("request_id", fmt.Sprintf("%v", requestID)),
				zap.String("plan", req.Plan),
				zap.Strings("available_plans", getMapKeys(h.config.LemonSqueezy.TestVariantIDs)),
			)
			h.writeError(w, http.StatusInternalServerError, "Billing configuration error: test variant ID not found")
			return
		}
	} else {
		// Use live variant IDs
		// Log available live variant IDs for debugging
		h.logger.Info("Looking up live variant ID",
			zap.String("request_id", fmt.Sprintf("%v", requestID)),
			zap.String("plan", req.Plan),
			zap.Strings("available_plans", getMapKeys(h.config.LemonSqueezy.LiveVariantIDs)),
		)
		variantID, ok = h.config.LemonSqueezy.LiveVariantIDs[req.Plan]
		if !ok {
			h.logger.Error("Live variant ID not found for plan",
				zap.String("request_id", fmt.Sprintf("%v", requestID)),
				zap.String("plan", req.Plan),
				zap.Strings("available_plans", getMapKeys(h.config.LemonSqueezy.LiveVariantIDs)),
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
	// Build success URL (cancel URL is handled by Lemon Squeezy checkout options)
	successURL := fmt.Sprintf("%s/billing/success", h.config.LemonSqueezy.FrontendBaseURL)

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

// CreateCustomerPortalRequest represents the request body (empty for now, but extensible)
type CreateCustomerPortalRequest struct {
	// Empty - no input required from frontend
}

// CreateCustomerPortalResponse represents the response from creating a customer portal session
type CreateCustomerPortalResponse struct {
	PortalURL string `json:"portal_url"`
}

// CreateCustomerPortal creates a Lemon Squeezy customer portal session
// POST /api/billing/portal
// Requires: AuthMiddleware (sets user_id and user_email in context)
func (h *BillingHandlers) CreateCustomerPortal(w http.ResponseWriter, r *http.Request) {
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

	userEmail, ok := r.Context().Value("user_email").(string)
	if !ok || userEmail == "" {
		h.logger.Error("User email not found in context",
			zap.String("request_id", fmt.Sprintf("%v", requestID)),
			zap.String("user_id", userID),
		)
		h.writeError(w, http.StatusUnauthorized, "User email not found")
		return
	}

	h.logger.Info("Creating customer portal session",
		zap.String("request_id", fmt.Sprintf("%v", requestID)),
		zap.String("user_id", userID),
	)

	// Step 2: Fetch the user's latest subscription from database
	subscription, err := h.subscriptionRepo.GetActiveSubscriptionByUserID(r.Context(), userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.logger.Warn("No subscription found for user",
				zap.String("request_id", fmt.Sprintf("%v", requestID)),
				zap.String("user_id", userID),
			)
			h.writeError(w, http.StatusBadRequest, "No active subscription")
			return
		}
		h.logger.Error("Failed to get subscription",
			zap.String("request_id", fmt.Sprintf("%v", requestID)),
			zap.String("user_id", userID),
			zap.Error(err),
		)
		h.writeError(w, http.StatusInternalServerError, "Failed to retrieve subscription")
		return
	}

	// Step 3: Retrieve lemon_customer_id
	if subscription.LemonCustomerID == nil || *subscription.LemonCustomerID == "" {
		h.logger.Error("Missing lemon_customer_id in subscription",
			zap.String("request_id", fmt.Sprintf("%v", requestID)),
			zap.String("user_id", userID),
			zap.String("subscription_id", subscription.ID),
		)
		h.writeError(w, http.StatusInternalServerError, "Subscription customer ID not found")
		return
	}

	customerID := *subscription.LemonCustomerID

	h.logger.Info("Retrieved customer ID",
		zap.String("request_id", fmt.Sprintf("%v", requestID)),
		zap.String("user_id", userID),
		zap.String("customer_id", customerID),
	)

	// Step 4: Call Lemon Squeezy "Create Customer Portal" API
	portalURL, err := h.createLemonSqueezyCustomerPortal(r.Context(), customerID, requestID)
	if err != nil {
		h.logger.Error("Failed to create Lemon Squeezy customer portal",
			zap.String("request_id", fmt.Sprintf("%v", requestID)),
			zap.String("user_id", userID),
			zap.String("customer_id", customerID),
			zap.Error(err),
		)
		h.writeError(w, http.StatusBadGateway, "Failed to create customer portal session")
		return
	}

	// Step 5: Return portal URL to frontend
	response := CreateCustomerPortalResponse{
		PortalURL: portalURL,
	}

	h.logger.Info("Customer portal session created successfully",
		zap.String("request_id", fmt.Sprintf("%v", requestID)),
		zap.String("user_id", userID),
	)

	h.writeJSON(w, http.StatusOK, response)
}

// createLemonSqueezyCustomerPortal calls the Lemon Squeezy v1 API to create a customer portal session
func (h *BillingHandlers) createLemonSqueezyCustomerPortal(ctx context.Context, customerID string, requestID interface{}) (string, error) {
	// Build return URL
	returnURL := fmt.Sprintf("%s/billing", h.config.LemonSqueezy.FrontendBaseURL)

	// Prepare request payload for Lemon Squeezy v1 API
	// Documentation: https://docs.lemonsqueezy.com/api/customer-portals#create-a-customer-portal
	// Lemon Squeezy v1 uses JSON:API format
	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "customer-portals",
			"attributes": map[string]interface{}{
				"return_url": returnURL,
			},
			"relationships": map[string]interface{}{
				"customer": map[string]interface{}{
					"data": map[string]interface{}{
						"type": "customers",
						"id":   customerID,
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
	// Lemon Squeezy v1 API endpoint: https://api.lemonsqueezy.com/v1/customer-portals
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.lemonsqueezy.com/v1/customer-portals", bytes.NewBuffer(jsonPayload))
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
			zap.String("request_id", fmt.Sprintf("%v", requestID)),
		)
		return "", fmt.Errorf("Lemon Squeezy API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	// Parse response - Lemon Squeezy v1 returns JSON:API format
	var lemonResponse struct {
		Data struct {
			Attributes struct {
				URL string `json:"url"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &lemonResponse); err != nil {
		h.logger.Error("Failed to parse Lemon Squeezy response",
			zap.Error(err),
			zap.String("response_body", string(body)),
			zap.String("request_id", fmt.Sprintf("%v", requestID)),
		)
		return "", fmt.Errorf("failed to parse Lemon Squeezy response: %w", err)
	}

	// Return portal URL
	if lemonResponse.Data.Attributes.URL == "" {
		h.logger.Error("Portal URL not found in Lemon Squeezy response",
			zap.String("response_body", string(body)),
			zap.String("request_id", fmt.Sprintf("%v", requestID)),
		)
		return "", fmt.Errorf("portal URL not found in response")
	}

	return lemonResponse.Data.Attributes.URL, nil
}

// CancelSubscriptionRequest represents the request body (empty for now, but extensible)
type CancelSubscriptionRequest struct {
	// Empty - no input required from frontend (subscription_id is retrieved from DB)
}

// CancelSubscriptionResponse represents the response from canceling a subscription
type CancelSubscriptionResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// CancelSubscription cancels a user's subscription via Lemon Squeezy
// POST /api/billing/cancel
// Requires: AuthMiddleware (sets user_id in context)
func (h *BillingHandlers) CancelSubscription(w http.ResponseWriter, r *http.Request) {
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

	h.logger.Info("Canceling subscription",
		zap.String("request_id", fmt.Sprintf("%v", requestID)),
		zap.String("user_id", userID),
	)

	// Step 2: Fetch the user's latest active or past_due subscription from database
	subscription, err := h.subscriptionRepo.GetActiveSubscriptionByUserID(r.Context(), userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.logger.Warn("No subscription found for user",
				zap.String("request_id", fmt.Sprintf("%v", requestID)),
				zap.String("user_id", userID),
			)
			h.writeError(w, http.StatusBadRequest, "No active subscription to cancel")
			return
		}
		h.logger.Error("Failed to get subscription",
			zap.String("request_id", fmt.Sprintf("%v", requestID)),
			zap.String("user_id", userID),
			zap.Error(err),
		)
		h.writeError(w, http.StatusInternalServerError, "Failed to retrieve subscription")
		return
	}

	// Check if subscription is already cancelled or set to cancel at period end
	// Handle gracefully - idempotent operation
	if subscription.Status == "cancelled" || subscription.CancelAtPeriodEnd {
		// Already cancelled or cancellation already initiated - return success (idempotent)
		h.logger.Info("Subscription already cancelled or cancellation already initiated",
			zap.String("request_id", fmt.Sprintf("%v", requestID)),
			zap.String("user_id", userID),
			zap.String("subscription_id", subscription.ID),
			zap.String("status", subscription.Status),
			zap.Bool("cancel_at_period_end", subscription.CancelAtPeriodEnd),
		)
		response := CancelSubscriptionResponse{
			Status:  "ok",
			Message: "Subscription will be cancelled at the end of the billing period",
		}
		h.writeJSON(w, http.StatusOK, response)
		return
	}

	// Step 3: Retrieve lemon_subscription_id
	if subscription.LemonSubscriptionID == nil || *subscription.LemonSubscriptionID == "" {
		h.logger.Error("Missing lemon_subscription_id in subscription",
			zap.String("request_id", fmt.Sprintf("%v", requestID)),
			zap.String("user_id", userID),
			zap.String("subscription_id", subscription.ID),
		)
		h.writeError(w, http.StatusInternalServerError, "Subscription ID not found")
		return
	}

	lemonSubscriptionID := *subscription.LemonSubscriptionID

	h.logger.Info("Retrieved subscription ID",
		zap.String("request_id", fmt.Sprintf("%v", requestID)),
		zap.String("user_id", userID),
		zap.String("subscription_id", subscription.ID),
		zap.String("lemon_subscription_id", lemonSubscriptionID),
	)

	// Step 4: Call Lemon Squeezy "Cancel Subscription" API
	// Cancel at period end (do NOT immediately expire)
	if err := h.cancelLemonSqueezySubscription(r.Context(), lemonSubscriptionID, requestID); err != nil {
		h.logger.Error("Failed to cancel Lemon Squeezy subscription",
			zap.String("request_id", fmt.Sprintf("%v", requestID)),
			zap.String("user_id", userID),
			zap.String("lemon_subscription_id", lemonSubscriptionID),
			zap.Error(err),
		)
		h.writeError(w, http.StatusBadGateway, "Failed to cancel subscription")
		return
	}

	// Step 5: Update local DB - set cancel_at_period_end = true
	// Keep status unchanged until webhook arrives
	// Use transaction for atomic update
	if err := h.subscriptionRepo.SetCancelAtPeriodEnd(r.Context(), subscription.ID, true); err != nil {
		h.logger.Error("Failed to update cancel_at_period_end in database",
			zap.String("request_id", fmt.Sprintf("%v", requestID)),
			zap.String("user_id", userID),
			zap.String("subscription_id", subscription.ID),
			zap.Error(err),
		)
		// Note: Lemon Squeezy cancellation succeeded, but DB update failed
		// Webhook will eventually sync the state, but log the error
		// We still return success since cancellation was initiated
	}

	// Step 6: Return success response
	response := CancelSubscriptionResponse{
		Status:  "ok",
		Message: "Subscription will be cancelled at the end of the billing period",
	}

	h.logger.Info("Subscription cancellation initiated successfully",
		zap.String("request_id", fmt.Sprintf("%v", requestID)),
		zap.String("user_id", userID),
		zap.String("subscription_id", subscription.ID),
		zap.String("lemon_subscription_id", lemonSubscriptionID),
	)

	h.writeJSON(w, http.StatusOK, response)
}

// cancelLemonSqueezySubscription calls the Lemon Squeezy v1 API to cancel a subscription
// Documentation: https://docs.lemonsqueezy.com/api/subscriptions/cancel-subscription
// DELETE cancels the subscription but it remains active until the end of the billing period (ends_at)
// This is the desired behavior - cancel at period end, not immediately
func (h *BillingHandlers) cancelLemonSqueezySubscription(ctx context.Context, lemonSubscriptionID string, requestID interface{}) error {
	// Create HTTP request to Lemon Squeezy API
	// Lemon Squeezy v1 API endpoint: DELETE /v1/subscriptions/{id}
	// This cancels the subscription but keeps it active until period_end
	apiURL := fmt.Sprintf("https://api.lemonsqueezy.com/v1/subscriptions/%s", lemonSubscriptionID)
	req, err := http.NewRequestWithContext(ctx, "DELETE", apiURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	req.Header.Set("Accept", "application/vnd.api+json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", h.config.LemonSqueezy.APIKey))

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Make request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call Lemon Squeezy API: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Check response status
	// Lemon Squeezy returns 200 OK for successful cancellation
	if resp.StatusCode != http.StatusOK {
		h.logger.Error("Lemon Squeezy API returned error",
			zap.Int("status_code", resp.StatusCode),
			zap.String("response_body", string(body)),
			zap.String("request_id", fmt.Sprintf("%v", requestID)),
			zap.String("lemon_subscription_id", lemonSubscriptionID),
		)
		return fmt.Errorf("Lemon Squeezy API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	// Parse response to verify success
	// Lemon Squeezy returns JSON:API format with subscription data
	var lemonResponse struct {
		Data struct {
			ID         string `json:"id"`
			Attributes struct {
				Status    string `json:"status"`
				Cancelled bool   `json:"cancelled"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &lemonResponse); err != nil {
		h.logger.Warn("Failed to parse Lemon Squeezy response (but status was OK)",
			zap.Error(err),
			zap.String("response_body", string(body)),
			zap.String("request_id", fmt.Sprintf("%v", requestID)),
		)
		// Don't fail if we can't parse - status code was OK
	} else {
		h.logger.Info("Lemon Squeezy subscription cancellation confirmed",
			zap.String("request_id", fmt.Sprintf("%v", requestID)),
			zap.String("lemon_subscription_id", lemonSubscriptionID),
			zap.String("status", lemonResponse.Data.Attributes.Status),
			zap.Bool("cancelled", lemonResponse.Data.Attributes.Cancelled),
		)
	}

	return nil
}

// getMapKeys returns all keys from a map[string]string as a slice
func getMapKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
