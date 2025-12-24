// Package main provides the HTTP API server for Stackyn PaaS.
// Stackyn is a Platform-as-a-Service that allows users to deploy applications
// from Git repositories. This API server handles:
//   - App management (create, list, get, delete)
//   - Deployment operations (create, list, get logs)
//   - Repository validation (Dockerfile checking)
//
// The API follows RESTful conventions and uses Chi router for HTTP routing.
// It connects to PostgreSQL for data persistence and Docker for container management.
//
// API Endpoints:
//   - GET  /health - Health check endpoint
//   - GET  /api/v1/apps - List all apps
//   - POST /api/v1/apps - Create new app
//   - GET  /api/v1/apps/{id} - Get app by ID
//   - DELETE /api/v1/apps/{id} - Delete app
//   - POST /api/v1/apps/{id}/redeploy - Trigger redeployment
//   - GET  /api/v1/apps/{id}/deployments - List deployments for an app
//   - GET  /api/v1/deployments/{id} - Get deployment details
//   - GET  /api/v1/deployments/{id}/logs - Get deployment logs
//   - GET  /api/apps - List apps by authenticated user
package main

import (
	"context"
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
	"mvp-be/internal/dockerrun"
	"mvp-be/internal/gitrepo"
)

// contextKey is a type for context keys to avoid collisions
type contextKey string

const userIDKey contextKey = "user_id"

// main is the entry point for the API server.
// It performs the following initialization steps:
//   1. Load configuration from environment variables
//   2. Connect to PostgreSQL database
//   3. Run database migrations
//   4. Initialize data stores (apps, deployments)
//   5. Initialize Git cloner for repository validation
//   6. Setup HTTP router with CORS and middleware
//   7. Register API routes
//   8. Start HTTP server on configured port
//
// Environment Variables:
//   - DATABASE_URL: PostgreSQL connection string (default: postgres://postgres:ritesh@localhost:5432/mvp?sslmode=disable)
//   - DOCKER_HOST: Docker daemon address (default: tcp://localhost:2375)
//   - BASE_DOMAIN: Base domain for subdomain routing (default: localhost)
//   - PORT: HTTP server port (default: 8080)
func main() {
	log.Println("=== Starting Stackyn API Server ===")
	cfg := config.Load()
	log.Printf("Configuration loaded - Database: %s, Docker: %s, Base Domain: %s, Port: %s",
		cfg.DatabaseURL, cfg.DockerHost, cfg.BaseDomain, cfg.Port)

	// Initialize database
	log.Println("Connecting to database...")
	database, err := db.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()
	log.Println("Database connection established")

	// Run migrations
	log.Println("Running database migrations...")
	if err := database.Migrate(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("Database migrations completed")

	// Initialize stores
	log.Println("Initializing data stores...")
	appStore := apps.NewStore(database.DB)
	deploymentStore := deployments.NewStore(database.DB)
	log.Println("Data stores initialized")

	// Initialize git cloner for Dockerfile validation
	workDir := "/tmp/mvp-api-validation"
	log.Printf("Initializing Git cloner with work directory: %s", workDir)
	if err := os.MkdirAll(workDir, 0755); err != nil {
		log.Fatalf("Failed to create validation work directory: %v", err)
	}
	cloner := gitrepo.NewCloner(workDir)
	log.Println("Git cloner initialized")

	// Initialize Docker runner for container/image management
	log.Printf("Initializing Docker runner - Host: %s", cfg.DockerHost)
	runner, err := dockerrun.NewRunner(cfg.DockerHost)
	if err != nil {
		log.Fatalf("Failed to initialize Docker runner: %v", err)
	}
	log.Println("Docker runner initialized")

	// Setup router
	r := chi.NewRouter()
	
	// CORS middleware - must be first
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Set CORS headers
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
			// Include all common headers that browsers might send
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, Accept, Origin")
			w.Header().Set("Access-Control-Allow-Credentials", "false")
			w.Header().Set("Access-Control-Max-Age", "3600")
			
			// Handle preflight OPTIONS request
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			
			next.ServeHTTP(w, r)
		})
	})
	
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)

	// Required APIs
	// GET : Fetch all apps
	// POST : create app
	// GET : fetch app by id
	// POST : redeploy
	// GET : deployment status
	// GET : logs
	// POST : add env var
	// DELETE : env var

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		// Apps endpoints
		r.Route("/apps", func(r chi.Router) {
			r.Get("/", listApps(appStore))
			r.Post("/", createApp(appStore, deploymentStore, cloner))
			r.Get("/{id}", getApp(appStore, deploymentStore))
			r.Delete("/{id}", deleteApp(appStore, deploymentStore, runner))
			r.Post("/{id}/redeploy", redeployApp(appStore, deploymentStore, cloner))
			r.Get("/{id}/deployments", listDeployments(deploymentStore))
		})

		// Deployments endpoints
		r.Route("/deployments", func(r chi.Router) {
			r.Get("/{id}", getDeployment(deploymentStore))
			r.Get("/{id}/logs", getDeploymentLogs(deploymentStore))
		})
	})

	// New API route for listing apps by user (GET /api/apps)
	r.Get("/api/apps", listAppsByUser(appStore))

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[API] GET /health - Health check")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	port := cfg.Port
	log.Printf("=== API server starting on port %s ===", port)
	log.Println("API endpoints available:")
	log.Println("  GET  /health - Health check")
	log.Println("  GET  /api/v1/apps - List all apps")
	log.Println("  POST /api/v1/apps - Create new app")
	log.Println("  GET  /api/v1/apps/{id} - Get app by ID")
	log.Println("  DELETE /api/v1/apps/{id} - Delete app")
	log.Println("  POST /api/v1/apps/{id}/redeploy - Redeploy app")
	log.Println("  GET  /api/v1/apps/{id}/deployments - List deployments")
	log.Println("  GET  /api/v1/deployments/{id} - Get deployment")
	log.Println("  GET  /api/v1/deployments/{id}/logs - Get deployment logs")
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func listApps(store *apps.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[API] GET /api/v1/apps - Listing all apps")
		apps, err := store.List()
		if err != nil {
			log.Printf("[API] ERROR - Failed to list apps: %v", err)
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		log.Printf("[API] Successfully listed %d apps", len(apps))
		respondJSON(w, http.StatusOK, apps)
	}
}

func createApp(appStore *apps.Store, deploymentStore *deployments.Store, cloner *gitrepo.Cloner) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[API] POST /api/v1/apps - Creating new app")
		var req struct {
			Name    string `json:"name"`
			RepoURL string `json:"repo_url"`
			Branch  string `json:"branch"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Printf("[API] ERROR - Invalid request body: %v", err)
			respondJSON(w, http.StatusBadRequest, map[string]interface{}{
				"error": "Invalid request body",
				"app":   nil,
			})
			return
		}

		log.Printf("[API] Request - Name: %s, Repo: %s, Branch: %s", req.Name, req.RepoURL, req.Branch)

		if req.Name == "" || req.RepoURL == "" || req.Branch == "" {
			log.Printf("[API] ERROR - Missing required fields")
			respondJSON(w, http.StatusBadRequest, map[string]interface{}{
				"error": "name, repo_url, and branch are required",
				"app":   nil,
			})
			return
		}

		// Create app first
		log.Printf("[API] Creating app in database...")
		app, err := appStore.Create(req.Name, req.RepoURL, req.Branch)
		if err != nil {
			log.Printf("[API] ERROR - Failed to create app: %v", err)
			respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
				"error": err.Error(),
				"app":   nil,
			})
			return
		}
		log.Printf("[API] App created successfully - ID: %s, Name: %s", app.ID, app.Name)

		// Create initial deployment
		// Convert app.ID (string) to int for deployment creation
		appID, err := strconv.Atoi(app.ID)
		if err != nil {
			log.Printf("[API] ERROR - Invalid app ID format: %v", err)
			respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
				"error": fmt.Sprintf("Invalid app ID format: %v", err),
				"app":   app,
			})
			return
		}
		log.Printf("[API] Creating deployment for app ID: %d", appID)
		deployment, err := deploymentStore.Create(appID)
		if err != nil {
			log.Printf("[API] ERROR - Failed to create deployment: %v", err)
			respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
				"error": fmt.Sprintf("Failed to create deployment: %v", err),
				"app":   app,
			})
			return
		}
		log.Printf("[API] Deployment created - ID: %d, Status: %s", deployment.ID, deployment.Status)
		
		// Update app status to "Pending" when deployment is created
		if err := appStore.UpdateStatus(appID, "Pending"); err != nil {
			log.Printf("[API] WARNING - Failed to update app status to Pending: %v", err)
		}

		// Validate repository has Dockerfile after creating app and deployment
		// Use a temporary deployment ID for validation
		log.Printf("[API] Validating repository - Cloning %s (branch: %s)", req.RepoURL, req.Branch)
		tempDeploymentID := int(time.Now().Unix())
		repoPath, err := cloner.Clone(req.RepoURL, tempDeploymentID, req.Branch)
		if err != nil {
			log.Printf("[API] ERROR - Git clone failed: %v", err)
			// Update deployment with error
			errorMsg := fmt.Sprintf("Failed to clone repository: %v", err)
			deploymentStore.UpdateError(deployment.ID, errorMsg)
			// Update app status to "Failed"
			appStore.UpdateStatus(appID, "Failed")
			// Refresh deployment to get updated status
			deployment, _ = deploymentStore.GetByID(deployment.ID)
			respondJSON(w, http.StatusBadRequest, map[string]interface{}{
				"error":      errorMsg,
				"app":        app,
				"deployment": deployment,
			})
			return
		}
		log.Printf("[API] Repository cloned successfully to: %s", repoPath)

		// Check if Dockerfile exists
		log.Printf("[API] Checking for Dockerfile in repository...")
		if err := gitrepo.CheckDockerfile(repoPath); err != nil {
			log.Printf("[API] ERROR - Dockerfile not found: %v", err)
			// Clean up cloned repository
			os.RemoveAll(repoPath)
			// Update deployment with error
			errorMsg := "Dockerfile is not available in the repository root directory. Please ensure your repository contains a Dockerfile."
			deploymentStore.UpdateError(deployment.ID, errorMsg)
			// Update app status to "Failed"
			appStore.UpdateStatus(appID, "Failed")
			// Refresh deployment to get updated status
			deployment, _ = deploymentStore.GetByID(deployment.ID)
			respondJSON(w, http.StatusBadRequest, map[string]interface{}{
				"error":      errorMsg,
				"app":        app,
				"deployment": deployment,
			})
			return
		}

		// Clean up validation repository
		log.Printf("[API] Cleaning up validation repository...")
		os.RemoveAll(repoPath)

		// If validation passes, deployment remains in "pending" status for worker to process
		log.Printf("[API] App creation completed successfully - App ID: %s, Deployment ID: %d", app.ID, deployment.ID)
		respondJSON(w, http.StatusCreated, map[string]interface{}{
			"app":        app,
			"deployment": deployment,
		})
	}
}

func getApp(appStore *apps.Store, deploymentStore *deployments.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			log.Printf("[API] ERROR - Invalid app ID: %s", chi.URLParam(r, "id"))
			respondError(w, http.StatusBadRequest, "Invalid app ID")
			return
		}

		log.Printf("[API] GET /api/v1/apps/%d - Fetching app", id)
		app, err := appStore.GetByID(id)
		if err != nil {
			log.Printf("[API] ERROR - App not found: %d", id)
			respondError(w, http.StatusNotFound, "App not found")
			return
		}
		log.Printf("[API] App found - ID: %d, Name: %s, Status: %s", id, app.Name, app.Status)

		// Get the latest deployment for this app
		appDeployments, err := deploymentStore.ListByAppID(id)
		var activeDeployment *deployments.Deployment
		if err == nil && len(appDeployments) > 0 {
			activeDeployment = appDeployments[0] // First one is the latest (ordered by created_at DESC)
		}

		// Build response with runtime and deployment info
		response := map[string]interface{}{
			"id":        app.ID,
			"name":      app.Name,
			"slug":      app.Slug,
			"status":    app.Status,
			"url":       app.URL,
			"repo_url":  app.RepoURL,
			"branch":    app.Branch,
			"created_at": app.CreatedAt,
			"updated_at": app.UpdatedAt,
		}

		// Add deployment info
		if activeDeployment != nil {
			// Map deployment status to state
			state := string(activeDeployment.Status)
			// Format deployment ID as "dep_{id}"
			activeDeploymentID := fmt.Sprintf("dep_%d", activeDeployment.ID)
			
			response["deployment"] = map[string]interface{}{
				"active_deployment_id": activeDeploymentID,
				"last_deployed_at":     activeDeployment.UpdatedAt,
				"state":                state,
			}
		} else {
			// No deployment found
			response["deployment"] = map[string]interface{}{
				"active_deployment_id": nil,
				"last_deployed_at":    nil,
				"state":               "none",
			}
		}

		respondJSON(w, http.StatusOK, response)
	}
}

func redeployApp(appStore *apps.Store, deploymentStore *deployments.Store, cloner *gitrepo.Cloner) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			log.Printf("[API] ERROR - Invalid app ID: %s", chi.URLParam(r, "id"))
			respondError(w, http.StatusBadRequest, "Invalid app ID")
			return
		}

		log.Printf("[API] POST /api/v1/apps/%d/redeploy - Initiating redeployment", id)

		// Get the app
		app, err := appStore.GetByID(id)
		if err != nil {
			log.Printf("[API] ERROR - App not found: %d", id)
			respondError(w, http.StatusNotFound, "App not found")
			return
		}
		log.Printf("[API] App found - ID: %d, Name: %s", id, app.Name)

		// Create new deployment
		appID, err := strconv.Atoi(app.ID)
		if err != nil {
			log.Printf("[API] ERROR - Invalid app ID format: %v", err)
			respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
				"error": fmt.Sprintf("Invalid app ID format: %v", err),
				"app":   app,
			})
			return
		}

		log.Printf("[API] Creating new deployment for app ID: %d", appID)
		deployment, err := deploymentStore.Create(appID)
		if err != nil {
			log.Printf("[API] ERROR - Failed to create deployment: %v", err)
			respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
				"error": fmt.Sprintf("Failed to create deployment: %v", err),
				"app":   app,
			})
			return
		}
		log.Printf("[API] Deployment created - ID: %d", deployment.ID)
		
		// Update app status to "Pending" when redeployment is initiated
		if err := appStore.UpdateStatus(appID, "Pending"); err != nil {
			log.Printf("[API] WARNING - Failed to update app status to Pending: %v", err)
		}

		// Validate repository has Dockerfile
		// Use a temporary deployment ID for validation
		tempDeploymentID := int(time.Now().Unix())
		
		// Use branch from app, default to "main" if empty
		branch := app.Branch
		if branch == "" {
			branch = "main"
		}

		log.Printf("[API] Validating repository - Cloning %s (branch: %s)", app.RepoURL, branch)
		repoPath, err := cloner.Clone(app.RepoURL, tempDeploymentID, branch)
		if err != nil {
			log.Printf("[API] ERROR - Git clone failed: %v", err)
			// Update deployment with error
			errorMsg := fmt.Sprintf("Failed to clone repository: %v", err)
			deploymentStore.UpdateError(deployment.ID, errorMsg)
			// Update app status to "Failed"
			appStore.UpdateStatus(appID, "Failed")
			// Refresh deployment to get updated status
			deployment, _ = deploymentStore.GetByID(deployment.ID)
			respondJSON(w, http.StatusBadRequest, map[string]interface{}{
				"error":      errorMsg,
				"app":        app,
				"deployment": deployment,
			})
			return
		}
		log.Printf("[API] Repository cloned successfully")

		// Check if Dockerfile exists
		log.Printf("[API] Checking for Dockerfile...")
		if err := gitrepo.CheckDockerfile(repoPath); err != nil {
			log.Printf("[API] ERROR - Dockerfile not found: %v", err)
			// Clean up cloned repository
			os.RemoveAll(repoPath)
			// Update deployment with error
			errorMsg := "Dockerfile is not available in the repository root directory. Please ensure your repository contains a Dockerfile."
			deploymentStore.UpdateError(deployment.ID, errorMsg)
			// Update app status to "Failed"
			appStore.UpdateStatus(appID, "Failed")
			// Refresh deployment to get updated status
			deployment, _ = deploymentStore.GetByID(deployment.ID)
			respondJSON(w, http.StatusBadRequest, map[string]interface{}{
				"error":      errorMsg,
				"app":        app,
				"deployment": deployment,
			})
			return
		}

		// Clean up validation repository
		log.Printf("[API] Cleaning up validation repository...")
		os.RemoveAll(repoPath)

		// Deployment created successfully, will be processed by worker
		log.Printf("[API] Redeployment initiated successfully - Deployment ID: %d", deployment.ID)
		respondJSON(w, http.StatusCreated, map[string]interface{}{
			"message":    "Redeployment initiated",
			"app":        app,
			"deployment": deployment,
		})
	}
}

func deleteApp(appStore *apps.Store, deploymentStore *deployments.Store, runner *dockerrun.Runner) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			log.Printf("[API] ERROR - Invalid app ID: %s", chi.URLParam(r, "id"))
			respondError(w, http.StatusBadRequest, "Invalid app ID")
			return
		}

		log.Printf("[API] DELETE /api/v1/apps/%d - Deleting app and cleaning up resources", id)

		// Get all deployments for this app
		appDeployments, err := deploymentStore.ListByAppID(id)
		if err != nil {
			log.Printf("[API] WARNING - Failed to list deployments for app %d: %v", id, err)
			// Continue with deletion even if we can't list deployments
		} else {
			log.Printf("[API] Found %d deployment(s) for app %d, cleaning up containers and images...", len(appDeployments), id)
			
			ctx := r.Context()
			// Clean up each deployment's containers and images
			for _, deployment := range appDeployments {
				// Stop and remove container if it exists
				if deployment.ContainerID.Valid && deployment.ContainerID.String != "" {
					containerID := deployment.ContainerID.String
					log.Printf("[API] Stopping and removing container: %s", containerID)
					
					// Try to stop the container (ignore error if already stopped)
					if stopErr := runner.Stop(ctx, containerID); stopErr != nil {
						log.Printf("[API] WARNING - Failed to stop container %s: %v (may already be stopped)", containerID, stopErr)
					}
					
					// Remove the container
					if removeErr := runner.Remove(ctx, containerID); removeErr != nil {
						log.Printf("[API] WARNING - Failed to remove container %s: %v", containerID, removeErr)
					} else {
						log.Printf("[API] Container removed successfully: %s", containerID)
					}
				}
				
				// Remove Docker image if it exists
				if deployment.ImageName.Valid && deployment.ImageName.String != "" {
					imageName := deployment.ImageName.String
					log.Printf("[API] Removing Docker image: %s", imageName)
					
					if imageErr := runner.RemoveImage(ctx, imageName); imageErr != nil {
						log.Printf("[API] WARNING - Failed to remove image %s: %v", imageName, imageErr)
					} else {
						log.Printf("[API] Image removed successfully: %s", imageName)
					}
				}
			}
			
			// Delete all deployments from database
			log.Printf("[API] Deleting %d deployment(s) from database...", len(appDeployments))
			for _, deployment := range appDeployments {
				// Note: We don't have a Delete method in deployments.Store yet
				// For now, deployments will be deleted via CASCADE when app is deleted
				// But we could add explicit deletion if needed
			}
		}

		// Finally, delete the app (this will cascade delete deployments if foreign key is set up)
		if err := appStore.Delete(id); err != nil {
			log.Printf("[API] ERROR - Failed to delete app: %v", err)
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		log.Printf("[API] App and all associated resources deleted successfully - ID: %d", id)
		w.WriteHeader(http.StatusNoContent)
	}
}

func listDeployments(store *deployments.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		appID, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			log.Printf("[API] ERROR - Invalid app ID: %s", chi.URLParam(r, "id"))
			respondError(w, http.StatusBadRequest, "Invalid app ID")
			return
		}

		log.Printf("[API] GET /api/v1/apps/%d/deployments - Listing deployments", appID)
		deployments, err := store.ListByAppID(appID)
		if err != nil {
			log.Printf("[API] ERROR - Failed to list deployments: %v", err)
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		log.Printf("[API] Successfully listed %d deployment(s) for app %d", len(deployments), appID)
		respondJSON(w, http.StatusOK, deployments)
	}
}

func getDeployment(store *deployments.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			log.Printf("[API] ERROR - Invalid deployment ID: %s", chi.URLParam(r, "id"))
			respondError(w, http.StatusBadRequest, "Invalid deployment ID")
			return
		}

		log.Printf("[API] GET /api/v1/deployments/%d - Fetching deployment", id)
		deployment, err := store.GetByID(id)
		if err != nil {
			log.Printf("[API] ERROR - Deployment not found: %d", id)
			respondError(w, http.StatusNotFound, "Deployment not found")
			return
		}
		log.Printf("[API] Deployment found - ID: %d, Status: %s", id, deployment.Status)

		respondJSON(w, http.StatusOK, deployment)
	}
}

func getDeploymentLogs(store *deployments.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			log.Printf("[API] ERROR - Invalid deployment ID: %s", chi.URLParam(r, "id"))
			respondError(w, http.StatusBadRequest, "Invalid deployment ID")
			return
		}

		log.Printf("[API] GET /api/v1/deployments/%d/logs - Fetching deployment logs", id)
		deployment, err := store.GetByID(id)
		if err != nil {
			log.Printf("[API] ERROR - Deployment not found: %d", id)
			respondError(w, http.StatusNotFound, "Deployment not found")
			return
		}
		log.Printf("[API] Deployment logs retrieved - ID: %d, Has build log: %v", id, deployment.BuildLog.Valid)

		// Build response with logs
		response := map[string]interface{}{
			"deployment_id": deployment.ID,
			"status":        deployment.Status,
		}

		// Add build log if available
		if deployment.BuildLog.Valid && deployment.BuildLog.String != "" {
			response["build_log"] = deployment.BuildLog.String
		} else {
			response["build_log"] = nil
		}

		// Add error message if available
		if deployment.ErrorMessage.Valid && deployment.ErrorMessage.String != "" {
			response["error_message"] = deployment.ErrorMessage.String
		} else {
			response["error_message"] = nil
		}

		respondJSON(w, http.StatusOK, response)
	}
}

func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	// Ensure CORS headers are set (in case middleware didn't run)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

// getUserID extracts user_id from request context.
// Assumes authentication middleware has set user_id in context.
func getUserID(r *http.Request) (string, bool) {
	// Try different context key formats that might be used
	if userID, ok := r.Context().Value(userIDKey).(string); ok {
		return userID, true
	}
	// Also try string key directly (common pattern)
	if userID, ok := r.Context().Value("user_id").(string); ok {
		return userID, true
	}
	return "", false
}

// listAppsByUser handles GET /api/apps
// Lists all apps owned by the authenticated user.
// Response format:
//
//	[
//	  {
//	    "id": "app_123",
//	    "name": "testapp",
//	    "slug": "testapp",
//	    "status": "Healthy",
//	    "url": "https://testapp.staging.stackyn.com",
//	    "repo_url": "https://github.com/go-chi/chi.git",
//	    "branch": "main",
//	    "created_at": "2025-12-10T14:22:11Z",
//	    "updated_at": "2025-12-17T19:40:00Z"
//	  }
//	]
func listAppsByUser(store *apps.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract user_id from request context
		userID, ok := getUserID(r)
		if !ok {
			respondError(w, http.StatusUnauthorized, "user_id not found in request context")
			return
		}

		// Query apps for this user
		apps, err := store.ListAppsByUserID(r.Context(), userID)
		if err != nil {
			// On DB error, return 500 with JSON error message
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		// Return 200 with JSON array (empty array if none)
		respondJSON(w, http.StatusOK, apps)
	}
}
