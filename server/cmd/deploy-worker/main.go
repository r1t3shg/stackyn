package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
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

	logger.Info("Starting deploy worker",
		zap.String("redis_addr", config.Redis.Addr),
	)

	// Create root context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize log persistence service for runtime logs
	logStorageDir := "./logs" // Relative to worker binary
	if err := os.MkdirAll(logStorageDir, 0755); err != nil {
		logger.Fatal("Failed to create log storage directory", zap.Error(err))
	}
	
	usePostgres := false // TODO: Make configurable
	maxStoragePerAppMB := int64(100) // Default: 100 MB per app
	logPersistence := services.NewLogPersistenceService(logger, logStorageDir, usePostgres, maxStoragePerAppMB)

	// Initialize Docker deployment service with log persistence
	deploymentService, err := services.NewDeploymentService(config.Docker.Host, logger, logPersistence)
	if err != nil {
		logger.Fatal("Failed to create deployment service", zap.Error(err))
	}
	defer deploymentService.Close()

	// Initialize plan enforcement service
	planEnforcement := services.NewPlanEnforcementService(logger)

	// Initialize task handler with deployment service
	taskHandler := tasks.NewTaskHandler(
		logger,
		nil, // No Git service needed for deploy worker
		nil, // No Docker build service needed for deploy worker
		nil, // No runtime detector needed for deploy worker
		nil, // No Dockerfile generator needed for deploy worker
		logPersistence, // Log persistence for runtime logs
		deploymentService,
		nil, // No cleanup service needed for deploy worker
		planEnforcement,
	)

	// Initialize task state persistence (nil for now - wire up when DB is ready)
	var taskPersistence *tasks.TaskStatePersistence
	// TODO: Initialize with database repository when DB is connected

	// Initialize Asynq server
	server := workers.NewAsynqServer(config.Redis.Addr, logger, taskHandler, taskPersistence)
	server.RegisterHandlers()

	// Start server in goroutine
	go func() {
		logger.Info("Starting deploy worker server")
		if err := server.Start(ctx); err != nil && err != context.Canceled {
			logger.Fatal("Deploy worker server failed", zap.Error(err))
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down deploy worker...")

	// Cancel context to signal server to stop
	cancel()

	// Wait for server to finish with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Stop(shutdownCtx); err != nil {
		logger.Error("Deploy worker shutdown error", zap.Error(err))
	} else {
		logger.Info("Deploy worker stopped gracefully")
	}

	logger.Info("Deploy worker exited")
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
