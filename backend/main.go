package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"
)

func main() {
	// Setup routes
	http.HandleFunc("/health", healthHandler)
	
	// Auth routes
	http.HandleFunc("/api/auth/verify-token", verifyTokenHandler)
	http.HandleFunc("/api/auth/login", loginHandler)
	
	// User routes
	http.HandleFunc("/api/user/me", userMeHandler)
	
	// Apps routes
	http.HandleFunc("/api/apps", appsListHandler)
	http.HandleFunc("/api/v1/apps", appsCreateHandler)
	http.HandleFunc("/api/v1/apps/", appsHandler) // Handles /api/v1/apps/{id} and sub-routes
	
	// Deployments routes
	http.HandleFunc("/api/v1/deployments/", deploymentsHandler) // Handles /api/v1/deployments/{id} and sub-routes
	
	// Admin routes
	http.HandleFunc("/admin/users", adminUsersListHandler)
	http.HandleFunc("/admin/users/", adminUsersHandler) // Handles /admin/users/{id} and sub-routes
	http.HandleFunc("/admin/apps", adminAppsListHandler)
	http.HandleFunc("/admin/apps/", adminAppsHandler) // Handles /admin/apps/{id} and sub-routes
	
	// Enable CORS middleware
	handler := corsMiddleware(http.DefaultServeMux)
	
	log.Println("Starting server on :8080")
	log.Fatal(http.ListenAndServe(":8080", handler))
}

// CORS middleware
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// Helper to write JSON response
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// Helper to write error response
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// Helper to check auth token (mock - always succeeds)
func checkAuth(r *http.Request) bool {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return false
	}
	// Mock: accept any Bearer token
	return len(authHeader) > 7 && authHeader[:7] == "Bearer "
}

// Health check endpoint
func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Auth: Verify token endpoint
func verifyTokenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	
	var req struct {
		IDToken string `json:"id_token"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	
	// Mock response
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"uid":           "mock-user-id",
		"email":         "user@example.com",
		"email_verified": true,
	})
}

// Auth: Login endpoint
func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	
	// Mock response
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"user": map[string]string{
			"id":    "mock-admin-id",
			"email": req.Email,
		},
		"token": "mock-bearer-token",
	})
}

// User: Get current user profile
func userMeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	
	if !checkAuth(r) {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	
	// Mock response matching UserProfile schema
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":            "mock-user-id",
		"email":         "user@example.com",
		"full_name":     "Mock User",
		"company_name":  nil,
		"email_verified": true,
		"plan":          "free",
		"created_at":    time.Now().UTC().Format(time.RFC3339),
		"updated_at":    time.Now().UTC().Format(time.RFC3339),
		"quota": map[string]interface{}{
			"plan_name": "free",
			"plan": map[string]interface{}{
				"name":              "free",
				"display_name":      "Free",
				"price":             0,
				"max_ram_mb":        512,
				"max_disk_mb":       1024,
				"max_apps":          1,
				"always_on":         false,
				"auto_deploy":       true,
				"health_checks":     false,
				"logs":              true,
				"zero_downtime":     false,
				"workers":           false,
				"priority_builds":   false,
				"manual_deploy_only": false,
			},
			"app_count":    0,
			"total_ram_mb": 0,
			"total_disk_mb": 0,
		},
	})
}

// Apps: List apps
func appsListHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	
	if !checkAuth(r) {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	
	// Mock response - return empty array
	writeJSON(w, http.StatusOK, []interface{}{})
}

// Apps: Create app
func appsCreateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	
	if !checkAuth(r) {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	
	var req struct {
		Name    string `json:"name"`
		RepoURL string `json:"repo_url"`
		Branch  string `json:"branch"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	
	// Mock response matching CreateAppResponse schema
	now := time.Now().UTC().Format(time.RFC3339)
	appID := "mock-app-1"
	
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"app": map[string]interface{}{
			"id":         appID,
			"name":       req.Name,
			"slug":       "mock-app-1",
			"status":     "building",
			"url":        "https://mock-app-1.example.com",
			"repo_url":   req.RepoURL,
			"branch":     req.Branch,
			"created_at": now,
			"updated_at": now,
			"deployment": map[string]interface{}{
				"active_deployment_id": "1",
				"last_deployed_at":     now,
				"state":                "building",
			},
		},
		"deployment": map[string]interface{}{
			"id":          1,
			"app_id":      1,
			"status":      "building",
			"image_name":  nil,
			"container_id": nil,
			"subdomain":   nil,
			"build_log":   nil,
			"runtime_log": nil,
			"error_message": nil,
			"created_at":  now,
			"updated_at":  now,
		},
	})
}

// Apps: Handle /api/v1/apps/{id} and sub-routes
func appsHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	prefix := "/api/v1/apps/"
	
	if !strings.HasPrefix(path, prefix) {
		writeError(w, http.StatusNotFound, "Not found")
		return
	}
	
	remaining := path[len(prefix):]
	if remaining == "" {
		writeError(w, http.StatusNotFound, "Not found")
		return
	}
	
	if !checkAuth(r) {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	
	// Extract ID and remaining path
	parts := strings.SplitN(remaining, "/", 2)
	id := parts[0]
	subPath := ""
	if len(parts) > 1 {
		subPath = "/" + parts[1]
	}
	
	// Handle sub-routes
	switch {
	case subPath == "/redeploy" && r.Method == http.MethodPost:
		appsRedeployHandler(w, r, id)
		return
	case subPath == "/deployments" && r.Method == http.MethodGet:
		appsDeploymentsHandler(w, r, id)
		return
	case subPath == "/env" && r.Method == http.MethodGet:
		appsEnvVarsHandler(w, r, id)
		return
	case subPath == "/env" && r.Method == http.MethodPost:
		appsCreateEnvVarHandler(w, r, id)
		return
	case strings.HasPrefix(subPath, "/env/") && r.Method == http.MethodDelete:
		key := strings.TrimPrefix(subPath, "/env/")
		appsDeleteEnvVarHandler(w, r, id, key)
		return
	case subPath == "" && r.Method == http.MethodGet:
		appsGetByIdHandler(w, r, id)
		return
	case subPath == "" && r.Method == http.MethodDelete:
		appsDeleteHandler(w, r, id)
		return
	}
	
	writeError(w, http.StatusNotFound, "Not found")
}

// Apps: Get app by ID
func appsGetByIdHandler(w http.ResponseWriter, r *http.Request, id string) {
	now := time.Now().UTC().Format(time.RFC3339)
	
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":         id,
		"name":       "Mock App",
		"slug":       "mock-app",
		"status":     "running",
		"url":        "https://mock-app.example.com",
		"repo_url":   "https://github.com/example/repo",
		"branch":     "main",
		"created_at": now,
		"updated_at": now,
		"deployment": map[string]interface{}{
			"active_deployment_id": "1",
			"last_deployed_at":     now,
			"state":                "running",
			"resource_limits": map[string]interface{}{
				"memory_mb": 512,
				"cpu":       1,
				"disk_gb":   1,
			},
			"usage_stats": map[string]interface{}{
				"memory_usage_mb":      256,
				"memory_usage_percent": 50,
				"disk_usage_gb":        0.5,
				"disk_usage_percent":   50,
				"restart_count":        0,
			},
		},
	})
}

// Apps: Delete app
func appsDeleteHandler(w http.ResponseWriter, r *http.Request, id string) {
	// Return empty response with 200 OK
	w.WriteHeader(http.StatusOK)
}

// Apps: Redeploy app
func appsRedeployHandler(w http.ResponseWriter, r *http.Request, id string) {
	now := time.Now().UTC().Format(time.RFC3339)
	
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"app": map[string]interface{}{
			"id":         id,
			"name":       "Mock App",
			"slug":       "mock-app",
			"status":     "building",
			"url":        "https://mock-app.example.com",
			"repo_url":   "https://github.com/example/repo",
			"branch":     "main",
			"created_at": now,
			"updated_at": now,
			"deployment": map[string]interface{}{
				"active_deployment_id": "2",
				"last_deployed_at":     now,
				"state":                "building",
			},
		},
		"deployment": map[string]interface{}{
			"id":           2,
			"app_id":       1,
			"status":       "building",
			"image_name":   nil,
			"container_id": nil,
			"subdomain":    nil,
			"build_log":    nil,
			"runtime_log":  nil,
			"error_message": nil,
			"created_at":   now,
			"updated_at":   now,
		},
	})
}

// Apps: Get deployments for app
func appsDeploymentsHandler(w http.ResponseWriter, r *http.Request, id string) {
	// Return empty array
	writeJSON(w, http.StatusOK, []interface{}{})
}

// Apps: Get environment variables
func appsEnvVarsHandler(w http.ResponseWriter, r *http.Request, id string) {
	// Return empty array
	writeJSON(w, http.StatusOK, []interface{}{})
}

// Apps: Create environment variable
func appsCreateEnvVarHandler(w http.ResponseWriter, r *http.Request, id string) {
	var req struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	
	now := time.Now().UTC().Format(time.RFC3339)
	
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"id":         1,
		"app_id":     1, // Mock numeric app_id
		"key":        req.Key,
		"value":      req.Value,
		"created_at": now,
		"updated_at": now,
	})
}

// Apps: Delete environment variable
func appsDeleteEnvVarHandler(w http.ResponseWriter, r *http.Request, id string, key string) {
	// Return empty response with 200 OK
	w.WriteHeader(http.StatusOK)
}

// Deployments: Handle /api/v1/deployments/{id} and sub-routes
func deploymentsHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	prefix := "/api/v1/deployments/"
	
	if !strings.HasPrefix(path, prefix) {
		writeError(w, http.StatusNotFound, "Not found")
		return
	}
	
	remaining := path[len(prefix):]
	if remaining == "" {
		writeError(w, http.StatusNotFound, "Not found")
		return
	}
	
	if !checkAuth(r) {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	
	// Extract ID and remaining path
	parts := strings.SplitN(remaining, "/", 2)
	id := parts[0]
	subPath := ""
	if len(parts) > 1 {
		subPath = "/" + parts[1]
	}
	
	if subPath == "/logs" && r.Method == http.MethodGet {
		deploymentsLogsHandler(w, r, id)
		return
	}
	
	if subPath == "" && r.Method == http.MethodGet {
		deploymentsGetByIdHandler(w, r, id)
		return
	}
	
	writeError(w, http.StatusNotFound, "Not found")
}

// Deployments: Get deployment by ID
func deploymentsGetByIdHandler(w http.ResponseWriter, r *http.Request, id string) {
	now := time.Now().UTC().Format(time.RFC3339)
	
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":           1,
		"app_id":       1,
		"status":       "running",
		"image_name":   nil,
		"container_id": nil,
		"subdomain":    nil,
		"build_log":    nil,
		"runtime_log":  nil,
		"error_message": nil,
		"created_at":   now,
		"updated_at":   now,
	})
}

// Deployments: Get deployment logs
func deploymentsLogsHandler(w http.ResponseWriter, r *http.Request, id string) {
	// Parse id as number for deployment_id field
	var deploymentID int
	if id != "" {
		deploymentID = 1 // Mock value
	}
	
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"deployment_id": deploymentID,
		"status":        "running",
		"build_log":     nil,
		"runtime_log":   nil,
		"error_message": nil,
	})
}

// Admin: List users
func adminUsersListHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	
	if !checkAuth(r) {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	
	limit := r.URL.Query().Get("limit")
	offset := r.URL.Query().Get("offset")
	_ = limit
	_ = offset
	
	// Mock response
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"users": []interface{}{},
		"total":  0,
		"limit":  50,
		"offset": 0,
	})
}

// Admin: Handle /admin/users/{id} and sub-routes
func adminUsersHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	prefix := "/admin/users/"
	
	if !strings.HasPrefix(path, prefix) {
		writeError(w, http.StatusNotFound, "Not found")
		return
	}
	
	remaining := path[len(prefix):]
	if remaining == "" {
		writeError(w, http.StatusNotFound, "Not found")
		return
	}
	
	if !checkAuth(r) {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	
	// Extract ID and remaining path
	parts := strings.SplitN(remaining, "/", 2)
	id := parts[0]
	subPath := ""
	if len(parts) > 1 {
		subPath = "/" + parts[1]
	}
	
	if subPath == "/plan" && r.Method == http.MethodPatch {
		adminUsersUpdatePlanHandler(w, r, id)
		return
	}
	
	if subPath == "" && r.Method == http.MethodGet {
		adminUsersGetByIdHandler(w, r, id)
		return
	}
	
	writeError(w, http.StatusNotFound, "Not found")
}

// Admin: Get user by ID
func adminUsersGetByIdHandler(w http.ResponseWriter, r *http.Request, id string) {
	now := time.Now().UTC().Format(time.RFC3339)
	
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":             id,
		"email":          "user@example.com",
		"full_name":      "Mock User",
		"company_name":   nil,
		"email_verified": true,
		"plan":           "free",
		"is_admin":       false,
		"created_at":     now,
		"updated_at":     now,
		"quota": map[string]interface{}{
			"plan_name": "free",
			"plan": map[string]interface{}{
				"name":               "free",
				"display_name":       "Free",
				"price":              0,
				"max_ram_mb":         512,
				"max_disk_mb":        1024,
				"max_apps":           1,
				"always_on":          false,
				"auto_deploy":        true,
				"health_checks":      false,
				"logs":               true,
				"zero_downtime":      false,
				"workers":            false,
				"priority_builds":    false,
				"manual_deploy_only": false,
			},
			"app_count":     0,
			"total_ram_mb":  0,
			"total_disk_mb": 0,
		},
	})
}

// Admin: Update user plan
func adminUsersUpdatePlanHandler(w http.ResponseWriter, r *http.Request, id string) {
	var req struct {
		Plan string `json:"plan"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Plan updated",
		"user_id": id,
		"plan":    req.Plan,
	})
}

// Admin: List apps
func adminAppsListHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	
	if !checkAuth(r) {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	
	limit := r.URL.Query().Get("limit")
	offset := r.URL.Query().Get("offset")
	_ = limit
	_ = offset
	
	// Mock response
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"apps":  []interface{}{},
		"total": 0,
		"limit": 50,
		"offset": 0,
	})
}

// Admin: Handle /admin/apps/{id} and sub-routes
func adminAppsHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	prefix := "/admin/apps/"
	
	if !strings.HasPrefix(path, prefix) {
		writeError(w, http.StatusNotFound, "Not found")
		return
	}
	
	remaining := path[len(prefix):]
	if remaining == "" {
		writeError(w, http.StatusNotFound, "Not found")
		return
	}
	
	if !checkAuth(r) {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	
	// Extract ID and remaining path
	parts := strings.SplitN(remaining, "/", 2)
	id := parts[0]
	subPath := ""
	if len(parts) > 1 {
		subPath = "/" + parts[1]
	}
	
	switch {
	case subPath == "/stop" && r.Method == http.MethodPost:
		adminAppsStopHandler(w, r, id)
		return
	case subPath == "/start" && r.Method == http.MethodPost:
		adminAppsStartHandler(w, r, id)
		return
	case subPath == "/redeploy" && r.Method == http.MethodPost:
		adminAppsRedeployHandler(w, r, id)
		return
	}
	
	writeError(w, http.StatusNotFound, "Not found")
}

// Admin: Stop app
func adminAppsStopHandler(w http.ResponseWriter, r *http.Request, id string) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":          "App stopped",
		"app_id":           1,
		"stopped_containers": 1,
	})
}

// Admin: Start app
func adminAppsStartHandler(w http.ResponseWriter, r *http.Request, id string) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "App started",
		"app_id":  1,
	})
}

// Admin: Redeploy app
func adminAppsRedeployHandler(w http.ResponseWriter, r *http.Request, id string) {
	now := time.Now().UTC().Format(time.RFC3339)
	
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "App redeployed",
		"app_id":  1,
		"deployment": map[string]interface{}{
			"id":           1,
			"app_id":       1,
			"status":       "building",
			"image_name":   nil,
			"container_id": nil,
			"subdomain":    nil,
			"build_log":    nil,
			"runtime_log":  nil,
			"error_message": nil,
			"created_at":   now,
			"updated_at":   now,
		},
	})
}

