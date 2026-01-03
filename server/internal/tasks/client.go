package tasks

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

// TaskClient wraps Asynq client for task enqueueing
type TaskClient struct {
	client *asynq.Client
	logger *zap.Logger
}

// NewTaskClient creates a new task client
func NewTaskClient(redisAddr string, logger *zap.Logger) *TaskClient {
	redisOpt := asynq.RedisClientOpt{
		Addr: redisAddr,
	}

	return &TaskClient{
		client: asynq.NewClient(redisOpt),
		logger: logger,
	}
}

// Close closes the task client
func (c *TaskClient) Close() error {
	return c.client.Close()
}

// EnqueueBuildTask enqueues a build task with retries, backoff, and priority
func (c *TaskClient) EnqueueBuildTask(payload BuildTaskPayload, priority int) (*asynq.TaskInfo, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal build task payload: %w", err)
	}

	task := asynq.NewTask(TypeBuildTask, payloadBytes)

	// Configure task options
	opts := []asynq.Option{
		asynq.MaxRetry(0),                         // No retries - try once only
		asynq.Timeout(30 * time.Minute),           // 30 minute timeout
		asynq.Queue(getQueueByPriority(priority)), // Priority-based queue
	}

	taskInfo, err := c.client.Enqueue(task, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to enqueue build task: %w", err)
	}

	c.logger.Info("Build task enqueued",
		zap.String("task_id", taskInfo.ID),
		zap.String("app_id", payload.AppID),
		zap.Int("priority", priority),
	)

	return taskInfo, nil
}

// EnqueueDeployTask enqueues a deploy task with retries, backoff, and priority
func (c *TaskClient) EnqueueDeployTask(payload DeployTaskPayload, priority int) (*asynq.TaskInfo, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal deploy task payload: %w", err)
	}

	task := asynq.NewTask(TypeDeployTask, payloadBytes)

	// Configure task options
	opts := []asynq.Option{
		asynq.MaxRetry(3),                         // Retry up to 3 times
		asynq.Timeout(15 * time.Minute),          // 15 minute timeout
		asynq.Queue(getQueueByPriority(priority)), // Priority-based queue
		// Exponential backoff is configured at server level
	}

	taskInfo, err := c.client.Enqueue(task, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to enqueue deploy task: %w", err)
	}

	c.logger.Info("Deploy task enqueued",
		zap.String("task_id", taskInfo.ID),
		zap.String("app_id", payload.AppID),
		zap.Int("priority", priority),
	)

	return taskInfo, nil
}

// EnqueueCleanupTask enqueues a cleanup task with retries, backoff, and priority
func (c *TaskClient) EnqueueCleanupTask(payload CleanupTaskPayload, priority int) (*asynq.TaskInfo, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal cleanup task payload: %w", err)
	}

	task := asynq.NewTask(TypeCleanupTask, payloadBytes)

	// Configure task options
	opts := []asynq.Option{
		asynq.MaxRetry(2),                         // Retry up to 2 times
		asynq.Timeout(10 * time.Minute),         // 10 minute timeout
		asynq.Queue(getQueueByPriority(priority)), // Priority-based queue
		// Exponential backoff is configured at server level
	}

	taskInfo, err := c.client.Enqueue(task, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to enqueue cleanup task: %w", err)
	}

	c.logger.Info("Cleanup task enqueued",
		zap.String("task_id", taskInfo.ID),
		zap.String("app_id", payload.AppID),
		zap.Int("priority", priority),
	)

	return taskInfo, nil
}

// getQueueByPriority returns queue name based on priority
// Priority: 0-3 = low, 4-7 = default, 8-10 = critical
func getQueueByPriority(priority int) string {
	if priority >= 8 {
		return QueueCritical
	} else if priority >= 4 {
		return QueueDefault
	}
	return QueueLow
}

