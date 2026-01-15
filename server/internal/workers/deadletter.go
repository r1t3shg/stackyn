package workers

import (
	"context"
	"time"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

// DeadLetterQueueHandler handles tasks that have exceeded max retries
type DeadLetterQueueHandler struct {
	logger *zap.Logger
	// Add persistence repository here
}

// NewDeadLetterQueueHandler creates a new dead-letter queue handler
func NewDeadLetterQueueHandler(logger *zap.Logger) *DeadLetterQueueHandler {
	return &DeadLetterQueueHandler{
		logger: logger,
	}
}

// HandleDeadLetterTask processes tasks from the dead-letter queue
func (h *DeadLetterQueueHandler) HandleDeadLetterTask(ctx context.Context, t *asynq.Task) error {
	h.logger.Error("Processing dead-letter task",
		zap.String("task_type", t.Type()),
		zap.ByteString("payload", t.Payload()),
	)

	// TODO: Persist to database for manual review
	// TODO: Send alert/notification
	// TODO: Log for monitoring

	// Mark as permanently failed in database
	// This task will remain in dead-letter queue for manual inspection

	return nil
}

// SetupDeadLetterQueue configures dead-letter queue monitoring
func SetupDeadLetterQueue(redisAddr string, redisPassword string, logger *zap.Logger) {
	// Asynq automatically moves failed tasks to dead-letter queue after max retries
	// We can monitor this queue using Asynq's inspector
	
	// Create inspector to monitor dead-letter queue
	inspector := asynq.NewInspector(asynq.RedisClientOpt{
		Addr:     redisAddr,
		Password: redisPassword,
	})

	// Periodically check dead-letter queue
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			// Check dead-letter queue size
			queues, err := inspector.Queues()
			if err != nil {
				logger.Warn("Failed to get queue stats", zap.Error(err))
				continue
			}

			// Log dead-letter queue status
			for _, queueName := range queues {
				queueInfo, err := inspector.GetQueueInfo(queueName)
				if err != nil {
					logger.Warn("Failed to get queue info", zap.String("queue", queueName), zap.Error(err))
					continue
				}

				if queueInfo.Pending > 0 || queueInfo.Active > 0 || queueInfo.Scheduled > 0 || queueInfo.Retry > 0 {
					logger.Info("Queue status",
						zap.String("queue", queueName),
						zap.Int("pending", queueInfo.Pending),
						zap.Int("active", queueInfo.Active),
						zap.Int("scheduled", queueInfo.Scheduled),
						zap.Int("retry", queueInfo.Retry),
					)
				}
			}
		}
	}()

	logger.Info("Dead-letter queue monitoring enabled")
}

