package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"stackyn/server/internal/infra"
	"stackyn/server/internal/services"
	"stackyn/server/internal/tasks"
	"stackyn/server/internal/workers"

	"go.uber.org/zap"
)

func main() {
	// Load configuration
	config, err := infra.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger, err := initLogger(config.LogLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("Starting cleanup worker",
		zap.String("redis_addr", config.Redis.Addr),
	)

	// Create root context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize cleanup service
	// Temp directories to prune
	tempDirs := []string{
		filepath.Join(".", "clones"),    // Git clone directories
		filepath.Join(".", "logs"),      // Log directories (old logs)
		filepath.Join(".", "tmp"),        // Temporary files
		filepath.Join(".", "builds"),     // Build artifacts
	}

	maxDiskUsagePercent := 85.0 // Start cleanup when disk usage exceeds 85%

	cleanupService, err := services.NewCleanupService(config.Docker.Host, logger, tempDirs, maxDiskUsagePercent)
	if err != nil {
		logger.Fatal("Failed to create cleanup service", zap.Error(err))
	}
	defer cleanupService.Close()

	// Initialize plan enforcement service (not needed for cleanup, but required by interface)
	planEnforcement := services.NewPlanEnforcementService(logger)

	// Initialize constraints service (not needed for cleanup, but required by interface)
	maxBuildTimeMinutes := 15
	constraintsService := services.NewConstraintsService(logger, maxBuildTimeMinutes)

	// Initialize task handler with cleanup service
	taskHandler := tasks.NewTaskHandler(
		logger,
		nil, // No Git service needed for cleanup worker
		nil, // No Docker build service needed for cleanup worker
		nil, // No runtime detector needed for cleanup worker
		nil, // No Dockerfile generator needed for cleanup worker
		nil, // No log persister needed for cleanup worker
		nil, // No deployment service needed for cleanup worker
		cleanupService,
		planEnforcement,
		constraintsService,
		nil, // No task enqueue service needed for cleanup worker
		nil, // No WebSocket broadcast client needed for cleanup worker
		nil, // No deployment repository needed for cleanup worker
		nil, // No app repository needed for cleanup worker
	)

	// Initialize task state persistence (nil for now - wire up when DB is ready)
	var taskPersistence *tasks.TaskStatePersistence
	// TODO: Initialize with database repository when DB is connected

	// Initialize Asynq server - only listen to cleanup queue
	cleanupQueues := map[string]int{
		tasks.QueueCleanup: 5, // Only process cleanup tasks
	}
	server := workers.NewAsynqServer(config.Redis.Addr, config.Redis.Password, logger, taskHandler, taskPersistence, cleanupQueues)
	// Only register cleanup task handler for cleanup worker
	server.RegisterCleanupHandler()

	// Start server in goroutine
	go func() {
		logger.Info("Starting cleanup worker server")
		if err := server.Start(ctx); err != nil && err != context.Canceled {
			logger.Fatal("Cleanup worker server failed", zap.Error(err))
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down cleanup worker...")

	// Cancel context to signal server to stop
	cancel()

	// Wait for server to finish with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Stop(shutdownCtx); err != nil {
		logger.Error("Cleanup worker shutdown error", zap.Error(err))
	} else {
		logger.Info("Cleanup worker stopped gracefully")
	}

	logger.Info("Cleanup worker exited")
}

func initLogger(level string) (*zap.Logger, error) {
	config := zap.NewProductionConfig()
	
	var zapLevel zap.AtomicLevel
	switch level {
	case "debug":
		zapLevel = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		zapLevel = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		zapLevel = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		zapLevel = zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		zapLevel = zap.NewAtomicLevelAt(zap.InfoLevel)
	}
	
	config.Level = zapLevel
	return config.Build()
}
