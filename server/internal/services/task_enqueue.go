package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

// TaskEnqueueService handles enqueueing tasks with plan-based priority
type TaskEnqueueService struct {
	client          *asynq.Client
	logger          *zap.Logger
	planEnforcement *PlanEnforcementService
}

// NewTaskEnqueueService creates a new task enqueue service
func NewTaskEnqueueService(redisAddr string, logger *zap.Logger, planEnforcement *PlanEnforcementService) (*TaskEnqueueService, error) {
	redisOpt := asynq.RedisClientOpt{
		Addr: redisAddr,
	}

	client := asynq.NewClient(redisOpt)

	return &TaskEnqueueService{
		client:          client,
		logger:          logger,
		planEnforcement: planEnforcement,
	}, nil
}

// Close closes the Asynq client
func (s *TaskEnqueueService) Close() error {
	return s.client.Close()
}

// EnqueueBuildTask enqueues a build task with plan-based priority
func (s *TaskEnqueueService) EnqueueBuildTask(ctx context.Context, payload interface{}, userID string) (*asynq.TaskInfo, error) {
	// Get queue priority based on user's plan
	priority, err := s.planEnforcement.GetQueuePriority(ctx, userID)
	if err != nil {
		s.logger.Warn("Failed to get queue priority, using default", zap.Error(err))
		priority = 1 // Default priority
	}

	// Serialize payload
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create task
	task := asynq.NewTask("build_task", payloadBytes)

	// Use build-specific queue to ensure only build-worker processes it
	// Builds should only start when explicitly triggered by user (CreateApp or RedeployApp)
	info, err := s.client.Enqueue(task, 
		asynq.Queue("build"), // Use build-specific queue
		asynq.MaxRetry(0),    // No automatic retries - user must manually trigger redeploy
	)
	if err != nil {
		return nil, fmt.Errorf("failed to enqueue build task: %w", err)
	}

	s.logger.Info("Enqueued build task",
		zap.String("task_id", info.ID),
		zap.String("queue", "build"),
		zap.Int("priority", priority),
		zap.String("user_id", userID),
	)

	return info, nil
}

// EnqueueDeployTask enqueues a deploy task with plan-based priority
func (s *TaskEnqueueService) EnqueueDeployTask(ctx context.Context, payload interface{}, userID string) (*asynq.TaskInfo, error) {
	// Get queue priority based on user's plan
	priority, err := s.planEnforcement.GetQueuePriority(ctx, userID)
	if err != nil {
		s.logger.Warn("Failed to get queue priority, using default", zap.Error(err))
		priority = 1 // Default priority
	}

	// Serialize payload
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create task
	task := asynq.NewTask("deploy_task", payloadBytes)

	// Use deploy-specific queue to ensure only deploy-worker processes it
	info, err := s.client.Enqueue(task, asynq.Queue("deploy"))
	if err != nil {
		return nil, fmt.Errorf("failed to enqueue deploy task: %w", err)
	}

	s.logger.Info("Enqueued deploy task",
		zap.String("task_id", info.ID),
		zap.String("queue", "deploy"),
		zap.Int("priority", priority),
		zap.String("user_id", userID),
	)

	return info, nil
}

