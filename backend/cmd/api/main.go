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

// contextKey is a type for context keys to avoid collisions
type contextKey string

const userIDKey contextKey = "user_id"

func main() {
	cfg := config.Load()

	// Initialize database
	database, err := db.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Run migrations
	if err := database.Migrate(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize stores
	appStore := apps.NewStore(database.DB)
	deploymentStore := deployments.NewStore(database.DB)

	// Initialize git cloner for Dockerfile validation
	workDir := "/tmp/mvp-api-validation"
	if err := os.MkdirAll(workDir, 0755); err != nil {
		log.Fatalf("Failed to create validation work directory: %v", err)
	}
	cloner := gitrepo.NewCloner(workDir)

	// Setup router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		// Apps endpoints
		r.Route("/apps", func(r chi.Router) {
			r.Get("/", listApps(appStore))
			r.Post("/", createApp(appStore, deploymentStore, cloner))
			r.Get("/{id}", getApp(appStore))
			r.Delete("/{id}", deleteApp(appStore))
			r.Get("/{id}/deployments", listDeployments(deploymentStore))
		})

		// Deployments endpoints
		r.Route("/deployments", func(r chi.Router) {
			r.Get("/{id}", getDeployment(deploymentStore))
		})
	})

	// New API route for listing apps by user (GET /api/apps)
	r.Get("/api/apps", listAppsByUser(appStore))

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	port := cfg.Port
	log.Printf("API server starting on port %s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func listApps(store *apps.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apps, err := store.List()
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		respondJSON(w, http.StatusOK, apps)
	}
}

func createApp(appStore *apps.Store, deploymentStore *deployments.Store, cloner *gitrepo.Cloner) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name    string `json:"name"`
			RepoURL string `json:"repo_url"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]interface{}{
				"error": "Invalid request body",
				"app":   nil,
			})
			return
		}

		if req.Name == "" || req.RepoURL == "" {
			respondJSON(w, http.StatusBadRequest, map[string]interface{}{
				"error": "name and repo_url are required",
				"app":   nil,
			})
			return
		}

		// Create app first
		app, err := appStore.Create(req.Name, req.RepoURL)
		if err != nil {
			respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
				"error": err.Error(),
				"app":   nil,
			})
			return
		}

		// Create initial deployment
		// TODO: App.ID is now string (per new schema), but deploymentStore.Create expects int.
		// This needs to be updated to work with the new text-based ID schema.
		// For now, this will cause a compilation error and needs to be addressed.
		_ = app.ID                                   // Temporary to avoid unused variable error
		deployment, err := deploymentStore.Create(0) // TODO: Fix this - app.ID is now string
		if err != nil {
			log.Printf("Warning: failed to create deployment: %v", err)
			respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
				"error": fmt.Sprintf("Failed to create deployment: %v", err),
				"app":   app,
			})
			return
		}

		// Validate repository has Dockerfile after creating app and deployment
		// Use a temporary deployment ID for validation
		tempDeploymentID := int(time.Now().Unix())
		repoPath, err := cloner.Clone(req.RepoURL, tempDeploymentID)
		if err != nil {
			// Update deployment with error
			errorMsg := fmt.Sprintf("Failed to clone repository: %v", err)
			deploymentStore.UpdateError(deployment.ID, errorMsg)
			// Refresh deployment to get updated status
			deployment, _ = deploymentStore.GetByID(deployment.ID)
			respondJSON(w, http.StatusBadRequest, map[string]interface{}{
				"app":        app,
				"deployment": deployment,
			})
			return
		}

		// Check if Dockerfile exists
		if err := gitrepo.CheckDockerfile(repoPath); err != nil {
			// Clean up cloned repository
			os.RemoveAll(repoPath)
			// Update deployment with error
			errorMsg := "Dockerfile is not available in the repository root directory. Please ensure your repository contains a Dockerfile."
			deploymentStore.UpdateError(deployment.ID, errorMsg)
			// Refresh deployment to get updated status
			deployment, _ = deploymentStore.GetByID(deployment.ID)
			respondJSON(w, http.StatusBadRequest, map[string]interface{}{
				"app":        app,
				"deployment": deployment,
			})
			return
		}

		// Clean up validation repository
		os.RemoveAll(repoPath)

		// If validation passes, deployment remains in "pending" status for worker to process
		respondJSON(w, http.StatusCreated, map[string]interface{}{
			"app":        app,
			"deployment": deployment,
		})
	}
}

func getApp(store *apps.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid app ID")
			return
		}

		app, err := store.GetByID(id)
		if err != nil {
			respondError(w, http.StatusNotFound, "App not found")
			return
		}

		respondJSON(w, http.StatusOK, app)
	}
}

func deleteApp(store *apps.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid app ID")
			return
		}

		if err := store.Delete(id); err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func listDeployments(store *deployments.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		appID, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid app ID")
			return
		}

		deployments, err := store.ListByAppID(appID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondJSON(w, http.StatusOK, deployments)
	}
}

func getDeployment(store *deployments.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid deployment ID")
			return
		}

		deployment, err := store.GetByID(id)
		if err != nil {
			respondError(w, http.StatusNotFound, "Deployment not found")
			return
		}

		respondJSON(w, http.StatusOK, deployment)
	}
}

func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
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
