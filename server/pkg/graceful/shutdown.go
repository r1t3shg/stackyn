package graceful

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
)

// ShutdownHandler manages graceful shutdown of services
type ShutdownHandler struct {
	logger   *zap.Logger
	services []Shutdownable
	timeout  time.Duration
}

// Shutdownable is an interface for services that can be gracefully shut down
type Shutdownable interface {
	Shutdown(ctx context.Context) error
}

// NewShutdownHandler creates a new shutdown handler
func NewShutdownHandler(logger *zap.Logger, timeout time.Duration) *ShutdownHandler {
	return &ShutdownHandler{
		logger:  logger,
		timeout: timeout,
	}
}

// Register registers a service for graceful shutdown
func (h *ShutdownHandler) Register(service Shutdownable) {
	h.services = append(h.services, service)
}

// WaitForShutdown waits for shutdown signals and gracefully shuts down all services
func (h *ShutdownHandler) WaitForShutdown() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	h.logger.Info("Shutdown signal received, starting graceful shutdown...")

	ctx, cancel := context.WithTimeout(context.Background(), h.timeout)
	defer cancel()

	// Shutdown all services
	for _, service := range h.services {
		if err := service.Shutdown(ctx); err != nil {
			h.logger.Error("Service shutdown error", zap.Error(err))
		}
	}

	h.logger.Info("Graceful shutdown completed")
}

