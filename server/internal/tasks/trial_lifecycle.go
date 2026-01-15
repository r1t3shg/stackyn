package tasks

import (
	"context"
	"time"

	"go.uber.org/zap"
	"stackyn/server/internal/services"
)

// TrialLifecycleTask handles daily trial lifecycle processing
// This cron job should run daily to:
// - Expire trials that have ended
// - Send trial ending reminders (at day 6)
type TrialLifecycleTask struct {
	subscriptionService *services.SubscriptionService
	logger              *zap.Logger
}

// NewTrialLifecycleTask creates a new trial lifecycle task
func NewTrialLifecycleTask(subscriptionService *services.SubscriptionService, logger *zap.Logger) *TrialLifecycleTask {
	return &TrialLifecycleTask{
		subscriptionService: subscriptionService,
		logger:              logger,
	}
}

// Run processes trial subscriptions
// Should be called daily (e.g., via cron or scheduled task)
func (t *TrialLifecycleTask) Run(ctx context.Context) error {
	t.logger.Info("Starting trial lifecycle processing")
	
	if err := t.subscriptionService.ProcessTrialLifecycle(ctx); err != nil {
		t.logger.Error("Failed to process trial lifecycle",
			zap.Error(err),
		)
		return err
	}
	
	t.logger.Info("Trial lifecycle processing completed successfully")
	return nil
}

// StartPeriodicTask starts a periodic task that runs daily at the specified hour
// This is a convenience method for running the task periodically
func (t *TrialLifecycleTask) StartPeriodicTask(ctx context.Context, hour int) {
	ticker := time.NewTicker(1 * time.Hour) // Check every hour
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			t.logger.Info("Stopping trial lifecycle periodic task")
			return
		case now := <-ticker.C:
			// Run at the specified hour
			if now.Hour() == hour {
				t.logger.Info("Running scheduled trial lifecycle task",
					zap.Time("time", now),
				)
				if err := t.Run(ctx); err != nil {
					t.logger.Error("Trial lifecycle task failed",
						zap.Error(err),
					)
				}
			}
		}
	}
}

