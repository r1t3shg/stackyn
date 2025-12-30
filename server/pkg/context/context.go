package context

import (
	"context"
	"time"

	"go.uber.org/zap"
)

// PropagateContext propagates context values and cancellation to child operations
func PropagateContext(parent context.Context) context.Context {
	// Create a new context that inherits all values and cancellation from parent
	return parent
}

// WithLogger adds a logger to the context
func WithLogger(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, "logger", logger)
}

// LoggerFromContext retrieves the logger from context
func LoggerFromContext(ctx context.Context) *zap.Logger {
	if logger, ok := ctx.Value("logger").(*zap.Logger); ok {
		return logger
	}
	return zap.NewNop()
}

// WithRequestTimeout creates a context with request timeout
func WithRequestTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, timeout)
}

// IsCancelled checks if the context is cancelled
func IsCancelled(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

// WaitForCancellation blocks until context is cancelled
func WaitForCancellation(ctx context.Context) error {
	<-ctx.Done()
	return ctx.Err()
}

