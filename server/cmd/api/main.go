package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"stackyn/server/internal/api"
	"stackyn/server/internal/db"
	"stackyn/server/internal/infra"

	"go.uber.org/zap"
)

func main() {
	// Load configuration (fails fast on missing required configs)
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

	logger.Info("Configuration loaded successfully",
		zap.String("server_addr", config.Server.Addr),
		zap.String("server_port", config.Server.Port),
		zap.String("postgres_host", config.Postgres.Host),
		zap.String("redis_host", config.Redis.Host),
		zap.String("docker_host", config.Docker.Host),
	)

	// Initialize database connection
	database, err := db.NewDB(config.Postgres.DSN, logger)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer database.Close()

	// Initialize HTTP server with chi router
	router := api.Router(logger, config, database)
	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%s", config.Server.Addr, config.Server.Port),
		Handler: router,
	}

	// Start server in goroutine
	go func() {
		logger.Info("Starting API server", zap.String("addr", server.Addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server failed to start", zap.Error(err))
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Create shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Gracefully shutdown server
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited")
}

func initLogger(level string) (*zap.Logger, error) {
	config := zap.NewProductionConfig()
	
	// Parse log level
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

