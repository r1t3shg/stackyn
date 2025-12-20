// Package main provides the HTTP API server for the PaaS backend.
// This is the entry point for the API server that handles REST requests.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"mvp-be/internal/apps"
	"mvp-be/internal/config"
	"mvp-be/internal/db"
	"mvp-be/internal/deployments"
	"mvp-be/internal/gitrepo"
)

// main is the entry point for the API server.
// It initializes the database, sets up routes, and starts the HTTP server.
//
// Server setup process:
//  1. Load configuration from environment variables
//  2. Connect to PostgreSQL database
//  3. Run database migrations
//  4. Initialize data stores (apps, deployments)
//  5. Configure HTTP router with middleware
//  6. Register API endpoints
//  7. Start listening for HTTP requests
func main() {
	// Load configuration from environment variables
	cfg := config.Load()

	// Initialize database connection
	// This connects to PostgreSQL using the connection string from config
	database, err := db.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	// Ensure database connection is closed when server shuts down
	defer database.Close()

	// Run database migrations
	// This applies any pending SQL migrations to set up/update the schema
	if err := database.Migrate(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize data stores
	// These provide database operations for apps and deployments
	appStore := apps.NewStore(database.DB)
	deploymentStore := deployments.NewStore(database.DB)

	// Initialize git cloner for Dockerfile validation
	workDir := "/tmp/mvp-api-valiadation"
	if err := os.MkdirAll(workDir, 0755); err != nil {
		log.Fatalf("Failed to create validation work directory: %v", err)
	}
	cloner := gitrepo.NewCloner(workDir)

	// Setup HTTP router using chi (lightweight, fast router)
	r := chi.NewRouter()

	// Add middleware (executed in order for all requests)
	r.Use(middleware.Logger)    // Log all HTTP requests
	r.Use(middleware.Recoverer) // Recover from panics and return 500 errors
	r.Use(middleware.RequestID) // Add unique request ID to each request
	r.Use(middleware.RealIP)    // Get real client IP (useful behind proxies)

	// Register API routes under /api/v1 prefix
	r.Route("/api/v1", func(r chi.Router) {
		// Apps endpoints - manage applications
		r.Route("/apps", func(r chi.Router) {
			r.Get("/", listApps(appStore))                               // GET /api/v1/apps - List all apps
			r.Post("/", createApp(appStore, deploymentStore, cloner))    // POST /api/v1/apps - Create new app
			r.Get("/{id}", getApp(appStore))                             // GET /api/v1/apps/{id} - Get app by ID
			r.Delete("/{id}", deleteApp(appStore))                       // DELETE /api/v1/apps/{id} - Delete app
			r.Get("/{id}/deployments", listDeployments(deploymentStore)) // GET /api/v1/apps/{id}/deployments - List app deployments
		})

		// Deployments endpoints - manage deployments
		r.Route("/deployments", func(r chi.Router) {
			r.Get("/{id}", getDeployment(deploymentStore)) // GET /api/v1/deployments/{id} - Get deployment by ID
		})
	})

	// Health check endpoint (not under /api/v1)
	// Used by load balancers and monitoring systems to check if server is alive
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// Start HTTP server
	port := cfg.Port
	log.Printf("API server starting on port %s", port)
	// ListenAndServe blocks until the server stops (or fails)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// listApps returns an HTTP handler that lists all applications.
// GET /api/v1/apps
//
// Returns:
//   - 200 OK: JSON array of all apps
//   - 500 Internal Server Error: Database error
func listApps(store *apps.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Fetch all apps from database
		apps, err := store.List()
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		// Return apps as JSON
		respondJSON(w, http.StatusOK, apps)
	}
}

// createApp returns an HTTP handler that creates a new application.
// POST /api/v1/apps
//
// Request body:
//
//	{
//	  "name": "my-app",
//	  "repo_url": "https://github.com/user/repo.git"
//	}
//
// Returns:
//   - 201 Created: JSON object with created app and initial deployment
//   - 400 Bad Request: Invalid request body or missing required fields
//   - 500 Internal Server Error: Database error
func createApp(appStore *apps.Store, deploymentStore *deployments.Store, cloner *gitrepo.Cloner) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parse request body
		var req struct {
			Name    string `json:"name"`     // Application name (must be unique)
			RepoURL string `json:"repo_url"` // Git repository URL
		}

		// Decode JSON request body
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		// Validate required fields
		if req.Name == "" || req.RepoURL == "" {
			respondError(w, http.StatusBadRequest, "name and repo_url are required")
			return
		}

		// Validate repository has Dockerfile before creating app
		// Use a temporary deployment ID for validation
		tempDeploymentID := int(time.Now().Unix())
		repoPath, err := cloner.Clone(req.RepoURL, tempDeploymentID)
		if err != nil {
			respondError(w, http.StatusBadRequest, fmt.Sprintf("Failed to clone repository: %v", err))
			return
		}

		// Check if Dockerfile exists
		if err := gitrepo.CheckDockerfile(repoPath); err != nil {
			// Clean up cloned repository
			os.RemoveAll(repoPath)
			respondError(w, http.StatusBadRequest, "Dockerfile is not available in the repository root directory. Please ensure your repository contains a Dockerfile.")
			return
		}

		// Clean up validation repository
		os.RemoveAll(repoPath)

		// Create app if Dockerfile validation passes
		app, err := appStore.Create(req.Name, req.RepoURL)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		// Automatically create an initial deployment with status "pending"
		// The worker will pick this up and process it
		deployment, err := deploymentStore.Create(app.ID)
		if err != nil {
			// Log warning but don't fail the request
			// The app was created successfully, deployment can be created later
			log.Printf("Warning: failed to create deployment: %v", err)
		}

		// Return both app and deployment
		respondJSON(w, http.StatusCreated, map[string]interface{}{
			"app":        app,
			"deployment": deployment,
		})
	}
}

// getApp returns an HTTP handler that retrieves an app by ID.
// GET /api/v1/apps/{id}
//
// Returns:
//   - 200 OK: JSON object with app data
//   - 400 Bad Request: Invalid app ID format
//   - 404 Not Found: App not found
func getApp(store *apps.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract ID from URL parameter
		id, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid app ID")
			return
		}

		// Fetch app from database
		app, err := store.GetByID(id)
		if err != nil {
			respondError(w, http.StatusNotFound, "App not found")
			return
		}

		// Return app as JSON
		respondJSON(w, http.StatusOK, app)
	}
}

// deleteApp returns an HTTP handler that deletes an app by ID.
// DELETE /api/v1/apps/{id}
//
// Note: This will cascade delete all associated deployments (database foreign key constraint).
//
// Returns:
//   - 204 No Content: App deleted successfully
//   - 400 Bad Request: Invalid app ID format
//   - 500 Internal Server Error: Database error
func deleteApp(store *apps.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract ID from URL parameter
		id, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid app ID")
			return
		}

		// Delete app from database
		if err := store.Delete(id); err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		// Return 204 No Content (successful deletion)
		w.WriteHeader(http.StatusNoContent)
	}
}

// listDeployments returns an HTTP handler that lists all deployments for an app.
// GET /api/v1/apps/{id}/deployments
//
// Returns:
//   - 200 OK: JSON array of deployments for the app
//   - 400 Bad Request: Invalid app ID format
//   - 500 Internal Server Error: Database error
func listDeployments(store *deployments.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract app ID from URL parameter
		appID, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid app ID")
			return
		}

		// Fetch all deployments for this app
		deployments, err := store.ListByAppID(appID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		// Return deployments as JSON
		respondJSON(w, http.StatusOK, deployments)
	}
}

// getDeployment returns an HTTP handler that retrieves a deployment by ID.
// GET /api/v1/deployments/{id}
//
// Returns:
//   - 200 OK: JSON object with deployment data (including build logs, status, etc.)
//   - 400 Bad Request: Invalid deployment ID format
//   - 404 Not Found: Deployment not found
func getDeployment(store *deployments.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract ID from URL parameter
		id, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid deployment ID")
			return
		}

		// Fetch deployment from database
		deployment, err := store.GetByID(id)
		if err != nil {
			respondError(w, http.StatusNotFound, "Deployment not found")
			return
		}

		// Return deployment as JSON
		respondJSON(w, http.StatusOK, deployment)
	}
}

// respondJSON is a helper function that sends a JSON response with the given status code.
//
// Parameters:
//   - w: HTTP response writer
//   - status: HTTP status code (e.g., 200, 404, 500)
//   - payload: The data to encode as JSON (can be any type)
func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	// Set Content-Type header to indicate JSON response
	w.Header().Set("Content-Type", "application/json")
	// Write HTTP status code
	w.WriteHeader(status)
	// Encode payload as JSON and write to response
	json.NewEncoder(w).Encode(payload)
}

// respondError is a helper function that sends an error response as JSON.
//
// Parameters:
//   - w: HTTP response writer
//   - status: HTTP status code (e.g., 400, 404, 500)
//   - message: Error message to include in the response
func respondError(w http.ResponseWriter, status int, message string) {
	// Send error as JSON with format: {"error": "message"}
	respondJSON(w, status, map[string]string{"error": message})
}
