package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// TaskStateRepository handles task state persistence
type TaskStateRepository interface {
	CreateTaskState(ctx context.Context, taskID, taskType, queueName string, payload interface{}, maxRetries int) error
	UpdateTaskState(ctx context.Context, taskID, status string, retryCount int, errorMsg string) error
	GetTaskState(ctx context.Context, taskID string) (*TaskState, error)
}

// TaskState represents a task state in the database
type TaskState struct {
	ID          string
	TaskID      string
	TaskType    string
	QueueName   string
	Payload     json.RawMessage
	Status      string
	RetryCount  int
	MaxRetries  int
	ErrorMessage string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	CompletedAt *time.Time
	FailedAt    *time.Time
}

// TaskStatePersistence handles persisting task states
type TaskStatePersistence struct {
	repo   TaskStateRepository
	logger *zap.Logger
}

// NewTaskStatePersistence creates a new task state persistence handler
func NewTaskStatePersistence(repo TaskStateRepository, logger *zap.Logger) *TaskStatePersistence {
	return &TaskStatePersistence{
		repo:   repo,
		logger: logger,
	}
}

// OnTaskEnqueued persists task state when task is enqueued
func (p *TaskStatePersistence) OnTaskEnqueued(ctx context.Context, taskID, taskType, queueName string, payload interface{}, maxRetries int) error {
	if err := p.repo.CreateTaskState(ctx, taskID, taskType, queueName, payload, maxRetries); err != nil {
		return fmt.Errorf("failed to persist task state: %w", err)
	}
	p.logger.Info("Task state persisted",
		zap.String("task_id", taskID),
		zap.String("task_type", taskType),
		zap.String("status", "pending"),
	)
	return nil
}

// OnTaskStarted updates task state when task starts processing
func (p *TaskStatePersistence) OnTaskStarted(ctx context.Context, taskID string) error {
	if err := p.repo.UpdateTaskState(ctx, taskID, "processing", 0, ""); err != nil {
		return fmt.Errorf("failed to update task state: %w", err)
	}
	return nil
}

// OnTaskCompleted updates task state when task completes successfully
func (p *TaskStatePersistence) OnTaskCompleted(ctx context.Context, taskID string) error {
	if err := p.repo.UpdateTaskState(ctx, taskID, "completed", 0, ""); err != nil {
		return fmt.Errorf("failed to update task state: %w", err)
	}
	p.logger.Info("Task completed",
		zap.String("task_id", taskID),
	)
	return nil
}

// OnTaskFailed updates task state when task fails
func (p *TaskStatePersistence) OnTaskFailed(ctx context.Context, taskID string, retryCount int, err error) error {
	status := "failed"
	if retryCount < 3 { // Will retry
		status = "retrying"
	}
	
	errorMsg := ""
	if err != nil {
		errorMsg = err.Error()
	}
	
	if updateErr := p.repo.UpdateTaskState(ctx, taskID, status, retryCount, errorMsg); updateErr != nil {
		return fmt.Errorf("failed to update task state: %w", updateErr)
	}
	
	p.logger.Warn("Task failed",
		zap.String("task_id", taskID),
		zap.Int("retry_count", retryCount),
		zap.Error(err),
	)
	return nil
}

