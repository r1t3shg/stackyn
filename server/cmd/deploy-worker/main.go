package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
)

func main() {
	// Initialize logger
	logger, err := initLogger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Create root context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize deploy worker
	worker := setupDeployWorker(ctx, logger)

	// Start worker in goroutine
	go func() {
		logger.Info("Starting deploy worker")
		if err := worker.Start(ctx); err != nil {
			logger.Fatal("Deploy worker failed", zap.Error(err))
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down deploy worker...")

	// Cancel context to signal worker to stop
	cancel()

	// Wait for worker to finish with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Wait for worker to finish gracefully
	done := make(chan error, 1)
	go func() {
		done <- worker.Stop(shutdownCtx)
	}()

	select {
	case err := <-done:
		if err != nil {
			logger.Error("Deploy worker shutdown error", zap.Error(err))
		} else {
			logger.Info("Deploy worker stopped gracefully")
		}
	case <-shutdownCtx.Done():
		logger.Warn("Deploy worker forced to shutdown due to timeout")
	}

	logger.Info("Deploy worker exited")
}

func initLogger() (*zap.Logger, error) {
	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	return config.Build()
}

func setupDeployWorker(ctx context.Context, logger *zap.Logger) *DeployWorker {
	// TODO: Initialize deploy worker from internal/workers package
	return &DeployWorker{logger: logger}
}

// DeployWorker is a placeholder - will be implemented in internal/workers
type DeployWorker struct {
	logger *zap.Logger
}

func (w *DeployWorker) Start(ctx context.Context) error {
	// TODO: Implement worker logic
	<-ctx.Done()
	return ctx.Err()
}

func (w *DeployWorker) Stop(ctx context.Context) error {
	// TODO: Implement graceful stop
	return nil
}

