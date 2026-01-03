package workers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"
	"stackyn/server/internal/tasks"
)

// AsynqServer wraps Asynq server for task processing
type AsynqServer struct {
	server   *asynq.Server
	mux      *asynq.ServeMux
	logger   *zap.Logger
	handler  *tasks.TaskHandler
	persist  *tasks.TaskStatePersistence
}

// NewAsynqServer creates a new Asynq server
// queues specifies which queues this worker should listen to (map of queue name to weight)
// If nil, defaults to all task-specific queues
func NewAsynqServer(redisAddr string, logger *zap.Logger, handler *tasks.TaskHandler, persist *tasks.TaskStatePersistence, queues map[string]int) *AsynqServer {
	redisOpt := asynq.RedisClientOpt{
		Addr: redisAddr,
	}

	// Default queues if not specified
	if queues == nil {
		queues = map[string]int{
			tasks.QueueBuild:   10, // Build tasks
			tasks.QueueDeploy:  10, // Deploy tasks
			tasks.QueueCleanup: 5,  // Cleanup tasks
		}
	}

	// Configure server with dead-letter queue
	config := asynq.Config{
		Concurrency: 10, // Process 10 tasks concurrently
		Queues:      queues,
		StrictPriority: false, // No strict priority needed with task-specific queues
		ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
			// Check if error is "handler not found" - this shouldn't happen with task-specific queues
			// but log as warning instead of error if it does
			errMsg := err.Error()
			if strings.Contains(errMsg, "handler not found") {
				logger.Warn("Task handler not found - this worker should not process this task type",
					zap.String("task_type", task.Type()),
					zap.Error(err),
				)
			} else {
				logger.Error("Task processing error",
					zap.String("task_type", task.Type()),
					zap.Error(err),
				)
			}
		}),
		RetryDelayFunc: func(n int, err error, task *asynq.Task) time.Duration {
			// Exponential backoff with jitter
			baseDelay := time.Duration(n) * time.Second
			if baseDelay > 30*time.Second {
				baseDelay = 30 * time.Second
			}
			return baseDelay
		},
		// Dead-letter queue configuration
		IsFailure: func(err error) bool {
			// Consider all errors as failures (can be customized)
			return err != nil
		},
	}

	server := asynq.NewServer(redisOpt, config)
	mux := asynq.NewServeMux()

	asynqServer := &AsynqServer{
		server:  server,
		mux:     mux,
		logger:  logger,
		handler: handler,
		persist: persist,
	}

	// Setup dead-letter queue monitoring
	SetupDeadLetterQueue(redisAddr, logger)

	return asynqServer
}

// RegisterHandlers registers all task handlers with middleware
// This is a convenience method that registers all handlers
func (s *AsynqServer) RegisterHandlers() {
	s.RegisterBuildHandler()
	s.RegisterDeployHandler()
	s.RegisterCleanupHandler()
}

// RegisterBuildHandler registers only the build task handler
func (s *AsynqServer) RegisterBuildHandler() {
	s.mux.HandleFunc(tasks.TypeBuildTask, s.withPersistence(s.handler.HandleBuildTask))
}

// RegisterDeployHandler registers only the deploy task handler
func (s *AsynqServer) RegisterDeployHandler() {
	s.mux.HandleFunc(tasks.TypeDeployTask, s.withPersistence(s.handler.HandleDeployTask))
}

// RegisterCleanupHandler registers only the cleanup task handler
func (s *AsynqServer) RegisterCleanupHandler() {
	s.mux.HandleFunc(tasks.TypeCleanupTask, s.withPersistence(s.handler.HandleCleanupTask))
}

// withPersistence wraps a task handler with state persistence
func (s *AsynqServer) withPersistence(handler func(context.Context, *asynq.Task) error) asynq.HandlerFunc {
	return func(ctx context.Context, t *asynq.Task) error {
		// Task ID is not directly available in handler context
		// We'll track it via task payload or use a different approach
		// For now, skip persistence tracking in handler (can be added via middleware)
		
		// Execute handler
		err := handler(ctx, t)

		// TODO: Track task state via database queries using task payload
		// This requires extracting task ID from payload or using a different tracking mechanism

		return err
	}
}

// Start starts the Asynq server
func (s *AsynqServer) Start(ctx context.Context) error {
	s.logger.Info("Starting Asynq server")

	if err := s.server.Start(s.mux); err != nil {
		return fmt.Errorf("failed to start Asynq server: %w", err)
	}

	// Wait for context cancellation
	<-ctx.Done()
	return ctx.Err()
}

// Stop gracefully stops the Asynq server
func (s *AsynqServer) Stop(ctx context.Context) error {
	s.logger.Info("Stopping Asynq server")
	s.server.Shutdown()
	return nil
}

// Name returns the server name
func (s *AsynqServer) Name() string {
	return "asynq-server"
}

