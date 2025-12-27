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
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"mvp-be/internal/apps"
	"mvp-be/internal/auth"
	"mvp-be/internal/config"
	"mvp-be/internal/db"
	"mvp-be/internal/deployments"
	"mvp-be/internal/dockerrun"
	"mvp-be/internal/envvars"
	"mvp-be/internal/firebase"
	"mvp-be/internal/gitrepo"
	"mvp-be/internal/logs"
	"mvp-be/internal/users"
)


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
	envVarStore := envvars.NewStore(database.DB)
	userStore := users.NewStore(database.DB)
	log.Println("Data stores initialized")

	// Initialize Firebase Auth service
	log.Println("Initializing Firebase Auth service...")
	firebaseService, err := firebase.NewService()
	if err != nil {
		log.Printf("WARNING - Failed to initialize Firebase Auth: %v (authentication will not work)", err)
		firebaseService = nil
	} else {
		log.Println("Firebase Auth service initialized")
	}

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

	// Public authentication endpoints (no auth required)
	r.Route("/api/auth", func(r chi.Router) {
		r.Post("/signup", signup(userStore)) // Legacy endpoint - keep for backward compatibility
		r.Post("/login", login(userStore))
		// Firebase Auth signup flow endpoints
		r.Post("/signup/firebase", signupFirebase(firebaseService, userStore))
		r.Post("/signup/complete", signupCompleteFirebase(firebaseService, userStore))
		r.Post("/verify-token", verifyFirebaseToken(firebaseService))
	})

	// Protected API routes (require authentication)
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(createAuthMiddleware(firebaseService, userStore)) // All routes under /api/v1 require authentication

		// Apps endpoints
		r.Route("/apps", func(r chi.Router) {
			r.Post("/", createApp(appStore, deploymentStore, cloner))
			r.Get("/{id}", getApp(appStore, deploymentStore, runner))
			r.Delete("/{id}", deleteApp(appStore, deploymentStore, runner))
			r.Post("/{id}/redeploy", redeployApp(appStore, deploymentStore, cloner))
			r.Get("/{id}/deployments", listDeployments(deploymentStore))
			// Environment variables endpoints
			r.Get("/{id}/env", listEnvVars(envVarStore))
			r.Post("/{id}/env", createEnvVar(envVarStore))
			r.Delete("/{id}/env/{key}", deleteEnvVar(envVarStore))
		})

		// Deployments endpoints
		r.Route("/deployments", func(r chi.Router) {
			r.Get("/{id}", getDeployment(deploymentStore))
			r.Get("/{id}/logs", getDeploymentLogs(deploymentStore, runner))
		})
	})

	// Authenticated endpoint for listing apps by user (GET /api/apps)
	r.Route("/api/apps", func(r chi.Router) {
		r.Use(createAuthMiddleware(firebaseService, userStore))
		r.Get("/", listAppsByUser(appStore))
	})

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
	log.Println("  POST /api/auth/signup - Sign up new user (legacy)")
	log.Println("  POST /api/auth/signup/firebase - Create Firebase user")
	log.Println("  POST /api/auth/signup/complete - Complete signup with details")
	log.Println("  POST /api/auth/verify-token - Verify Firebase token")
	log.Println("  POST /api/auth/login - Login user")
	log.Println("  GET  /api/apps - List apps for authenticated user (protected)")
	log.Println("  POST /api/v1/apps - Create new app (protected)")
	log.Println("  GET  /api/v1/apps/{id} - Get app by ID (protected)")
	log.Println("  DELETE /api/v1/apps/{id} - Delete app (protected)")
	log.Println("  POST /api/v1/apps/{id}/redeploy - Redeploy app (protected)")
	log.Println("  GET  /api/v1/apps/{id}/deployments - List deployments (protected)")
	log.Println("  GET  /api/v1/deployments/{id} - Get deployment (protected)")
	log.Println("  GET  /api/v1/deployments/{id}/logs - Get deployment logs (protected)")
	log.Println("  GET  /api/v1/apps/{id}/env - List environment variables (protected)")
	log.Println("  POST /api/v1/apps/{id}/env - Create/update environment variable (protected)")
	log.Println("  DELETE /api/v1/apps/{id}/env/{key} - Delete environment variable (protected)")
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

		// Get user_id from context (set by auth middleware)
		userID, ok := auth.GetUserID(r)
		if !ok {
			log.Printf("[API] ERROR - User ID not found in context")
			respondError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		// Check app limit (3 apps per user)
		appCount, err := appStore.CountByUserID(r.Context(), userID)
		if err != nil {
			log.Printf("[API] ERROR - Failed to count user apps: %v", err)
			respondError(w, http.StatusInternalServerError, "Failed to check app limit")
			return
		}
		if appCount >= 3 {
			log.Printf("[API] ERROR - User %s has reached app limit (3 apps)", userID)
			respondJSON(w, http.StatusForbidden, map[string]interface{}{
				"error": "App limit reached. You can only have up to 3 apps. Please delete an existing app to create a new one.",
				"app":   nil,
			})
			return
		}

		// Create app first
		log.Printf("[API] Creating app in database for user: %s (current count: %d/3)", userID, appCount)
		app, err := appStore.Create(userID, req.Name, req.RepoURL, req.Branch)
		if err != nil {
			log.Printf("[API] ERROR - Failed to create app: %v", err)
			// Check if it's a duplicate app name error
			if strings.Contains(err.Error(), "an app with this name already exists") {
				respondJSON(w, http.StatusConflict, map[string]interface{}{
					"error": "An app with this name already exists. Please choose a different name.",
					"app":   nil,
				})
				return
			}
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

func getApp(appStore *apps.Store, deploymentStore *deployments.Store, runner *dockerrun.Runner) http.HandlerFunc {
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
			
			deploymentInfo := map[string]interface{}{
				"active_deployment_id": activeDeploymentID,
				"last_deployed_at":     activeDeployment.UpdatedAt,
				"state":                state,
			}
			
			// Try to get resource limits and usage stats from Docker container if it exists
			if activeDeployment.ContainerID.Valid && activeDeployment.ContainerID.String != "" {
				ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				defer cancel()
				
				memoryLimitMB, cpuLimit, diskLimitGB, limitsErr := runner.GetResourceLimits(ctx, activeDeployment.ContainerID.String)
				if limitsErr == nil {
					deploymentInfo["resource_limits"] = map[string]interface{}{
						"memory_mb": memoryLimitMB,
						"cpu":       cpuLimit,
						"disk_gb":   diskLimitGB,
					}
					log.Printf("[API] Resource limits retrieved - Memory: %d MB, CPU: %.2f, Disk: %d GB", 
						memoryLimitMB, cpuLimit, diskLimitGB)
					
					// Get usage stats
					usageStats, usageErr := runner.GetContainerUsageStats(ctx, activeDeployment.ContainerID.String, memoryLimitMB, diskLimitGB)
					if usageErr == nil {
						deploymentInfo["usage_stats"] = map[string]interface{}{
							"memory_usage_mb":     usageStats.MemoryUsageMB,
							"memory_usage_percent": usageStats.MemoryUsagePercent,
							"disk_usage_gb":        usageStats.DiskUsageGB,
							"disk_usage_percent":   usageStats.DiskUsagePercent,
							"restart_count":        usageStats.RestartCount,
						}
						log.Printf("[API] Usage stats retrieved - Memory: %d MB (%.1f%%), Disk: %.2f GB (%.1f%%), Restarts: %d",
							usageStats.MemoryUsageMB, usageStats.MemoryUsagePercent,
							usageStats.DiskUsageGB, usageStats.DiskUsagePercent, usageStats.RestartCount)
					} else {
						log.Printf("[API] WARNING - Failed to get usage stats: %v", usageErr)
					}
				} else {
					log.Printf("[API] WARNING - Failed to get resource limits: %v", limitsErr)
				}
			}
			
			response["deployment"] = deploymentInfo
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

		// Use a background context with timeout for cleanup operations
		// This ensures cleanup completes even if the HTTP request is cancelled
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		// Step 1: Get all deployments for this app
		appDeployments, err := deploymentStore.ListByAppID(id)
		if err != nil {
			log.Printf("[API] WARNING - Failed to list deployments for app %d: %v", id, err)
			// Continue with deletion even if we can't list deployments
		} else {
			log.Printf("[API] Found %d deployment(s) for app %d", len(appDeployments), id)
			
			// Step 2: Stop all Docker containers first
			log.Printf("[API] Step 1: Stopping all Docker containers...")
			stoppedContainers := make([]string, 0)
			for i := range appDeployments {
				deployment := appDeployments[i]
				if deployment.ContainerID.Valid && deployment.ContainerID.String != "" {
					containerID := deployment.ContainerID.String
					log.Printf("[API] Attempting to stop container: %s (deployment ID: %d)", containerID, deployment.ID)
					
					// Stop the container
					if stopErr := runner.Stop(ctx, containerID); stopErr != nil {
						log.Printf("[API] ERROR - Failed to stop container %s: %v", containerID, stopErr)
						// Try to stop by container name as fallback
						containerName := fmt.Sprintf("app-%d-%d", id, deployment.ID)
						log.Printf("[API] Attempting fallback: stopping container by name: %s", containerName)
						if nameStopErr := runner.Stop(ctx, containerName); nameStopErr != nil {
							log.Printf("[API] ERROR - Failed to stop container by name %s: %v", containerName, nameStopErr)
						} else {
							log.Printf("[API] Container stopped successfully by name: %s", containerName)
							stoppedContainers = append(stoppedContainers, containerName)
						}
					} else {
						log.Printf("[API] Container stopped successfully: %s", containerID)
						stoppedContainers = append(stoppedContainers, containerID)
					}
				} else {
					log.Printf("[API] WARNING - Deployment %d has no container ID stored", deployment.ID)
				}
			}
			
			// Wait a moment for containers to fully stop
			if len(stoppedContainers) > 0 {
				log.Printf("[API] Waiting 2 seconds for containers to fully stop...")
				time.Sleep(2 * time.Second)
			}
			
			// Step 3: Remove all containers (after they're stopped)
			log.Printf("[API] Step 1.5: Removing all Docker containers...")
			for i := range appDeployments {
				deployment := appDeployments[i]
				if deployment.ContainerID.Valid && deployment.ContainerID.String != "" {
					containerID := deployment.ContainerID.String
					log.Printf("[API] Attempting to remove container: %s (deployment ID: %d)", containerID, deployment.ID)
					
					if removeErr := runner.Remove(ctx, containerID); removeErr != nil {
						log.Printf("[API] ERROR - Failed to remove container %s: %v", containerID, removeErr)
						// Try to remove by container name as fallback
						containerName := fmt.Sprintf("app-%d-%d", id, deployment.ID)
						log.Printf("[API] Attempting fallback: removing container by name: %s", containerName)
						if nameRemoveErr := runner.Remove(ctx, containerName); nameRemoveErr != nil {
							log.Printf("[API] ERROR - Failed to remove container by name %s: %v", containerName, nameRemoveErr)
						} else {
							log.Printf("[API] Container removed successfully by name: %s", containerName)
						}
					} else {
						log.Printf("[API] Container removed successfully: %s", containerID)
					}
				} else {
					log.Printf("[API] WARNING - Deployment %d has no container ID stored, trying by name", deployment.ID)
					containerName := fmt.Sprintf("app-%d-%d", id, deployment.ID)
					log.Printf("[API] Attempting to remove container by name: %s", containerName)
					if nameRemoveErr := runner.Remove(ctx, containerName); nameRemoveErr != nil {
						log.Printf("[API] ERROR - Failed to remove container by name %s: %v", containerName, nameRemoveErr)
					} else {
						log.Printf("[API] Container removed successfully by name: %s", containerName)
					}
				}
			}
			
			// Wait a moment for containers to be fully removed before deleting images
			log.Printf("[API] Waiting 1 second before deleting images...")
			time.Sleep(1 * time.Second)
			
			// Step 4: Delete all Docker images (after containers are stopped and removed)
			log.Printf("[API] Step 2: Deleting all Docker images...")
			deletedImages := 0
			failedImages := 0
			for i := range appDeployments {
				deployment := appDeployments[i]
				if deployment.ImageName.Valid && deployment.ImageName.String != "" {
					imageName := deployment.ImageName.String
					log.Printf("[API] Attempting to delete Docker image: %s (deployment ID: %d)", imageName, deployment.ID)
					
					if imageErr := runner.RemoveImage(ctx, imageName); imageErr != nil {
						log.Printf("[API] ERROR - Failed to delete image %s: %v", imageName, imageErr)
						failedImages++
					} else {
						log.Printf("[API] Image deleted successfully: %s", imageName)
						deletedImages++
					}
				} else {
					log.Printf("[API] WARNING - Deployment %d has no image name stored", deployment.ID)
				}
			}
			
			log.Printf("[API] Docker cleanup summary for app %d: %d images deleted, %d failed", id, deletedImages, failedImages)
			
			// Step 4.5: Clean up cloned repository directories
			// Note: This cleanup attempts to remove repos from /tmp/mvp-deployments.
			// If API and worker run in separate containers without shared volumes,
			// this may fail silently (logged as warning). In that case, repos should
			// be cleaned up manually or via a shared volume/cleanup job.
			log.Printf("[API] Step 2.5: Cleaning up cloned repository directories...")
			cleanedRepos := 0
			failedRepos := 0
			// Worker clones repos to /tmp/mvp-deployments/deployment-{deploymentID}
			workerWorkDir := "/tmp/mvp-deployments"
			for i := range appDeployments {
				deployment := appDeployments[i]
				repoDir := fmt.Sprintf("%s/deployment-%d", workerWorkDir, deployment.ID)
				log.Printf("[API] Attempting to remove cloned repository: %s (deployment ID: %d)", repoDir, deployment.ID)
				
				if err := os.RemoveAll(repoDir); err != nil {
					log.Printf("[API] WARNING - Failed to remove cloned repository %s: %v (may be in different container)", repoDir, err)
					failedRepos++
				} else {
					log.Printf("[API] Cloned repository removed successfully: %s", repoDir)
					cleanedRepos++
				}
			}
			log.Printf("[API] Repository cleanup summary for app %d: %d repos cleaned, %d failed", id, cleanedRepos, failedRepos)
		}

		// Step 5: Finally, delete the app from PostgreSQL database (this will cascade delete deployments)
		log.Printf("[API] Step 3: Removing app entry from PostgreSQL database...")
		if err := appStore.Delete(id); err != nil {
			log.Printf("[API] ERROR - Failed to delete app from database: %v", err)
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

func getDeploymentLogs(store *deployments.Store, runner *dockerrun.Runner) http.HandlerFunc {
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
		log.Printf("[API] Deployment logs retrieved - ID: %d, Has build log: %v, Has runtime log: %v", id, deployment.BuildLog.Valid, deployment.RuntimeLog.Valid)

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

		// For runtime logs, try to fetch fresh logs from Docker if container is running
		runtimeLog := ""
		if deployment.Status == deployments.StatusRunning && deployment.ContainerID.Valid && deployment.ContainerID.String != "" {
			// Try to fetch fresh runtime logs from Docker
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			
			log.Printf("[API] Fetching fresh runtime logs from container %s (includes stdout/stderr from application)", deployment.ContainerID.String)
			// Fetch all logs (up to last 500 lines) to include console.log output from the application
			runtimeLogReader, fetchErr := runner.GetLogs(ctx, deployment.ContainerID.String, "500")
			if fetchErr == nil {
				parsedLog, parseErr := logs.ParseRuntimeLog(runtimeLogReader)
				if parseErr == nil {
					if parsedLog != "" {
						runtimeLog = parsedLog
						// Update the database with fresh logs
						if updateErr := store.UpdateRuntimeLog(id, runtimeLog); updateErr != nil {
							log.Printf("[API] WARNING - Failed to update runtime log in database: %v", updateErr)
						}
						log.Printf("[API] Fresh runtime logs fetched successfully (length: %d chars, contains application stdout/stderr)", len(runtimeLog))
					} else {
						log.Printf("[API] Runtime logs are empty (container may not have produced any output yet)")
					}
				} else {
					log.Printf("[API] WARNING - Failed to parse fresh runtime logs: %v", parseErr)
				}
			} else {
				log.Printf("[API] WARNING - Failed to fetch fresh runtime logs: %v", fetchErr)
			}
		}
		
		// Use fresh logs if available, otherwise fall back to stored logs
		if runtimeLog != "" {
			response["runtime_log"] = runtimeLog
		} else if deployment.RuntimeLog.Valid && deployment.RuntimeLog.String != "" {
			response["runtime_log"] = deployment.RuntimeLog.String
		} else {
			response["runtime_log"] = nil
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

// createAuthMiddleware creates an authentication middleware that supports both JWT (legacy) and Firebase tokens
func createAuthMiddleware(firebaseService *firebase.Service, userStore *users.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get the Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Authorization header required", http.StatusUnauthorized)
				return
			}

			// Check if it starts with "Bearer "
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
				return
			}

			tokenString := parts[1]
			var userID string

			// Try to verify as JWT token first (legacy)
			claims, err := auth.VerifyToken(tokenString)
			if err == nil {
				// Legacy JWT token - use user_id from claims
				userID = claims.UserID
				log.Printf("[AUTH] Verified legacy JWT token for user: %s", userID)
			} else {
				// Try to verify as Firebase token
				ctx := r.Context()
				var uid, email string
				var firebaseErr error

				// Try using Admin SDK first, fallback to REST API verification
				if firebaseService != nil {
					uid, email, firebaseErr = firebaseService.VerifyIDToken(ctx, tokenString)
				} else {
					// Use REST API verification (no Admin SDK required)
					cfg := config.Load()
					uid, email, firebaseErr = firebase.VerifyIDTokenREST(ctx, tokenString, cfg.FirebaseProjectID)
				}

				if firebaseErr != nil {
					log.Printf("[AUTH] Token verification failed (both JWT and Firebase): JWT error: %v, Firebase error: %v", err, firebaseErr)
					http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
					return
				}

				// Firebase token verified - get user_id from database by email
				user, dbErr := userStore.GetUserByEmail(email)
				if dbErr != nil {
					log.Printf("[AUTH] Firebase token verified but user not found in database: %s, error: %v", email, dbErr)
					// If user doesn't exist in database, use Firebase UID as fallback
					// This handles cases where user was created in Firebase but not yet in our DB
					userID = uid
				} else {
					userID = user.ID
				}
				log.Printf("[AUTH] Verified Firebase token for user: %s (email: %s)", userID, email)
			}

			// Set user_id in context
			ctx := context.WithValue(r.Context(), auth.GetUserIDKey(), userID)
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		})
	}
}

// signup handles POST /api/auth/signup
func signup(store *users.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if req.Email == "" || req.Password == "" {
			respondError(w, http.StatusBadRequest, "email and password are required")
			return
		}

		// Check if user already exists
		_, err := store.GetUserByEmail(req.Email)
		if err == nil {
			respondError(w, http.StatusConflict, "Email already registered")
			return
		}

		// Create new user
		user, err := store.CreateUser(req.Email, req.Password)
		if err != nil {
			log.Printf("[API] ERROR - Failed to create user: %v", err)
			respondError(w, http.StatusInternalServerError, "Failed to create user")
			return
		}

		// Generate JWT token
		token, err := auth.GenerateToken(user.ID, user.Email)
		if err != nil {
			log.Printf("[API] ERROR - Failed to generate token: %v", err)
			respondError(w, http.StatusInternalServerError, "Failed to generate token")
			return
		}

		respondJSON(w, http.StatusCreated, map[string]interface{}{
			"user": map[string]interface{}{
				"id":    user.ID,
				"email": user.Email,
			},
			"token": token,
		})
	}
}

// login handles POST /api/auth/login
func login(store *users.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if req.Email == "" || req.Password == "" {
			respondError(w, http.StatusBadRequest, "email and password are required")
			return
		}

		// Get user by email
		user, err := store.GetUserByEmail(req.Email)
		if err != nil {
			log.Printf("[API] ERROR - User not found: %v", err)
			respondError(w, http.StatusUnauthorized, "Invalid email or password")
			return
		}

		// Verify password
		if !store.VerifyPassword(user, req.Password) {
			log.Printf("[API] ERROR - Invalid password for user: %s", req.Email)
			respondError(w, http.StatusUnauthorized, "Invalid email or password")
			return
		}

		// Generate JWT token
		token, err := auth.GenerateToken(user.ID, user.Email)
		if err != nil {
			log.Printf("[API] ERROR - Failed to generate token: %v", err)
			respondError(w, http.StatusInternalServerError, "Failed to generate token")
			return
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"user": map[string]interface{}{
				"id":    user.ID,
				"email": user.Email,
			},
			"token": token,
		})
	}
}

// signupFirebase handles POST /api/auth/signup/firebase
// Step 1: User creates Firebase account with email/password
// Firebase handles email verification automatically
func signupFirebase(firebaseService *firebase.Service, userStore *users.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if firebaseService == nil {
			respondError(w, http.StatusServiceUnavailable, "Firebase Auth not configured")
			return
		}

		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if req.Email == "" || req.Password == "" {
			respondError(w, http.StatusBadRequest, "email and password are required")
			return
		}

		// Basic email validation
		if !strings.Contains(req.Email, "@") {
			respondError(w, http.StatusBadRequest, "Invalid email format")
			return
		}

		// Check if user already exists in our database
		_, err := userStore.GetUserByEmail(req.Email)
		if err == nil {
			respondError(w, http.StatusConflict, "Email already registered")
			return
		}

		// Create Firebase user
		ctx := r.Context()
		firebaseUser, err := firebaseService.CreateUser(ctx, req.Email, req.Password)
		if err != nil {
			log.Printf("[API] ERROR - Failed to create Firebase user: %v", err)
			// Check if it's a duplicate email error
			if strings.Contains(err.Error(), "email already exists") {
				respondError(w, http.StatusConflict, "Email already registered")
				return
			}
			respondError(w, http.StatusInternalServerError, "Failed to create user")
			return
		}

		log.Printf("[API] Firebase user created: %s (UID: %s)", req.Email, firebaseUser.UID)

		respondJSON(w, http.StatusCreated, map[string]interface{}{
			"message": "User created successfully. Please verify your email.",
			"uid":     firebaseUser.UID,
			"email":   firebaseUser.Email,
		})
	}
}

// signupCompleteFirebase handles POST /api/auth/signup/complete
// Step 2: User provides account details after email verification
// Requires Firebase ID token to verify the user
func signupCompleteFirebase(firebaseService *firebase.Service, userStore *users.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			IDToken     string `json:"id_token"` // Firebase ID token
			FullName    string `json:"full_name"`
			CompanyName string `json:"company_name"`
			Email       string `json:"email"` // Email from verified Firebase user (optional)
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if req.IDToken == "" || req.FullName == "" {
			respondError(w, http.StatusBadRequest, "id_token and full_name are required")
			return
		}

		// Verify Firebase ID token
		ctx := r.Context()
		var uid, email string
		var err error

		// Try using Admin SDK first, fallback to REST API verification
		if firebaseService != nil {
			uid, email, err = firebaseService.VerifyIDToken(ctx, req.IDToken)
			if err != nil {
				log.Printf("[API] ERROR - Failed to verify Firebase token via Admin SDK: %v", err)
				respondError(w, http.StatusUnauthorized, "Invalid or expired token")
				return
			}

			// Get Firebase user to check email verification status
			firebaseUser, err := firebaseService.GetUserByEmail(ctx, email)
			if err == nil {
				if !firebaseUser.EmailVerified {
					respondError(w, http.StatusBadRequest, "Email not verified. Please verify your email first.")
					return
				}
			}
		} else {
			// Use REST API verification (no Admin SDK required)
			cfg := config.Load()
			uid, email, err = firebase.VerifyIDTokenREST(ctx, req.IDToken, cfg.FirebaseProjectID)
			if err != nil {
				log.Printf("[API] ERROR - Failed to verify Firebase token via REST: %v", err)
				respondError(w, http.StatusUnauthorized, "Invalid or expired token")
				return
			}
			log.Printf("[API] Verified Firebase token via REST API for user: %s", email)
			// For REST API, we trust the frontend has verified the email
			// since we can't check it without Admin SDK
		}

		// Use email from request if provided (from frontend), otherwise use from token
		if req.Email != "" {
			email = req.Email
		}


		// Check if user already exists in our database
		existingUser, err := userStore.GetUserByEmail(email)
		if err == nil {
			// User exists, update their details
			// For now, we'll just return success
			log.Printf("[API] User already exists: %s", email)
			respondJSON(w, http.StatusOK, map[string]interface{}{
				"message": "User already exists",
				"user": map[string]interface{}{
					"id":             existingUser.ID,
					"email":          existingUser.Email,
					"full_name":      existingUser.FullName,
					"company_name":   existingUser.CompanyName,
					"email_verified": existingUser.EmailVerified,
				},
			})
			return
		}

		// Create user in our database
		// Use Firebase UID as the user ID, or generate a new UUID
		// We'll store Firebase UID separately or use it as the primary ID
		user, err := userStore.CreateUserWithDetails(
			email,
			"", // No password needed - Firebase handles auth
			req.FullName,
			req.CompanyName,
			true, // email_verified = true (verified by Firebase)
		)
		if err != nil {
			log.Printf("[API] ERROR - Failed to create user: %v", err)
			respondError(w, http.StatusInternalServerError, "Failed to create user")
			return
		}

		// Generate our JWT token for API access
		token, err := auth.GenerateToken(user.ID, user.Email)
		if err != nil {
			log.Printf("[API] ERROR - Failed to generate token: %v", err)
			respondError(w, http.StatusInternalServerError, "Failed to generate token")
			return
		}

		respondJSON(w, http.StatusCreated, map[string]interface{}{
			"user": map[string]interface{}{
				"id":             user.ID,
				"email":          user.Email,
				"full_name":      user.FullName,
				"company_name":   user.CompanyName,
				"email_verified": user.EmailVerified,
			},
			"token": token,
			"firebase_uid": uid,
		})
	}
}

// verifyFirebaseToken handles POST /api/auth/verify-token
// Verifies a Firebase ID token and returns user info
func verifyFirebaseToken(firebaseService *firebase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			IDToken string `json:"id_token"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if req.IDToken == "" {
			respondError(w, http.StatusBadRequest, "id_token is required")
			return
		}

		// Verify Firebase ID token
		ctx := r.Context()
		var uid, email string
		var emailVerified bool
		var err error

		// Try using Admin SDK first, fallback to REST API verification
		if firebaseService != nil {
			uid, email, err = firebaseService.VerifyIDToken(ctx, req.IDToken)
			if err != nil {
				log.Printf("[API] ERROR - Failed to verify Firebase token: %v", err)
				respondError(w, http.StatusUnauthorized, "Invalid or expired token")
				return
			}

			// Get Firebase user details
			firebaseUser, err := firebaseService.GetUserByEmail(ctx, email)
			if err == nil {
				emailVerified = firebaseUser.EmailVerified
			}
		} else {
			// Use REST API verification (no Admin SDK required)
			cfg := config.Load()
			uid, email, err = firebase.VerifyIDTokenREST(ctx, req.IDToken, cfg.FirebaseProjectID)
			if err != nil {
				log.Printf("[API] ERROR - Failed to verify Firebase token via REST: %v", err)
				respondError(w, http.StatusUnauthorized, "Invalid or expired token")
				return
			}
			// For REST API, we can't check email verification status
			// Default to true since frontend handles verification
			emailVerified = true
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"uid":            uid,
			"email":          email,
			"email_verified": emailVerified,
		})
	}
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
		// Extract user_id from request context (set by auth middleware)
		userID, ok := auth.GetUserID(r)
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

// listEnvVars handles GET /api/v1/apps/{id}/env
// Lists all environment variables for an app.
func listEnvVars(store *envvars.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		appID, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			log.Printf("[API] ERROR - Invalid app ID: %s", chi.URLParam(r, "id"))
			respondError(w, http.StatusBadRequest, "Invalid app ID")
			return
		}

		log.Printf("[API] GET /api/v1/apps/%d/env - Listing environment variables", appID)
		envVars, err := store.GetByAppID(appID)
		if err != nil {
			log.Printf("[API] ERROR - Failed to list environment variables: %v", err)
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		log.Printf("[API] Successfully listed %d environment variable(s) for app %d", len(envVars), appID)
		respondJSON(w, http.StatusOK, envVars)
	}
}

// createEnvVar handles POST /api/v1/apps/{id}/env
// Creates or updates an environment variable for an app.
func createEnvVar(store *envvars.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		appID, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			log.Printf("[API] ERROR - Invalid app ID: %s", chi.URLParam(r, "id"))
			respondError(w, http.StatusBadRequest, "Invalid app ID")
			return
		}

		log.Printf("[API] POST /api/v1/apps/%d/env - Creating/updating environment variable", appID)
		var req struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Printf("[API] ERROR - Invalid request body: %v", err)
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if req.Key == "" {
			log.Printf("[API] ERROR - Missing required field: key")
			respondError(w, http.StatusBadRequest, "key is required")
			return
		}

		log.Printf("[API] Request - Key: %s, Value: [REDACTED]", req.Key)
		envVar, err := store.Create(appID, req.Key, req.Value)
		if err != nil {
			log.Printf("[API] ERROR - Failed to create environment variable: %v", err)
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		log.Printf("[API] Environment variable created/updated successfully - ID: %d, Key: %s", envVar.ID, envVar.Key)
		respondJSON(w, http.StatusOK, envVar)
	}
}

// deleteEnvVar handles DELETE /api/v1/apps/{id}/env/{key}
// Deletes an environment variable for an app.
func deleteEnvVar(store *envvars.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		appID, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			log.Printf("[API] ERROR - Invalid app ID: %s", chi.URLParam(r, "id"))
			respondError(w, http.StatusBadRequest, "Invalid app ID")
			return
		}

		key := chi.URLParam(r, "key")
		if key == "" {
			log.Printf("[API] ERROR - Missing key parameter")
			respondError(w, http.StatusBadRequest, "key parameter is required")
			return
		}

		log.Printf("[API] DELETE /api/v1/apps/%d/env/%s - Deleting environment variable", appID, key)
		if err := store.Delete(appID, key); err != nil {
			log.Printf("[API] ERROR - Failed to delete environment variable: %v", err)
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		log.Printf("[API] Environment variable deleted successfully - App ID: %d, Key: %s", appID, key)
		w.WriteHeader(http.StatusNoContent)
	}
}
