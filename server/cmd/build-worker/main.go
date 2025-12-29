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

	logger.Info("Starting build worker",
		zap.String("redis_addr", config.Redis.Addr),
	)

	// Create root context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize Git service
	// Clone directory: ./clones (relative to worker binary)
	cloneDir := filepath.Join(".", "clones")
	if err := os.MkdirAll(cloneDir, 0755); err != nil {
		logger.Fatal("Failed to create clone directory", zap.Error(err))
	}
	gitService := services.NewGitService(logger, cloneDir)

	// Initialize Docker build service
	dockerBuild, err := services.NewDockerBuildService(config.Docker.Host, logger)
	if err != nil {
		logger.Fatal("Failed to create Docker build service", zap.Error(err))
	}
	defer dockerBuild.Close()

	// Initialize runtime detector
	runtimeDetector := services.NewRuntimeDetector(logger)

	// Initialize Dockerfile generator
	dockerfileGen := services.NewDockerfileGenerator(logger)

	// Initialize log persistence service
	// Storage directory: ./logs (relative to worker binary)
	logStorageDir := filepath.Join(".", "logs")
	if err := os.MkdirAll(logStorageDir, 0755); err != nil {
		logger.Fatal("Failed to create log storage directory", zap.Error(err))
	}
	
	// Use filesystem storage (can be switched to Postgres via config)
	usePostgres := false // TODO: Make configurable
	maxStoragePerAppMB := int64(100) // Default: 100 MB per app
	
	logPersistence := services.NewLogPersistenceService(logger, logStorageDir, usePostgres, maxStoragePerAppMB)

	// Initialize task handler with all services
	taskHandler := tasks.NewTaskHandler(
		logger,
		gitService,
		dockerBuild,
		runtimeDetector,
		dockerfileGen,
		logPersistence,
		nil, // No deployment service needed for build worker
		nil, // No cleanup service needed for build worker
	)

	// Initialize task state persistence (nil for now - wire up when DB is ready)
	var taskPersistence *tasks.TaskStatePersistence
	// TODO: Initialize with database repository when DB is connected
	// taskPersistence = tasks.NewTaskStatePersistence(dbRepo, logger)

	// Initialize Asynq server
	server := workers.NewAsynqServer(config.Redis.Addr, logger, taskHandler, taskPersistence)
	server.RegisterHandlers()

	// Start server in goroutine
	go func() {
		logger.Info("Starting build worker server")
		if err := server.Start(ctx); err != nil && err != context.Canceled {
			logger.Fatal("Build worker server failed", zap.Error(err))
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down build worker...")

	// Cancel context to signal server to stop
	cancel()

	// Wait for server to finish with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Stop(shutdownCtx); err != nil {
		logger.Error("Build worker shutdown error", zap.Error(err))
	} else {
		logger.Info("Build worker stopped gracefully")
	}

	logger.Info("Build worker exited")
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
