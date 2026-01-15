package workers

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"stackyn/server/internal/services"
)

// BillingWorker handles trial expiration and billing lifecycle
// Runs every 30 minutes to check for expired trials
type BillingWorker struct {
	pool                *pgxpool.Pool
	subscriptionService *services.SubscriptionService
	logger              *zap.Logger
	interval            time.Duration
}

// NewBillingWorker creates a new billing worker
func NewBillingWorker(pool *pgxpool.Pool, subscriptionService *services.SubscriptionService, logger *zap.Logger) *BillingWorker {
	return &BillingWorker{
		pool:                pool,
		subscriptionService: subscriptionService,
		logger:              logger,
		interval:            30 * time.Minute, // Run every 30 minutes
	}
}

// Start starts the billing worker
// It runs in a loop, checking for expired trials every 30 minutes
func (w *BillingWorker) Start(ctx context.Context) error {
	w.logger.Info("Starting billing worker", zap.Duration("interval", w.interval))

	// Run immediately on startup, then every interval
	if err := w.processExpiredTrials(ctx); err != nil {
		w.logger.Error("Failed to process expired trials on startup", zap.Error(err))
		// Continue anyway - don't fail startup
	}

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Billing worker stopped")
			return ctx.Err()
		case <-ticker.C:
			if err := w.processExpiredTrials(ctx); err != nil {
				w.logger.Error("Failed to process expired trials", zap.Error(err))
				// Continue - don't stop worker on error
			}
		}
	}
}

// processExpiredTrials processes users with expired trials
// Queries users table directly as specified in requirements
func (w *BillingWorker) processExpiredTrials(ctx context.Context) error {
	w.logger.Info("Processing expired trials")

	// Query users table directly: WHERE billing_status = 'trial' AND trial_ends_at < NOW()
	rows, err := w.pool.Query(ctx,
		`SELECT id, email, billing_status, plan, trial_started_at, trial_ends_at 
		 FROM users 
		 WHERE billing_status = 'trial' 
		   AND trial_ends_at IS NOT NULL 
		   AND trial_ends_at < NOW()`,
	)
	if err != nil {
		return fmt.Errorf("failed to query expired trials: %w", err)
	}
	defer rows.Close()

	var expiredUsers []struct {
		ID             string
		Email          string
		BillingStatus  string
		Plan           string
		TrialStartedAt *time.Time
		TrialEndsAt    *time.Time
	}

	for rows.Next() {
		var user struct {
			ID             string
			Email          string
			BillingStatus  string
			Plan           string
			TrialStartedAt *time.Time
			TrialEndsAt    *time.Time
		}
		var trialStartedAt, trialEndsAt sql.NullTime

		if err := rows.Scan(&user.ID, &user.Email, &user.BillingStatus, &user.Plan, &trialStartedAt, &trialEndsAt); err != nil {
			w.logger.Error("Failed to scan expired trial user", zap.Error(err))
			continue
		}

		if trialStartedAt.Valid {
			user.TrialStartedAt = &trialStartedAt.Time
		}
		if trialEndsAt.Valid {
			user.TrialEndsAt = &trialEndsAt.Time
		}

		expiredUsers = append(expiredUsers, user)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating expired trial users: %w", err)
	}

	w.logger.Info("Found expired trials", zap.Int("count", len(expiredUsers)))

	// Process each expired user
	for _, user := range expiredUsers {
		// Use subscription service to expire trial (handles stopping apps, sending emails, etc.)
		if err := w.subscriptionService.ExpireTrial(ctx, user.ID, user.Email); err != nil {
			w.logger.Error("Failed to expire trial for user",
				zap.Error(err),
				zap.String("user_id", user.ID),
				zap.String("user_email", user.Email),
			)
			// Continue processing other users
			continue
		}

		w.logger.Info("Expired trial for user",
			zap.String("user_id", user.ID),
			zap.String("user_email", user.Email),
		)
	}

	w.logger.Info("Trial expiration processing completed", zap.Int("processed", len(expiredUsers)))
	return nil
}

