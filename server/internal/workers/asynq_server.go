package workers

import (
	"context"
	"fmt"
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
func NewAsynqServer(redisAddr string, logger *zap.Logger, handler *tasks.TaskHandler, persist *tasks.TaskStatePersistence) *AsynqServer {
	redisOpt := asynq.RedisClientOpt{
		Addr: redisAddr,
	}

	// Configure server with dead-letter queue
	config := asynq.Config{
		Concurrency: 10, // Process 10 tasks concurrently
		Queues: map[string]int{
			tasks.QueueCritical: 6, // Higher priority
			tasks.QueueDefault:  3,
			tasks.QueueLow:      1,
		},
		StrictPriority: true, // Process critical queue first
		ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
			logger.Error("Task processing error",
				zap.String("task_type", task.Type()),
				zap.Error(err),
			)
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

// RegisterHandlers registers task handlers with middleware
func (s *AsynqServer) RegisterHandlers() {
	// Register build task handler with persistence middleware
	s.mux.HandleFunc(tasks.TypeBuildTask, s.withPersistence(s.handler.HandleBuildTask))
	
	// Register deploy task handler with persistence middleware
	s.mux.HandleFunc(tasks.TypeDeployTask, s.withPersistence(s.handler.HandleDeployTask))
	
	// Register cleanup task handler with persistence middleware
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

