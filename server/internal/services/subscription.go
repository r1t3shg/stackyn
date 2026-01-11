package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
	"stackyn/server/internal/api"
)

// SubscriptionService handles subscription and trial management
type SubscriptionService struct {
	subscriptionRepo api.SubscriptionRepo
	emailService     *EmailService
	userRepo         api.UserRepository
	logger           *zap.Logger
}

// SubscriptionRepo interface for subscription repository operations
// This interface allows us to use the repository from the api package
type SubscriptionRepo interface {
	GetSubscriptionByUserID(ctx context.Context, userID string) (*api.Subscription, error)
	CreateSubscription(ctx context.Context, userID, lemonSubscriptionID, plan, status string, trialStartedAt, trialEndsAt *time.Time, ramLimitMB, diskLimitGB int) (*api.Subscription, error)
	UpdateSubscriptionByUserID(ctx context.Context, userID, plan, status string, ramLimitMB, diskLimitGB *int, lemonSubID *string) error
	GetTrialSubscriptions(ctx context.Context) ([]*api.Subscription, error)
}

// UserRepository interface for user operations
// This matches the UserRepository from api package
type UserRepository interface {
	GetUserByID(userID string) (*api.User, error)
	GetUserByEmail(email string) (*api.User, error)
}

// NewSubscriptionService creates a new subscription service
func NewSubscriptionService(
	subscriptionRepo SubscriptionRepo,
	emailService *EmailService,
	userRepo api.UserRepository,
	logger *zap.Logger,
) *SubscriptionService {
	return &SubscriptionService{
		subscriptionRepo: subscriptionRepo,
		emailService:     emailService,
		userRepo:         userRepo,
		logger:           logger,
	}
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
func (s *SubscriptionService) GetSubscriptionByUserID(ctx context.Context, userID string) (*api.Subscription, error) {
	return s.subscriptionRepo.GetSubscriptionByUserID(ctx, userID)
}

// IsSubscriptionActive checks if a subscription allows deployments
// Returns true if status is "trial" or "active"
func (s *SubscriptionService) IsSubscriptionActive(sub *api.Subscription) bool {
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
		"",    // Don't change plan
		"expired",
		nil,   // Don't change RAM limit
		nil,   // Don't change disk limit
		nil,   // Don't change lemon_subscription_id
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
		"",    // Don't change plan
		"cancelled",
		nil,   // Don't change RAM limit
		nil,   // Don't change disk limit
		nil,   // Don't change lemon_subscription_id
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

// CheckResourceLimits checks if user's total resource usage is within subscription limits
// Returns error if limits are exceeded
func (s *SubscriptionService) CheckResourceLimits(ctx context.Context, userID string, currentRAMMB, currentDiskGB int, newAppRAMMB, newAppDiskGB int) error {
	sub, err := s.subscriptionRepo.GetSubscriptionByUserID(ctx, userID)
	if err != nil {
		// If no subscription found, deny access (should not happen with trial)
		return fmt.Errorf("subscription not found for user %s", userID)
	}

	// Check subscription status - only allow deployments for trial or active
	if !s.IsSubscriptionActive(sub) {
		return fmt.Errorf("subscription is not active (status: %s). Upgrade to continue", sub.Status)
	}

	// Calculate total usage after adding new app
	totalRAMMB := currentRAMMB + newAppRAMMB
	totalDiskGB := currentDiskGB + newAppDiskGB

	// Check RAM limit
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

