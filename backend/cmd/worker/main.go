// Package main provides the deployment worker for the PaaS backend.
// This is the entry point for the background worker that processes deployments.
// It runs separately from the API server and continuously processes pending deployments.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"mvp-be/internal/apps"
	"mvp-be/internal/config"
	"mvp-be/internal/dockerbuild"
	"mvp-be/internal/db"
	"mvp-be/internal/deployments"
	"mvp-be/internal/dockerrun"
	"mvp-be/internal/engine"
	"mvp-be/internal/envvars"
	"mvp-be/internal/gitrepo"
)

// main is the entry point for the deployment worker.
// It initializes all dependencies and starts the deployment processing loop.
//
// Worker setup process:
//   1. Load configuration from environment variables
//   2. Connect to PostgreSQL database
//   3. Run database migrations
//   4. Initialize data stores (apps, deployments)
//   5. Initialize Git cloner (with work directory)
//   6. Initialize Docker builder (connects to Docker daemon)
//   7. Initialize Docker runner (connects to Docker daemon)
//   8. Create deployment engine with all dependencies
//   9. Setup graceful shutdown signal handling
//   10. Start the deployment processing loop
func main() {
	log.Println("=== Starting Stackyn Deployment Worker ===")
	// Load configuration from environment variables
	cfg := config.Load()
	log.Printf("Configuration loaded - Database: %s, Docker: %s, Base Domain: %s",
		cfg.DatabaseURL, cfg.DockerHost, cfg.BaseDomain)

	// Initialize database connection
	// This connects to PostgreSQL using the connection string from config
	log.Println("Connecting to database...")
	database, err := db.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	// Ensure database connection is closed when worker shuts down
	defer database.Close()
	log.Println("Database connection established")

	// Run database migrations
	// This applies any pending SQL migrations to set up/update the schema
	log.Println("Running database migrations...")
	if err := database.Migrate(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("Database migrations completed")

	// Initialize data stores
	// These provide database operations for apps and deployments
	log.Println("Initializing data stores...")
	appStore := apps.NewStore(database.DB)
	deploymentStore := deployments.NewStore(database.DB)
	envVarStore := envvars.NewStore(database.DB)
	log.Println("Data stores initialized")

	// Initialize Git cloner
	// This will clone repositories to a temporary directory
	workDir := "/tmp/mvp-deployments"
	log.Printf("Initializing Git cloner with work directory: %s", workDir)
	// Create the work directory if it doesn't exist
	// Permissions: 0755 (owner: read/write/execute, group/others: read/execute)
	if err := os.MkdirAll(workDir, 0755); err != nil {
		log.Fatalf("Failed to create work directory: %v", err)
	}
	cloner := gitrepo.NewCloner(workDir)
	log.Println("Git cloner initialized")

	// Initialize Dockerfile-based builder
	// This builder uses automatic Dockerfile generation for Node.js, Python, Java, and Go apps
	// Dockerfiles are generated automatically based on detected app type
	log.Println("Initializing Docker builder...")
	builder, err := dockerbuild.NewBuilder(cfg.DockerHost)
	if err != nil {
		log.Fatalf("Failed to create Docker builder: %v", err)
	}
	log.Println("Docker builder initialized")

	// Initialize Docker runner
	// This connects to the Docker daemon to run containers
	log.Println("Initializing Docker runner...")
	runner, err := dockerrun.NewRunner(cfg.DockerHost)
	if err != nil {
		log.Fatalf("Failed to create Docker runner: %v", err)
	}
	log.Println("Docker runner initialized")

	// Initialize deployment engine
	// This orchestrates the entire deployment pipeline
	deploymentEngine := engine.NewEngine(
		deploymentStore, // Store for deployment database operations
		appStore,        // Store for app database operations
		envVarStore,     // Store for environment variables
		cloner,          // Git repository cloner
		builder,         // Docker image builder
		runner,          // Docker container runner
		cfg.BaseDomain,  // Base domain for subdomain routing
		database.DB,     // Database connection for advisory locks
	)

	// Setup graceful shutdown
	// Create a cancellable context that can be used to stop the deployment loop
	ctx, cancel := context.WithCancel(context.Background())
	// Ensure cancel is called when function exits
	defer cancel()

	// Setup signal handling for graceful shutdown
	// This allows the worker to cleanly shut down when receiving SIGTERM or SIGINT
	sigChan := make(chan os.Signal, 1)
	// Register to receive interrupt (Ctrl+C) and termination signals
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start a goroutine to handle shutdown signals
	go func() {
		// Wait for a signal
		sig := <-sigChan
		log.Printf("Received signal: %v, shutting down...", sig)
		// Cancel the context, which will stop the deployment loop
		cancel()
	}()

	// Start the deployment processing loop
	// This will run until the context is cancelled (e.g., on SIGTERM)
	// The loop continuously polls for pending deployments and processes them
	deploymentEngine.RunLoop(ctx)
}

