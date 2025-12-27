// Package admin provides admin middleware and utilities for admin-only endpoints.
package admin

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"mvp-be/internal/apps"
	"mvp-be/internal/auth"
	"mvp-be/internal/deployments"
	"mvp-be/internal/dockerrun"
	"mvp-be/internal/quota"
	"mvp-be/internal/users"
)

// AdminMiddleware creates middleware that requires admin role
func AdminMiddleware(userStore *users.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get user_id from context (set by auth middleware)
			userID, ok := auth.GetUserID(r)
			if !ok {
				log.Printf("[ADMIN] ERROR - User ID not found in context")
				respondError(w, http.StatusUnauthorized, "Unauthorized")
				return
			}

			// Get user to check admin status
			user, err := userStore.GetUserByID(userID)
			if err != nil {
				log.Printf("[ADMIN] ERROR - Failed to get user: %v", err)
				respondError(w, http.StatusUnauthorized, "Unauthorized")
				return
			}

			if !user.IsAdmin {
				log.Printf("[ADMIN] ERROR - User %s attempted to access admin endpoint", userID)
				respondError(w, http.StatusForbidden, "Forbidden: Admin access required")
				return
			}

			// User is admin, proceed
			next.ServeHTTP(w, r)
		})
	}
}

// AdminUserService handles admin user management operations
type AdminUserService struct {
	userStore   *users.Store
	quotaService *quota.Service
}

func NewAdminUserService(userStore *users.Store, quotaService *quota.Service) *AdminUserService {
	return &AdminUserService{
		userStore:   userStore,
		quotaService: quotaService,
	}
}

// ListUsers handles GET /admin/users
func (s *AdminUserService) ListUsers(w http.ResponseWriter, r *http.Request) {
	// Parse pagination params
	limit := 50 // default
	offset := 0
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Parse search param
	searchEmail := r.URL.Query().Get("search")

	// Get users
	userList, err := s.userStore.ListUsers(limit, offset, searchEmail)
	if err != nil {
		log.Printf("[ADMIN] ERROR - Failed to list users: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to list users")
		return
	}

	// Get total count for pagination
	total, err := s.userStore.CountUsers(searchEmail)
	if err != nil {
		log.Printf("[ADMIN] ERROR - Failed to count users: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to count users")
		return
	}

	// Get quota info for each user
	type UserWithQuota struct {
		*users.User
		Quota *quota.UserQuota `json:"quota,omitempty"`
	}

	usersWithQuota := make([]UserWithQuota, 0, len(userList))
	for _, u := range userList {
		userQuota, err := s.quotaService.GetUserQuota(r.Context(), u.ID)
		if err != nil {
			log.Printf("[ADMIN] WARNING - Failed to get quota for user %s: %v", u.ID, err)
			// Continue without quota info
		}
		uwq := UserWithQuota{User: u}
		if userQuota != nil {
			uwq.Quota = userQuota
		}
		usersWithQuota = append(usersWithQuota, uwq)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"users": usersWithQuota,
		"total": total,
		"limit": limit,
		"offset": offset,
	})
}

// GetUser handles GET /admin/users/{id}
func (s *AdminUserService) GetUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	if userID == "" {
		respondError(w, http.StatusBadRequest, "User ID required")
		return
	}

	user, err := s.userStore.GetUserByID(userID)
	if err != nil {
		if err == sql.ErrNoRows {
			respondError(w, http.StatusNotFound, "User not found")
			return
		}
		log.Printf("[ADMIN] ERROR - Failed to get user: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to get user")
		return
	}

	// Get quota info
	userQuota, err := s.quotaService.GetUserQuota(r.Context(), userID)
	if err != nil {
		log.Printf("[ADMIN] WARNING - Failed to get quota for user %s: %v", userID, err)
	}

	response := map[string]interface{}{
		"id":             user.ID,
		"email":          user.Email,
		"full_name":      user.FullName,
		"company_name":   user.CompanyName,
		"email_verified": user.EmailVerified,
		"plan":           user.Plan,
		"is_admin":       user.IsAdmin,
		"created_at":     user.CreatedAt,
		"updated_at":     user.UpdatedAt,
	}

	if userQuota != nil {
		response["quota"] = userQuota
	}

	respondJSON(w, http.StatusOK, response)
}

// UpdateUserPlan handles PATCH /admin/users/{id}/plan
func (s *AdminUserService) UpdateUserPlan(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	if userID == "" {
		respondError(w, http.StatusBadRequest, "User ID required")
		return
	}

	var req struct {
		Plan string `json:"plan"`
	}

	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate plan name
	if err := quota.ValidatePlanName(req.Plan); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid plan name")
		return
	}

	// Update plan
	if err := s.userStore.UpdatePlan(userID, req.Plan); err != nil {
		log.Printf("[ADMIN] ERROR - Failed to update user plan: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to update plan")
		return
	}

	// Get updated user with quota info
	user, err := s.userStore.GetUserByID(userID)
	if err != nil {
		log.Printf("[ADMIN] WARNING - Failed to get updated user: %v", err)
		// Still return success, but without user details
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"message": "Plan updated successfully",
			"user_id": userID,
			"plan": req.Plan,
		})
		return
	}

	// Get updated quota info
	userQuota, err := s.quotaService.GetUserQuota(r.Context(), userID)
	if err != nil {
		log.Printf("[ADMIN] WARNING - Failed to get updated quota: %v", err)
	}

	// Build response with updated user and quota
	response := map[string]interface{}{
		"message": "Plan updated successfully",
		"user_id": userID,
		"plan": req.Plan,
		"user": map[string]interface{}{
			"id":             user.ID,
			"email":          user.Email,
			"full_name":      user.FullName,
			"company_name":   user.CompanyName,
			"email_verified": user.EmailVerified,
			"plan":           user.Plan,
			"is_admin":       user.IsAdmin,
			"created_at":     user.CreatedAt,
			"updated_at":     user.UpdatedAt,
		},
	}

	if userQuota != nil {
		response["quota"] = userQuota
	}

	respondJSON(w, http.StatusOK, response)
}

// AdminAppService handles admin app management operations
type AdminAppService struct {
	appStore        *apps.Store
	deploymentStore *deployments.Store
	runner          *dockerrun.Runner
}

func NewAdminAppService(appStore *apps.Store, deploymentStore *deployments.Store, runner *dockerrun.Runner) *AdminAppService {
	return &AdminAppService{
		appStore:        appStore,
		deploymentStore: deploymentStore,
		runner:          runner,
	}
}

// ListApps handles GET /admin/apps
func (s *AdminAppService) ListApps(w http.ResponseWriter, r *http.Request) {
	// Parse pagination params
	limit := 50
	offset := 0
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Get all apps
	allApps, err := s.appStore.List()
	if err != nil {
		log.Printf("[ADMIN] ERROR - Failed to list apps: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to list apps")
		return
	}

	// Apply pagination
	total := len(allApps)
	start := offset
	end := offset + limit
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	var paginatedApps []*apps.App
	if start < total {
		paginatedApps = allApps[start:end]
	}

	// Enrich with deployment info
	type AppWithDetails struct {
		*apps.App
		DeploymentCount int    `json:"deployment_count"`
		LatestStatus    string `json:"latest_status"`
	}

	appsWithDetails := make([]AppWithDetails, 0, len(paginatedApps))
	for _, app := range paginatedApps {
		appID, _ := strconv.Atoi(app.ID)
		deployments, err := s.deploymentStore.ListByAppID(appID)
		if err != nil {
			log.Printf("[ADMIN] WARNING - Failed to get deployments for app %s: %v", app.ID, err)
		}

		latestStatus := "none"
		if len(deployments) > 0 {
			latestStatus = string(deployments[0].Status)
		}

		appsWithDetails = append(appsWithDetails, AppWithDetails{
			App:             app,
			DeploymentCount: len(deployments),
			LatestStatus:    latestStatus,
		})
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"apps":   appsWithDetails,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// StopApp handles POST /admin/apps/{id}/stop
func (s *AdminAppService) StopApp(w http.ResponseWriter, r *http.Request) {
	appIDStr := chi.URLParam(r, "id")
	appID, err := strconv.Atoi(appIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid app ID")
		return
	}

	// Get app to verify it exists
	_, err = s.appStore.GetByID(appID)
	if err != nil {
		if err == sql.ErrNoRows {
			respondError(w, http.StatusNotFound, "App not found")
			return
		}
		log.Printf("[ADMIN] ERROR - Failed to get app: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to get app")
		return
	}

	// Get running deployments
	deploys, err := s.deploymentStore.GetRunningByAppID(appID)
	if err != nil {
		log.Printf("[ADMIN] ERROR - Failed to get deployments: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to get deployments")
		return
	}

	// Stop all running containers
	ctx := r.Context()
	stoppedCount := 0
	for _, dep := range deploys {
		if dep.ContainerID.Valid && dep.ContainerID.String != "" {
			if err := s.runner.Stop(ctx, dep.ContainerID.String); err != nil {
				log.Printf("[ADMIN] WARNING - Failed to stop container %s: %v", dep.ContainerID.String, err)
			} else {
				stoppedCount++
				// Update deployment status
				s.deploymentStore.UpdateStatus(dep.ID, deployments.StatusStopped)
			}
		}
	}

	// Update app status
	s.appStore.UpdateStatus(appID, "Stopped")

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":      "App stopped successfully",
		"app_id":       appID,
		"stopped_containers": stoppedCount,
	})
}

// StartApp handles POST /admin/apps/{id}/start
// Note: This triggers a redeploy since we don't have a simple container start method
func (s *AdminAppService) StartApp(w http.ResponseWriter, r *http.Request) {
	// For now, starting an app means triggering a redeploy
	// This ensures the container is properly configured and started
	s.RedeployApp(w, r)
}

// RedeployApp handles POST /admin/apps/{id}/redeploy
func (s *AdminAppService) RedeployApp(w http.ResponseWriter, r *http.Request) {
	appIDStr := chi.URLParam(r, "id")
	appID, err := strconv.Atoi(appIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid app ID")
		return
	}

	// Get app to verify it exists
	_, err = s.appStore.GetByID(appID)
	if err != nil {
		if err == sql.ErrNoRows {
			respondError(w, http.StatusNotFound, "App not found")
			return
		}
		log.Printf("[ADMIN] ERROR - Failed to get app: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to get app")
		return
	}

	// Create new deployment
	deployment, err := s.deploymentStore.Create(appID)
	if err != nil {
		log.Printf("[ADMIN] ERROR - Failed to create deployment: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to create deployment")
		return
	}

	// Update app status
	s.appStore.UpdateStatus(appID, "Pending")

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"message":    "Redeployment triggered",
		"app_id":     appID,
		"deployment": deployment,
	})
}

// Helper functions
func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

func decodeJSON(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}

