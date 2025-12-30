package services

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// BillingService handles billing and subscription management
type BillingService struct {
	logger *zap.Logger
	// TODO: Add database repository when DB is connected
	// subscriptionRepo SubscriptionRepository
}

// NewBillingService creates a new billing service
func NewBillingService(logger *zap.Logger) *BillingService {
	return &BillingService{
		logger: logger,
	}
}

// SubscriptionStatus represents the status of a subscription
type SubscriptionStatus string

const (
	SubscriptionStatusActive   SubscriptionStatus = "active"
	SubscriptionStatusInactive SubscriptionStatus = "inactive"
	SubscriptionStatusCanceled SubscriptionStatus = "canceled"
	SubscriptionStatusExpired  SubscriptionStatus = "expired"
	SubscriptionStatusPastDue  SubscriptionStatus = "past_due"
	SubscriptionStatusTrialing SubscriptionStatus = "trialing"
)

// Subscription represents a user's subscription
type Subscription struct {
	UserID        string            `json:"user_id"`
	SubscriptionID string            `json:"subscription_id"` // External subscription ID (e.g., Lemon Squeezy)
	Plan          string            `json:"plan"`              // Plan name/identifier
	Status        SubscriptionStatus `json:"status"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
}

// GetSubscription retrieves a subscription for a user
// TODO: Query database when DB is connected
func (s *BillingService) GetSubscription(ctx context.Context, userID string) (*Subscription, error) {
	// TODO: Query database
	// subscription, err := s.subscriptionRepo.GetByUserID(ctx, userID)
	// if err != nil {
	//     return nil, err
	// }
	// return subscription, nil

	// Placeholder: return default subscription
	return &Subscription{
		UserID:        userID,
		SubscriptionID: "",
		Plan:          "free",
		Status:        SubscriptionStatusActive,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}, nil
}

// UpdateSubscription updates a subscription
// TODO: Update database when DB is connected
func (s *BillingService) UpdateSubscription(ctx context.Context, subscription *Subscription) error {
	// TODO: Update database
	// return s.subscriptionRepo.Update(ctx, subscription)

	s.logger.Info("Subscription updated",
		zap.String("user_id", subscription.UserID),
		zap.String("subscription_id", subscription.SubscriptionID),
		zap.String("plan", subscription.Plan),
		zap.String("status", string(subscription.Status)),
	)

	return nil
}

// CreateSubscription creates a new subscription
// TODO: Insert into database when DB is connected
func (s *BillingService) CreateSubscription(ctx context.Context, subscription *Subscription) error {
	// TODO: Insert into database
	// return s.subscriptionRepo.Create(ctx, subscription)

	s.logger.Info("Subscription created",
		zap.String("user_id", subscription.UserID),
		zap.String("subscription_id", subscription.SubscriptionID),
		zap.String("plan", subscription.Plan),
		zap.String("status", string(subscription.Status)),
	)

	return nil
}

// LemonSqueezyWebhookEvent represents a Lemon Squeezy webhook event
type LemonSqueezyWebhookEvent struct {
	Meta struct {
		EventName string `json:"event_name"`
		CustomData map[string]interface{} `json:"custom_data,omitempty"`
	} `json:"meta"`
	Data struct {
		Type       string                 `json:"type"`
		ID         string                 `json:"id"`
		Attributes map[string]interface{} `json:"attributes"`
	} `json:"data"`
}

// ProcessLemonSqueezyWebhook processes a Lemon Squeezy webhook event
// This is a stub implementation - no actual payment logic
func (s *BillingService) ProcessLemonSqueezyWebhook(ctx context.Context, event *LemonSqueezyWebhookEvent) error {
	s.logger.Info("Processing Lemon Squeezy webhook",
		zap.String("event_name", event.Meta.EventName),
		zap.String("type", event.Data.Type),
		zap.String("id", event.Data.ID),
	)

	// Extract user ID from custom data (Lemon Squeezy allows custom data)
	userID := ""
	if event.Meta.CustomData != nil {
		if uid, ok := event.Meta.CustomData["user_id"].(string); ok {
			userID = uid
		}
	}

	// Handle different event types
	switch event.Meta.EventName {
	case "subscription_created":
		return s.handleSubscriptionCreated(ctx, userID, event)
	case "subscription_updated":
		return s.handleSubscriptionUpdated(ctx, userID, event)
	case "subscription_cancelled":
		return s.handleSubscriptionCancelled(ctx, userID, event)
	case "subscription_resumed":
		return s.handleSubscriptionResumed(ctx, userID, event)
	case "subscription_expired":
		return s.handleSubscriptionExpired(ctx, userID, event)
	case "subscription_payment_success":
		return s.handleSubscriptionPaymentSuccess(ctx, userID, event)
	case "subscription_payment_failed":
		return s.handleSubscriptionPaymentFailed(ctx, userID, event)
	default:
		s.logger.Warn("Unhandled Lemon Squeezy webhook event",
			zap.String("event_name", event.Meta.EventName),
		)
		return nil // Don't fail on unknown events
	}
}

// handleSubscriptionCreated handles subscription creation webhook
func (s *BillingService) handleSubscriptionCreated(ctx context.Context, userID string, event *LemonSqueezyWebhookEvent) error {
	subscriptionID := event.Data.ID
	
	// Extract plan from attributes
	plan := "free"
	if planName, ok := event.Data.Attributes["variant_name"].(string); ok {
		plan = planName
	} else if planID, ok := event.Data.Attributes["variant_id"].(string); ok {
		// Map variant_id to plan name (would need a mapping table)
		plan = planID
	}

	subscription := &Subscription{
		UserID:        userID,
		SubscriptionID: subscriptionID,
		Plan:          plan,
		Status:        SubscriptionStatusActive,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	return s.CreateSubscription(ctx, subscription)
}

// handleSubscriptionUpdated handles subscription update webhook
func (s *BillingService) handleSubscriptionUpdated(ctx context.Context, userID string, event *LemonSqueezyWebhookEvent) error {
	subscriptionID := event.Data.ID
	
	// Get existing subscription
	subscription, err := s.GetSubscription(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	// Update subscription details
	if subscriptionID != "" {
		subscription.SubscriptionID = subscriptionID
	}
	
	// Update plan if changed
	if planName, ok := event.Data.Attributes["variant_name"].(string); ok {
		subscription.Plan = planName
	}

	subscription.UpdatedAt = time.Now()

	return s.UpdateSubscription(ctx, subscription)
}

// handleSubscriptionCancelled handles subscription cancellation webhook
func (s *BillingService) handleSubscriptionCancelled(ctx context.Context, userID string, event *LemonSqueezyWebhookEvent) error {
	subscription, err := s.GetSubscription(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	subscription.Status = SubscriptionStatusCanceled
	subscription.UpdatedAt = time.Now()

	return s.UpdateSubscription(ctx, subscription)
}

// handleSubscriptionResumed handles subscription resumption webhook
func (s *BillingService) handleSubscriptionResumed(ctx context.Context, userID string, event *LemonSqueezyWebhookEvent) error {
	subscription, err := s.GetSubscription(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	subscription.Status = SubscriptionStatusActive
	subscription.UpdatedAt = time.Now()

	return s.UpdateSubscription(ctx, subscription)
}

// handleSubscriptionExpired handles subscription expiration webhook
func (s *BillingService) handleSubscriptionExpired(ctx context.Context, userID string, event *LemonSqueezyWebhookEvent) error {
	subscription, err := s.GetSubscription(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	subscription.Status = SubscriptionStatusExpired
	subscription.UpdatedAt = time.Now()

	return s.UpdateSubscription(ctx, subscription)
}

// handleSubscriptionPaymentSuccess handles successful payment webhook
func (s *BillingService) handleSubscriptionPaymentSuccess(ctx context.Context, userID string, event *LemonSqueezyWebhookEvent) error {
	subscription, err := s.GetSubscription(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	// Ensure subscription is active
	if subscription.Status != SubscriptionStatusActive {
		subscription.Status = SubscriptionStatusActive
		subscription.UpdatedAt = time.Now()
		return s.UpdateSubscription(ctx, subscription)
	}

	s.logger.Info("Subscription payment successful",
		zap.String("user_id", userID),
		zap.String("subscription_id", subscription.SubscriptionID),
	)

	return nil
}

// handleSubscriptionPaymentFailed handles failed payment webhook
func (s *BillingService) handleSubscriptionPaymentFailed(ctx context.Context, userID string, event *LemonSqueezyWebhookEvent) error {
	subscription, err := s.GetSubscription(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	subscription.Status = SubscriptionStatusPastDue
	subscription.UpdatedAt = time.Now()

	return s.UpdateSubscription(ctx, subscription)
}

