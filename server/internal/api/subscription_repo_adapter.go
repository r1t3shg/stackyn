package api

import (
	"context"
	"time"

	"stackyn/server/internal/services"
)

// SubscriptionRepoAdapter adapts the api.SubscriptionRepo to services.SubscriptionRepo interface
type SubscriptionRepoAdapter struct {
	repo *SubscriptionRepo
}

// NewSubscriptionRepoAdapter creates a new adapter
func NewSubscriptionRepoAdapter(repo *SubscriptionRepo) *SubscriptionRepoAdapter {
	return &SubscriptionRepoAdapter{repo: repo}
}

// GetSubscriptionByUserID retrieves a subscription for a user
func (a *SubscriptionRepoAdapter) GetSubscriptionByUserID(ctx context.Context, userID string) (*services.Subscription, error) {
	sub, err := a.repo.GetSubscriptionByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return a.convertSubscription(sub), nil
}

// CreateSubscription creates a new subscription
func (a *SubscriptionRepoAdapter) CreateSubscription(ctx context.Context, userID, lemonSubscriptionID, plan, status string, trialStartedAt, trialEndsAt *time.Time, ramLimitMB, diskLimitGB int) (*services.Subscription, error) {
	sub, err := a.repo.CreateSubscription(ctx, userID, lemonSubscriptionID, plan, status, trialStartedAt, trialEndsAt, ramLimitMB, diskLimitGB)
	if err != nil {
		return nil, err
	}
	return a.convertSubscription(sub), nil
}

// UpdateSubscriptionByUserID updates a user's subscription
func (a *SubscriptionRepoAdapter) UpdateSubscriptionByUserID(ctx context.Context, userID, plan, status string, ramLimitMB, diskLimitGB *int, lemonSubID *string) error {
	return a.repo.UpdateSubscriptionByUserID(ctx, userID, plan, status, ramLimitMB, diskLimitGB, lemonSubID)
}

// GetTrialSubscriptions retrieves all trial subscriptions that need processing
func (a *SubscriptionRepoAdapter) GetTrialSubscriptions(ctx context.Context) ([]*services.Subscription, error) {
	subs, err := a.repo.GetTrialSubscriptions(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*services.Subscription, len(subs))
	for i, sub := range subs {
		result[i] = a.convertSubscription(sub)
	}
	return result, nil
}

// convertSubscription converts api.Subscription to services.Subscription
func (a *SubscriptionRepoAdapter) convertSubscription(sub *Subscription) *services.Subscription {
	return &services.Subscription{
		ID:                 sub.ID,
		UserID:             sub.UserID,
		LemonSubscriptionID: sub.LemonSubscriptionID,
		Plan:               sub.Plan,
		Status:             sub.Status,
		TrialStartedAt:     sub.TrialStartedAt,
		TrialEndsAt:        sub.TrialEndsAt,
		RAMLimitMB:         sub.RAMLimitMB,
		DiskLimitGB:        sub.DiskLimitGB,
		CreatedAt:          sub.CreatedAt,
		UpdatedAt:          sub.UpdatedAt,
	}
}

// UserRepoAdapter adapts the api.UserRepo to services.UserRepository interface
type UserRepoAdapter struct {
	repo *UserRepo
}

// NewUserRepoAdapter creates a new adapter
func NewUserRepoAdapter(repo *UserRepo) *UserRepoAdapter {
	return &UserRepoAdapter{repo: repo}
}

// GetUserByID retrieves a user by ID
func (a *UserRepoAdapter) GetUserByID(userID string) (*services.User, error) {
	user, err := a.repo.GetUserByID(userID)
	if err != nil {
		return nil, err
	}
	return &services.User{
		ID:    user.ID,
		Email: user.Email,
	}, nil
}

// GetUserByEmail retrieves a user by email
func (a *UserRepoAdapter) GetUserByEmail(email string) (*services.User, error) {
	user, err := a.repo.GetUserByEmail(email)
	if err != nil {
		return nil, err
	}
	return &services.User{
		ID:    user.ID,
		Email: user.Email,
	}, nil
}

// UpdateUserBilling implements services.UserBillingUpdater interface
func (a *UserRepoAdapter) UpdateUserBilling(ctx context.Context, userID, billingStatus, plan, subscriptionID string, trialStartedAt, trialEndsAt *time.Time) error {
	return a.repo.UpdateUserBilling(ctx, userID, billingStatus, plan, subscriptionID, trialStartedAt, trialEndsAt)
}

