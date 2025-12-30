package workers

import (
	"context"

	"go.uber.org/zap"
)

// Worker is the interface that all workers must implement
type Worker interface {
	// Start begins processing work items
	Start(ctx context.Context) error
	// Stop gracefully stops the worker
	Stop(ctx context.Context) error
	// Name returns the worker's name for logging
	Name() string
}

// BaseWorker provides common functionality for workers
type BaseWorker struct {
	Logger *zap.Logger
	name   string
}

// NewBaseWorker creates a new base worker
func NewBaseWorker(name string, logger *zap.Logger) *BaseWorker {
	return &BaseWorker{
		Logger: logger,
		name:   name,
	}
}

// Name returns the worker's name
func (w *BaseWorker) Name() string {
	return w.name
}

