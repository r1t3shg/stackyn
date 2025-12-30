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

	// Map priority to Asynq queue
	// Higher priority number = higher priority queue
	var queueName string
	switch {
	case priority >= 10:
		queueName = "critical" // Premium plans
	case priority >= 5:
		queueName = "default" // Pro plans
	default:
		queueName = "low" // Free plans
	}

	// Serialize payload
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create task
	task := asynq.NewTask("build_task", payloadBytes)

	// Enqueue with priority
	info, err := s.client.Enqueue(task, asynq.Queue(queueName))
	if err != nil {
		return nil, fmt.Errorf("failed to enqueue build task: %w", err)
	}

	s.logger.Info("Enqueued build task",
		zap.String("task_id", info.ID),
		zap.String("queue", queueName),
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

	// Map priority to Asynq queue
	var queueName string
	switch {
	case priority >= 10:
		queueName = "critical"
	case priority >= 5:
		queueName = "default"
	default:
		queueName = "low"
	}

	// Serialize payload
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create task
	task := asynq.NewTask("deploy_task", payloadBytes)

	// Enqueue with priority
	info, err := s.client.Enqueue(task, asynq.Queue(queueName))
	if err != nil {
		return nil, fmt.Errorf("failed to enqueue deploy task: %w", err)
	}

	s.logger.Info("Enqueued deploy task",
		zap.String("task_id", info.ID),
		zap.String("queue", queueName),
		zap.Int("priority", priority),
		zap.String("user_id", userID),
	)

	return info, nil
}

