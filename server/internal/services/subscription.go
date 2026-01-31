package services

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// AppStopper interface for stopping user apps
type AppStopper interface {
	StopAllUserApps(ctx context.Context, userID string) error
}

// SubscriptionService handles subscription and trial management
type SubscriptionService struct {
	subscriptionRepo SubscriptionRepo
	emailService     *EmailService
	userRepo         UserRepository
	billingUpdater   UserBillingUpdater // Optional - for syncing billing fields to users table
	appStopper       AppStopper         // Optional - for stopping apps when trial expires
	logger           *zap.Logger
}

// Subscription represents a subscription from the database
type Subscription struct {
	ID                  string
	UserID              string
	LemonSubscriptionID *string    // nullable
	Plan                string     // starter | pro
	Status              string     // trial | active | expired | cancelled
	TrialStartedAt      *time.Time // nullable
	TrialEndsAt         *time.Time // nullable
	RAMLimitMB          int
	DiskLimitGB         int
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// User represents a user (minimal fields needed for subscription service)
type User struct {
	ID    string
	Email string
}

// SubscriptionRepo interface for subscription repository operations
type SubscriptionRepo interface {
	GetSubscriptionByUserID(ctx context.Context, userID string) (*Subscription, error)
	CreateSubscription(ctx context.Context, userID, lemonSubscriptionID, lemonCustomerID, plan, status string, trialStartedAt, trialEndsAt *time.Time, ramLimitMB, diskLimitGB int) (*Subscription, error)
	UpdateSubscriptionByUserID(ctx context.Context, userID, plan, status string, ramLimitMB, diskLimitGB *int, lemonSubID *string) error
	GetTrialSubscriptions(ctx context.Context) ([]*Subscription, error)
}

// UserRepository interface for user operations
type UserRepository interface {
	GetUserByID(userID string) (*User, error)
	GetUserByEmail(email string) (*User, error)
}

// UserBillingUpdater interface for updating user billing fields
// This allows the subscription service to sync billing status to users table
type UserBillingUpdater interface {
	UpdateUserBilling(ctx context.Context, userID, billingStatus, plan, subscriptionID string, trialStartedAt, trialEndsAt *time.Time) error
}

// NewSubscriptionService creates a new subscription service
func NewSubscriptionService(
	subscriptionRepo SubscriptionRepo,
	emailService *EmailService,
	userRepo UserRepository,
	logger *zap.Logger,
) *SubscriptionService {
	return &SubscriptionService{
		subscriptionRepo: subscriptionRepo,
		emailService:     emailService,
		userRepo:         userRepo,
		billingUpdater:   nil, // Can be set later if needed
		appStopper:       nil, // Can be set later if needed
		logger:           logger,
	}
}

// SetBillingUpdater sets the billing updater for syncing billing fields to users table
func (s *SubscriptionService) SetBillingUpdater(updater UserBillingUpdater) {
	s.billingUpdater = updater
}

// SetAppStopper sets the app stopper for stopping apps when trial expires
func (s *SubscriptionService) SetAppStopper(appStopper AppStopper) {
	s.appStopper = appStopper
}

// CreateTrial creates a 7-day free trial for a new user
// Trial defaults to Pro plan limits (2GB RAM / 20GB Disk)
func (s *SubscriptionService) CreateTrial(ctx context.Context, userID, userEmail string) error {
	now := time.Now()
	trialEndsAt := now.Add(7 * 24 * time.Hour) // 7 days from now

	// Trial uses Pro plan limits: 2GB RAM, 20GB Disk
	subscription, err := s.subscriptionRepo.CreateSubscription(
		ctx,
		userID,
		"",           // No lemon_subscription_id for trials
		"",           // No lemon_customer_id for trials
		"pro",        // Plan name (trial gets Pro features)
		"trial",      // Status
		&now,         // trial_started_at
		&trialEndsAt, // trial_ends_at
		2048,         // 2GB RAM in MB
		20,           // 20GB Disk
	)
	if err != nil {
		s.logger.Error("Failed to create trial subscription",
			zap.Error(err),
			zap.String("user_id", userID),
		)
		return fmt.Errorf("failed to create trial: %w", err)
	}

	s.logger.Info("Trial subscription created",
		zap.String("user_id", userID),
		zap.String("subscription_id", subscription.ID),
		zap.Time("trial_ends_at", trialEndsAt),
	)

	// Sync billing fields to users table (non-blocking)
	if s.billingUpdater != nil {
		if err := s.billingUpdater.UpdateUserBilling(ctx, userID, "trial", "free_trial", "", &now, &trialEndsAt); err != nil {
			s.logger.Warn("Failed to sync billing fields to users table",
				zap.Error(err),
				zap.String("user_id", userID),
			)
			// Non-critical - subscription table is source of truth
		}
	}

	// Send trial started email (non-blocking - don't fail signup if email fails)
	go func() {
		if err := s.emailService.SendTrialStartedEmail(userEmail, trialEndsAt); err != nil {
			s.logger.Warn("Failed to send trial started email",
				zap.Error(err),
				zap.String("user_email", userEmail),
			)
			// Email failure should NOT block signup
		} else {
			s.logger.Info("Trial started email sent",
				zap.String("user_email", userEmail),
			)
		}
	}()

	return nil
}

// GetSubscriptionByUserID retrieves a user's subscription
func (s *SubscriptionService) GetSubscriptionByUserID(ctx context.Context, userID string) (*Subscription, error) {
	return s.subscriptionRepo.GetSubscriptionByUserID(ctx, userID)
}

// IsSubscriptionActive checks if a subscription allows deployments
// Returns true if status is "trial" or "active"
func (s *SubscriptionService) IsSubscriptionActive(sub *Subscription) bool {
	if sub == nil {
		return false
	}
	return sub.Status == "trial" || sub.Status == "active"
}

// GetPlanLimits returns RAM and disk limits for a plan
func GetPlanLimits(planName string) (ramLimitMB, diskLimitGB int) {
	switch planName {
	case "starter":
		return 512, 5 // 512 MB RAM, 5 GB Disk
	case "pro":
		return 2048, 20 // 2 GB RAM, 20 GB Disk
	default:
		// Default to starter limits for unknown plans
		return 512, 5
	}
}

// ActivateSubscription activates a subscription from trial or webhook
func (s *SubscriptionService) ActivateSubscription(ctx context.Context, userID, plan, lemonSubID string, ramLimitMB, diskLimitGB int, userEmail string) error {
	err := s.subscriptionRepo.UpdateSubscriptionByUserID(
		ctx,
		userID,
		plan,
		"active",
		&ramLimitMB,
		&diskLimitGB,
		&lemonSubID,
	)
	if err != nil {
		s.logger.Error("Failed to activate subscription",
			zap.Error(err),
			zap.String("user_id", userID),
			zap.String("plan", plan),
		)
		return fmt.Errorf("failed to activate subscription: %w", err)
	}

	s.logger.Info("Subscription activated",
		zap.String("user_id", userID),
		zap.String("plan", plan),
		zap.String("lemon_sub_id", lemonSubID),
	)

	// Sync billing fields to users table (non-blocking)
	if s.billingUpdater != nil {
		if err := s.billingUpdater.UpdateUserBilling(ctx, userID, "active", plan, lemonSubID, nil, nil); err != nil {
			s.logger.Warn("Failed to sync billing fields to users table",
				zap.Error(err),
				zap.String("user_id", userID),
			)
			// Non-critical - subscription table is source of truth
		}
	}

	// Send subscription activated email (non-blocking)
	if userEmail != "" {
		go func() {
			if err := s.emailService.SendSubscriptionActivatedEmail(userEmail, plan, ramLimitMB, diskLimitGB); err != nil {
				s.logger.Warn("Failed to send subscription activated email",
					zap.Error(err),
					zap.String("user_email", userEmail),
				)
			} else {
				s.logger.Info("Subscription activated email sent",
					zap.String("user_email", userEmail),
				)
			}
		}()
	}

	return nil
}

// ExpireTrial expires a trial subscription
func (s *SubscriptionService) ExpireTrial(ctx context.Context, userID, userEmail string) error {
	err := s.subscriptionRepo.UpdateSubscriptionByUserID(
		ctx,
		userID,
		"", // Don't change plan
		"expired",
		nil, // Don't change RAM limit
		nil, // Don't change disk limit
		nil, // Don't change lemon_subscription_id
	)
	if err != nil {
		s.logger.Error("Failed to expire trial",
			zap.Error(err),
			zap.String("user_id", userID),
		)
		return fmt.Errorf("failed to expire trial: %w", err)
	}

	s.logger.Info("Trial expired",
		zap.String("user_id", userID),
	)

	// Sync billing fields to users table (non-blocking)
	if s.billingUpdater != nil {
		// Get current subscription to preserve plan
		sub, err := s.subscriptionRepo.GetSubscriptionByUserID(ctx, userID)
		plan := ""
		if err == nil && sub != nil {
			plan = sub.Plan
		}
		if err := s.billingUpdater.UpdateUserBilling(ctx, userID, "expired", plan, "", nil, nil); err != nil {
			s.logger.Warn("Failed to sync billing fields to users table",
				zap.Error(err),
				zap.String("user_id", userID),
			)
			// Non-critical - subscription table is source of truth
		}
	}

	// Stop all running apps for the user (blocking - must complete before continuing)
	if s.appStopper != nil {
		if err := s.appStopper.StopAllUserApps(ctx, userID); err != nil {
			s.logger.Warn("Failed to stop user apps after trial expiration",
				zap.Error(err),
				zap.String("user_id", userID),
			)
			// Continue anyway - don't fail trial expiration if app stopping fails
		} else {
			s.logger.Info("Stopped all user apps after trial expiration",
				zap.String("user_id", userID),
			)
		}
	} else {
		s.logger.Warn("AppStopper not set - apps will not be stopped automatically",
			zap.String("user_id", userID),
		)
	}

	// Send trial expired email (non-blocking)
	if userEmail != "" {
		go func() {
			if err := s.emailService.SendTrialExpiredEmail(userEmail); err != nil {
				s.logger.Warn("Failed to send trial expired email",
					zap.Error(err),
					zap.String("user_email", userEmail),
				)
			} else {
				s.logger.Info("Trial expired email sent",
					zap.String("user_email", userEmail),
				)
			}
		}()
	}

	return nil
}

// CancelSubscription cancels a subscription
func (s *SubscriptionService) CancelSubscription(ctx context.Context, userID string) error {
	err := s.subscriptionRepo.UpdateSubscriptionByUserID(
		ctx,
		userID,
		"", // Don't change plan
		"cancelled",
		nil, // Don't change RAM limit
		nil, // Don't change disk limit
		nil, // Don't change lemon_subscription_id
	)
	if err != nil {
		s.logger.Error("Failed to cancel subscription",
			zap.Error(err),
			zap.String("user_id", userID),
		)
		return fmt.Errorf("failed to cancel subscription: %w", err)
	}

	s.logger.Info("Subscription cancelled",
		zap.String("user_id", userID),
	)

	return nil
}

// ExpireSubscription expires a subscription (payment failed or subscription expired)
// Stops all apps and sends email notification
func (s *SubscriptionService) ExpireSubscription(ctx context.Context, userID, userEmail string) error {
	err := s.subscriptionRepo.UpdateSubscriptionByUserID(
		ctx,
		userID,
		"", // Don't change plan
		"expired",
		nil, // Don't change RAM limit
		nil, // Don't change disk limit
		nil, // Don't change lemon_subscription_id
	)
	if err != nil {
		s.logger.Error("Failed to expire subscription",
			zap.Error(err),
			zap.String("user_id", userID),
		)
		return fmt.Errorf("failed to expire subscription: %w", err)
	}

	s.logger.Info("Subscription expired",
		zap.String("user_id", userID),
	)

	// Sync billing fields to users table (non-blocking)
	if s.billingUpdater != nil {
		// Get current subscription to preserve plan
		sub, err := s.subscriptionRepo.GetSubscriptionByUserID(ctx, userID)
		plan := ""
		subscriptionID := ""
		if err == nil && sub != nil {
			plan = sub.Plan
			if sub.LemonSubscriptionID != nil {
				subscriptionID = *sub.LemonSubscriptionID
			}
		}
		if err := s.billingUpdater.UpdateUserBilling(ctx, userID, "expired", plan, subscriptionID, nil, nil); err != nil {
			s.logger.Warn("Failed to sync billing fields to users table",
				zap.Error(err),
				zap.String("user_id", userID),
			)
			// Non-critical - subscription table is source of truth
		}
	}

	// Stop all running apps for the user (blocking - must complete)
	if s.appStopper != nil {
		if err := s.appStopper.StopAllUserApps(ctx, userID); err != nil {
			s.logger.Warn("Failed to stop user apps after subscription expiration",
				zap.Error(err),
				zap.String("user_id", userID),
			)
			// Continue anyway - don't fail expiration if app stopping fails
		} else {
			s.logger.Info("Stopped all user apps after subscription expiration",
				zap.String("user_id", userID),
			)
		}
	} else {
		s.logger.Warn("AppStopper not set - apps will not be stopped automatically",
			zap.String("user_id", userID),
		)
	}

	// Send payment failed email (non-blocking)
	if userEmail != "" {
		go func() {
			if err := s.emailService.SendPaymentFailedEmail(userEmail); err != nil {
				s.logger.Warn("Failed to send payment failed email",
					zap.Error(err),
					zap.String("user_email", userEmail),
				)
			} else {
				s.logger.Info("Payment failed email sent",
					zap.String("user_email", userEmail),
				)
			}
		}()
	}

	return nil
}

// IsTrialExpired checks if user's trial has expired
func (s *SubscriptionService) IsTrialExpired(ctx context.Context, userID string) (bool, error) {
	sub, err := s.subscriptionRepo.GetSubscriptionByUserID(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("failed to get subscription: %w", err)
	}

	if sub == nil {
		// No subscription - treat as expired
		return true, nil
	}

	// Check if status is explicitly expired
	if sub.Status == "expired" {
		return true, nil
	}

	// Check if trial has passed its end date
	if sub.Status == "trial" {
		if sub.TrialEndsAt == nil {
			// Trial without end date - treat as expired for safety
			return true, nil
		}
		if time.Now().After(*sub.TrialEndsAt) || time.Now().Equal(*sub.TrialEndsAt) {
			return true, nil
		}
	}

	return false, nil
}

// CheckResourceLimits checks if user's total resource usage is within subscription limits
// Returns error if limits are exceeded
func (s *SubscriptionService) CheckResourceLimits(ctx context.Context, userID string, currentRAMMB, currentDiskGB int, newAppRAMMB, newAppDiskGB int) error {
	sub, err := s.subscriptionRepo.GetSubscriptionByUserID(ctx, userID)
	if err != nil {
		// If no subscription found, automatically create a trial
		s.logger.Warn("No subscription found for user, creating trial automatically",
			zap.String("user_id", userID),
			zap.Error(err),
		)

		// Get user email
		user, userErr := s.userRepo.GetUserByID(userID)
		if userErr != nil {
			return fmt.Errorf("subscription not found and failed to get user: %w", userErr)
		}

		// Create trial
		if createErr := s.CreateTrial(ctx, userID, user.Email); createErr != nil {
			// Check if error is due to unique constraint, try to get existing
			if retrySub, retryErr := s.subscriptionRepo.GetSubscriptionByUserID(ctx, userID); retryErr == nil {
				sub = retrySub
			} else {
				return fmt.Errorf("failed to create trial: %w", createErr)
			}
		} else {
			// Get created subscription
			sub, err = s.subscriptionRepo.GetSubscriptionByUserID(ctx, userID)
			if err != nil {
				return fmt.Errorf("subscription not found after trial creation: %w", err)
			}
		}
	}

	// Double check subscription is valid now
	if sub == nil {
		return fmt.Errorf("failed to retrieve or create subscription")
	}

	// Check status
	if !s.IsSubscriptionActive(sub) {
		return fmt.Errorf("subscription is not active (status: %s). Upgrade to continue", sub.Status)
	}

	// Calculate total usage
	totalRAMMB := currentRAMMB + newAppRAMMB
	totalDiskGB := currentDiskGB + newAppDiskGB

	// Check RAM limit
	// Use limits from the subscription record, which we now know are correct (e.g. 2048 for Pro)
	if totalRAMMB > sub.RAMLimitMB {
		return fmt.Errorf("plan limit exceeded. Total RAM usage (%d MB) exceeds limit (%d MB). Upgrade to continue", totalRAMMB, sub.RAMLimitMB)
	}

	// Check disk limit
	if totalDiskGB > sub.DiskLimitGB {
		return fmt.Errorf("plan limit exceeded. Total disk usage (%d GB) exceeds limit (%d GB). Upgrade to continue", totalDiskGB, sub.DiskLimitGB)
	}

	return nil
}

// ProcessTrialLifecycle processes trial subscriptions for expiration and reminders
// This is called by the cron job daily
func (s *SubscriptionService) ProcessTrialLifecycle(ctx context.Context) error {
	now := time.Now()
	trialEndingThreshold := now.Add(24 * time.Hour) // 24 hours from now

	// Get all trial subscriptions
	trialSubs, err := s.subscriptionRepo.GetTrialSubscriptions(ctx)
	if err != nil {
		return fmt.Errorf("failed to get trial subscriptions: %w", err)
	}

	s.logger.Info("Processing trial lifecycle",
		zap.Int("trial_count", len(trialSubs)),
	)

	for _, sub := range trialSubs {
		if sub.TrialEndsAt == nil {
			s.logger.Warn("Trial subscription missing trial_ends_at",
				zap.String("subscription_id", sub.ID),
				zap.String("user_id", sub.UserID),
			)
			continue
		}

		// Get user email for notifications
		user, err := s.userRepo.GetUserByID(sub.UserID)
		if err != nil {
			s.logger.Warn("Failed to get user for trial processing",
				zap.Error(err),
				zap.String("user_id", sub.UserID),
			)
			// Continue processing other trials even if one user lookup fails
			// Expire trial anyway even if we can't send email
			if now.After(*sub.TrialEndsAt) || now.Equal(*sub.TrialEndsAt) {
				if err := s.ExpireTrial(ctx, sub.UserID, ""); err != nil {
					s.logger.Error("Failed to expire trial",
						zap.Error(err),
						zap.String("user_id", sub.UserID),
					)
				}
			}
			continue
		}

		// Check if trial has expired
		if now.After(*sub.TrialEndsAt) || now.Equal(*sub.TrialEndsAt) {
			// Trial expired - update status
			if err := s.ExpireTrial(ctx, sub.UserID, user.Email); err != nil {
				s.logger.Error("Failed to expire trial",
					zap.Error(err),
					zap.String("user_id", sub.UserID),
					zap.String("subscription_id", sub.ID),
				)
				// Continue processing other trials
				continue
			}
			s.logger.Info("Trial expired successfully",
				zap.String("user_id", sub.UserID),
				zap.String("subscription_id", sub.ID),
			)
		} else if sub.TrialEndsAt.Before(trialEndingThreshold) && sub.TrialEndsAt.After(now) {
			// Trial ending soon (within 24 hours) - send reminder
			// Note: Email idempotency is handled by checking if email was already sent
			// For MVP, we send reminder daily until trial expires
			// TODO: Add email_sent flag to subscriptions table for better idempotency
			go func(email string, endsAt time.Time) {
				if err := s.emailService.SendTrialEndingEmail(email, endsAt); err != nil {
					s.logger.Warn("Failed to send trial ending email",
						zap.Error(err),
						zap.String("user_email", email),
					)
				} else {
					s.logger.Info("Trial ending email sent",
						zap.String("user_email", email),
					)
				}
			}(user.Email, *sub.TrialEndsAt)
		}
	}

	return nil
}
