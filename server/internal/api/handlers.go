package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
	"stackyn/server/internal/services"
)

// Mock data structures matching frontend types exactly

type App struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	Status    string    `json:"status"`
	URL       string    `json:"url"`
	RepoURL   string    `json:"repo_url"`
	Branch    string    `json:"branch"`
	CreatedAt string    `json:"created_at"`
	UpdatedAt string    `json:"updated_at"`
	Deployment *AppDeployment `json:"deployment,omitempty"`
}

type AppDeployment struct {
	ActiveDeploymentID string                 `json:"active_deployment_id"`
	LastDeployedAt     string                 `json:"last_deployed_at"`
	State              string                 `json:"state"`
	ResourceLimits     *ResourceLimits        `json:"resource_limits,omitempty"`
	UsageStats         *UsageStats            `json:"usage_stats,omitempty"`
}

type ResourceLimits struct {
	MemoryMB int `json:"memory_mb"`
	CPU      int `json:"cpu"`
	DiskGB   int `json:"disk_gb"`
}

type UsageStats struct {
	MemoryUsageMB      int     `json:"memory_usage_mb"`
	MemoryUsagePercent float64 `json:"memory_usage_percent"`
	DiskUsageGB        float64 `json:"disk_usage_gb"`
	DiskUsagePercent   float64 `json:"disk_usage_percent"`
	RestartCount       int     `json:"restart_count"`
}

type Deployment struct {
	ID          int         `json:"id"`
	AppID       int         `json:"app_id"`
	Status      string      `json:"status"`
	ImageName   interface{} `json:"image_name,omitempty"`
	ContainerID interface{} `json:"container_id,omitempty"`
	Subdomain   interface{} `json:"subdomain,omitempty"`
	BuildLog    interface{} `json:"build_log,omitempty"`
	RuntimeLog  interface{} `json:"runtime_log,omitempty"`
	ErrorMessage interface{} `json:"error_message,omitempty"`
	CreatedAt   string      `json:"created_at"`
	UpdatedAt   string      `json:"updated_at"`
}

type DeploymentLogs struct {
	DeploymentID int    `json:"deployment_id"`
	Status      string `json:"status"`
	BuildLog    string `json:"build_log,omitempty"`
	RuntimeLog  string `json:"runtime_log,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

type CreateAppRequest struct {
	Name    string `json:"name"`
	RepoURL string `json:"repo_url"`
	Branch  string `json:"branch"`
}

type CreateAppResponse struct {
	App       App        `json:"app"`
	Deployment Deployment `json:"deployment"`
	Error     string     `json:"error,omitempty"`
}

type EnvVar struct {
	ID        int    `json:"id"`
	AppID     int    `json:"app_id"`
	Key       string `json:"key"`
	Value     string `json:"value"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type CreateEnvVarRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type UserProfile struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	FullName     string    `json:"full_name,omitempty"`
	CompanyName  string    `json:"company_name,omitempty"`
	EmailVerified bool     `json:"email_verified"`
	Plan         string    `json:"plan"`
	CreatedAt    string    `json:"created_at"`
	UpdatedAt    string    `json:"updated_at"`
	Quota        *Quota    `json:"quota,omitempty"`
}

type Quota struct {
	PlanName  string `json:"plan_name"`
	Plan      Plan   `json:"plan"`
	AppCount  int    `json:"app_count"`
	TotalRAMMB int   `json:"total_ram_mb"`
	TotalDiskMB int  `json:"total_disk_mb"`
}

type Plan struct {
	Name            string `json:"name"`
	DisplayName     string `json:"display_name"`
	Price           int    `json:"price"`
	MaxRAMMB        int    `json:"max_ram_mb"`
	MaxDiskMB       int    `json:"max_disk_mb"`
	MaxApps         int    `json:"max_apps"`
	AlwaysOn        bool   `json:"always_on"`
	AutoDeploy      bool   `json:"auto_deploy"`
	HealthChecks    bool   `json:"health_checks"`
	Logs            bool   `json:"logs"`
	ZeroDowntime    bool   `json:"zero_downtime"`
	Workers         bool   `json:"workers"`
	PriorityBuilds  bool   `json:"priority_builds"`
	ManualDeployOnly bool  `json:"manual_deploy_only"`
}

type VerifyTokenRequest struct {
	IDToken string `json:"id_token"`
}

type VerifyTokenResponse struct {
	UID          string `json:"uid"`
	Email        string `json:"email"`
	EmailVerified bool  `json:"email_verified"`
}

type HealthResponse struct {
	Status string `json:"status"`
}

// Handlers

type Handlers struct {
	logger            *zap.Logger
	logPersistence    LogPersistenceService
	containerLogs     ContainerLogService
	planEnforcement   PlanEnforcementService
	billingService    BillingService
	constraintsService ConstraintsService
}

// ConstraintsService interface for constraint enforcement
type ConstraintsService interface {
	ValidateRepoURL(ctx context.Context, repoURL string) error
	ValidateAllConstraints(ctx context.Context, repoURL, repoPath string) error
	ValidateBuildTime(ctx context.Context, buildTimeMinutes int) error
}

// BillingService interface for billing operations
type BillingService interface {
	ProcessLemonSqueezyWebhook(ctx context.Context, event *services.LemonSqueezyWebhookEvent) error
	GetSubscription(ctx context.Context, userID string) (*services.Subscription, error)
}

// LogPersistenceService interface for log persistence
type LogPersistenceService interface {
	PersistLog(ctx context.Context, entry LogEntry) error
	PersistLogStream(ctx context.Context, entry LogEntry, reader io.Reader) error
	GetLogs(ctx context.Context, appID string, logType LogType, limit int, offset int) ([]LogEntry, error)
	DeleteOldLogs(ctx context.Context, appID string, before time.Time) error
}

// ContainerLogService interface for container log streaming
type ContainerLogService interface {
	StreamContainerLogs(ctx context.Context, containerID string, since string, tail string, follow bool) (io.ReadCloser, error)
	GetContainerLogs(ctx context.Context, containerID string, since string, tail string) (string, error)
}

// PlanEnforcementService interface for plan enforcement
type PlanEnforcementService interface {
	CheckMaxApps(ctx context.Context, userID string, currentAppCount int) error
	CheckMaxRAM(ctx context.Context, userID string, requestedRAMMB int) error
	CheckMaxConcurrentBuilds(ctx context.Context, userID string) error
	GetQueuePriority(ctx context.Context, userID string) (int, error)
	IncrementBuildCount(ctx context.Context, userID string) error
	DecrementBuildCount(ctx context.Context, userID string) error
	IncrementRAMUsage(ctx context.Context, userID string, ramMB int) error
	DecrementRAMUsage(ctx context.Context, userID string, ramMB int) error
}

// GetPlanLimitError extracts PlanLimitError from error
func GetPlanLimitError(err error) (*services.PlanLimitError, bool) {
	planErr, ok := err.(*services.PlanLimitError)
	return planErr, ok
}

// GetConstraintError extracts ConstraintError from error
func GetConstraintError(err error) (*services.ConstraintError, bool) {
	constraintErr, ok := err.(*services.ConstraintError)
	return constraintErr, ok
}

// LogEntry represents a log entry (from services package)
type LogEntry struct {
	AppID        string    `json:"app_id"`
	BuildJobID   string    `json:"build_job_id,omitempty"`
	DeploymentID string    `json:"deployment_id,omitempty"`
	LogType      string    `json:"log_type"`
	Timestamp    time.Time `json:"timestamp"`
	Content      string    `json:"content"`
	Size         int64     `json:"size"`
}

// LogType represents the type of log (from services package)
type LogType string

func NewHandlers(logger *zap.Logger, logPersistence LogPersistenceService, containerLogs ContainerLogService, planEnforcement PlanEnforcementService, billingService BillingService, constraintsService ConstraintsService) *Handlers {
	return &Handlers{
		logger:            logger,
		logPersistence:   logPersistence,
		containerLogs:      containerLogs,
		planEnforcement:  planEnforcement,
		billingService:    billingService,
		constraintsService: constraintsService,
	}
}

// getUserIDFromContext extracts user ID from request context
// TODO: Implement proper user extraction from JWT/auth context
func (h *Handlers) getUserIDFromContext(r *http.Request) string {
	// For now, return a placeholder
	// In production, this would extract from JWT token or session
	// Example: userID := r.Context().Value("user_id").(string)
	return "user-1" // Placeholder
}

// getCurrentAppCount gets the current number of apps for a user
// TODO: Query database when DB is connected
func (h *Handlers) getCurrentAppCount(ctx context.Context, userID string) (int, error) {
	// TODO: Query database for app count
	// For now, return 0
	return 0, nil
}

// Helper to write JSON response
func (h *Handlers) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", zap.Error(err))
	}
}

// Helper to write error response
func (h *Handlers) writeError(w http.ResponseWriter, status int, message string) {
	h.writeJSON(w, status, map[string]string{"error": message})
}

// GET /api/apps - List all apps
func (h *Handlers) ListApps(w http.ResponseWriter, r *http.Request) {
	now := time.Now().Format(time.RFC3339)
	apps := []App{
		{
			ID:        "1",
			Name:      "My First App",
			Slug:      "my-first-app",
			Status:    "running",
			URL:       "https://my-first-app.example.com",
			RepoURL:   "https://github.com/user/repo",
			Branch:    "main",
			CreatedAt: now,
			UpdatedAt: now,
			Deployment: &AppDeployment{
				ActiveDeploymentID: "dep_1",
				LastDeployedAt:     now,
				State:              "running",
				ResourceLimits: &ResourceLimits{
					MemoryMB: 512,
					CPU:      1,
					DiskGB:   10,
				},
				UsageStats: &UsageStats{
					MemoryUsageMB:      256,
					MemoryUsagePercent: 50.0,
					DiskUsageGB:        5.0,
					DiskUsagePercent:   50.0,
					RestartCount:       0,
				},
			},
		},
		{
			ID:        "2",
			Name:      "My Second App",
			Slug:      "my-second-app",
			Status:    "deploying",
			URL:       "https://my-second-app.example.com",
			RepoURL:   "https://github.com/user/repo2",
			Branch:    "develop",
			CreatedAt: now,
			UpdatedAt: now,
			Deployment: &AppDeployment{
				ActiveDeploymentID: "dep_2",
				LastDeployedAt:     now,
				State:              "building",
			},
		},
	}
	h.writeJSON(w, http.StatusOK, apps)
}

// GET /api/v1/apps/{id} - Get app by ID
func (h *Handlers) GetAppByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	now := time.Now().Format(time.RFC3339)
	
	app := App{
		ID:        id,
		Name:      "My App",
		Slug:      "my-app",
		Status:    "running",
		URL:       "https://my-app.example.com",
		RepoURL:   "https://github.com/user/repo",
		Branch:    "main",
		CreatedAt: now,
		UpdatedAt: now,
		Deployment: &AppDeployment{
			ActiveDeploymentID: "dep_" + id,
			LastDeployedAt:     now,
			State:              "running",
			ResourceLimits: &ResourceLimits{
				MemoryMB: 512,
				CPU:      1,
				DiskGB:   10,
			},
			UsageStats: &UsageStats{
				MemoryUsageMB:      256,
				MemoryUsagePercent: 50.0,
				DiskUsageGB:        5.0,
				DiskUsagePercent:   50.0,
				RestartCount:       0,
			},
		},
	}
	h.writeJSON(w, http.StatusOK, app)
}

// POST /api/v1/apps - Create app
func (h *Handlers) CreateApp(w http.ResponseWriter, r *http.Request) {
	var req CreateAppRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate MVP constraints - repository URL
	if h.constraintsService != nil {
		if err := h.constraintsService.ValidateRepoURL(r.Context(), req.RepoURL); err != nil {
			if constraintErr, ok := GetConstraintError(err); ok {
				h.writeError(w, http.StatusBadRequest, constraintErr.Message)
				return
			}
			h.writeError(w, http.StatusBadRequest, "Repository URL validation failed")
			return
		}
	}

	// Get user ID from context
	userID := h.getUserIDFromContext(r)

	// Check max apps limit
	if h.planEnforcement != nil {
		currentAppCount, err := h.getCurrentAppCount(r.Context(), userID)
		if err != nil {
			h.logger.Error("Failed to get current app count", zap.Error(err))
			h.writeError(w, http.StatusInternalServerError, "Failed to check plan limits")
			return
		}

		if err := h.planEnforcement.CheckMaxApps(r.Context(), userID, currentAppCount); err != nil {
			if planErr, ok := GetPlanLimitError(err); ok {
				h.writeError(w, http.StatusForbidden, planErr.Message)
				return
			}
			h.writeError(w, http.StatusForbidden, "Plan limit exceeded")
			return
		}
	}

	now := time.Now().Format(time.RFC3339)
	appID := "3"
	deploymentID := 3
	
	app := App{
		ID:        appID,
		Name:      req.Name,
		Slug:      "new-app",
		Status:    "deploying",
		URL:       "https://new-app.example.com",
		RepoURL:   req.RepoURL,
		Branch:    req.Branch,
		CreatedAt: now,
		UpdatedAt: now,
		Deployment: &AppDeployment{
			ActiveDeploymentID: "dep_" + appID,
			LastDeployedAt:     now,
			State:              "building",
		},
	}

	deployment := Deployment{
		ID:        deploymentID,
		AppID:     3,
		Status:    "building",
		CreatedAt: now,
		UpdatedAt: now,
	}

	response := CreateAppResponse{
		App:       app,
		Deployment: deployment,
	}
	h.writeJSON(w, http.StatusCreated, response)
}

// DELETE /api/v1/apps/{id} - Delete app
func (h *Handlers) DeleteApp(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	h.logger.Info("Deleting app", zap.String("id", id))
	w.WriteHeader(http.StatusNoContent)
}

// POST /api/v1/apps/{id}/redeploy - Redeploy app
func (h *Handlers) RedeployApp(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	
	// Get user ID from context
	userID := h.getUserIDFromContext(r)

	// Check max concurrent builds limit
	if h.planEnforcement != nil {
		if err := h.planEnforcement.CheckMaxConcurrentBuilds(r.Context(), userID); err != nil {
			if planErr, ok := GetPlanLimitError(err); ok {
				h.writeError(w, http.StatusForbidden, planErr.Message)
				return
			}
			h.writeError(w, http.StatusForbidden, "Plan limit exceeded")
			return
		}
	}

	now := time.Now().Format(time.RFC3339)
	
	appID, _ := strconv.Atoi(id)
	deploymentID := appID * 10
	
	app := App{
		ID:        id,
		Name:      "My App",
		Slug:      "my-app",
		Status:    "deploying",
		URL:       "https://my-app.example.com",
		RepoURL:   "https://github.com/user/repo",
		Branch:    "main",
		CreatedAt: now,
		UpdatedAt: now,
		Deployment: &AppDeployment{
			ActiveDeploymentID: "dep_" + id,
			LastDeployedAt:     now,
			State:              "building",
		},
	}

	deployment := Deployment{
		ID:        deploymentID,
		AppID:     appID,
		Status:    "building",
		CreatedAt: now,
		UpdatedAt: now,
	}

	response := CreateAppResponse{
		App:       app,
		Deployment: deployment,
	}
	h.writeJSON(w, http.StatusOK, response)
}

// GET /api/v1/apps/{id}/deployments - Get deployments for app
func (h *Handlers) GetAppDeployments(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	appID, _ := strconv.Atoi(id)
	now := time.Now().Format(time.RFC3339)
	
	deployments := []Deployment{
		{
			ID:        1,
			AppID:     appID,
			Status:    "running",
			ImageName: map[string]interface{}{"String": "my-app:latest", "Valid": true},
			ContainerID: map[string]interface{}{"String": "container-123", "Valid": true},
			Subdomain: map[string]interface{}{"String": "my-app", "Valid": true},
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        2,
			AppID:     appID,
			Status:    "failed",
			ImageName: map[string]interface{}{"String": "my-app:v1", "Valid": true},
			ErrorMessage: map[string]interface{}{"String": "Build failed", "Valid": true},
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
	h.writeJSON(w, http.StatusOK, deployments)
}

// GET /api/v1/apps/{id}/env - Get environment variables
func (h *Handlers) GetEnvVars(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	appID, _ := strconv.Atoi(id)
	now := time.Now().Format(time.RFC3339)
	
	envVars := []EnvVar{
		{
			ID:        1,
			AppID:     appID,
			Key:       "NODE_ENV",
			Value:     "production",
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        2,
			AppID:     appID,
			Key:       "API_KEY",
			Value:     "secret-key",
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
	h.writeJSON(w, http.StatusOK, envVars)
}

// POST /api/v1/apps/{id}/env - Create environment variable
func (h *Handlers) CreateEnvVar(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	appID, _ := strconv.Atoi(id)
	
	var req CreateEnvVarRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	now := time.Now().Format(time.RFC3339)
	envVar := EnvVar{
		ID:        3,
		AppID:     appID,
		Key:       req.Key,
		Value:     req.Value,
		CreatedAt: now,
		UpdatedAt: now,
	}
	h.writeJSON(w, http.StatusCreated, envVar)
}

// DELETE /api/v1/apps/{id}/env/{key} - Delete environment variable
func (h *Handlers) DeleteEnvVar(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	key := chi.URLParam(r, "key")
	h.logger.Info("Deleting env var", zap.String("app_id", id), zap.String("key", key))
	w.WriteHeader(http.StatusNoContent)
}

// GET /api/v1/deployments/{id} - Get deployment by ID
func (h *Handlers) GetDeploymentByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	deploymentID, _ := strconv.Atoi(id)
	now := time.Now().Format(time.RFC3339)
	
	deployment := Deployment{
		ID:        deploymentID,
		AppID:     1,
		Status:    "running",
		ImageName: map[string]interface{}{"String": "my-app:latest", "Valid": true},
		ContainerID: map[string]interface{}{"String": "container-123", "Valid": true},
		Subdomain: map[string]interface{}{"String": "my-app", "Valid": true},
		BuildLog: map[string]interface{}{"String": "Building...", "Valid": true},
		RuntimeLog: map[string]interface{}{"String": "Running...", "Valid": true},
		CreatedAt: now,
		UpdatedAt: now,
	}
	h.writeJSON(w, http.StatusOK, deployment)
}

// GET /api/v1/deployments/{id}/logs - Get deployment logs
func (h *Handlers) GetDeploymentLogs(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	deploymentID := id
	
	// Get app ID from query parameter or deployment
	appID := r.URL.Query().Get("app_id")
	if appID == "" {
		h.writeError(w, http.StatusBadRequest, "app_id query parameter required")
		return
	}

	// Get build logs
	buildLogs, err := h.logPersistence.GetLogs(r.Context(), appID, LogType("build"), 100, 0)
	if err != nil {
		h.logger.Warn("Failed to get build logs", zap.Error(err))
	}

	// Get runtime logs
	runtimeLogs, err := h.logPersistence.GetLogs(r.Context(), appID, LogType("runtime"), 100, 0)
	if err != nil {
		h.logger.Warn("Failed to get runtime logs", zap.Error(err))
	}

	// Find logs for this deployment
	var buildLog, runtimeLog string
	for _, log := range buildLogs {
		if log.BuildJobID == deploymentID || log.DeploymentID == deploymentID {
			buildLog = log.Content
			break
		}
	}
	for _, log := range runtimeLogs {
		if log.DeploymentID == deploymentID {
			runtimeLog = log.Content
			break
		}
	}

	logs := DeploymentLogs{
		DeploymentID: 0, // Will be converted from string if needed
		Status:       "running",
		BuildLog:     buildLog,
		RuntimeLog:   runtimeLog,
	}
	h.writeJSON(w, http.StatusOK, logs)
}

// GET /api/v1/apps/{id}/logs/build - Get build logs for an app
func (h *Handlers) GetBuildLogs(w http.ResponseWriter, r *http.Request) {
	appID := chi.URLParam(r, "id")
	
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	
	limit := 100
	offset := 0
	
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil {
			offset = o
		}
	}

	logs, err := h.logPersistence.GetLogs(r.Context(), appID, LogType("build"), limit, offset)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get build logs: %v", err))
		return
	}

	h.writeJSON(w, http.StatusOK, logs)
}

// GET /api/v1/apps/{id}/logs/runtime - Get runtime logs for an app
func (h *Handlers) GetRuntimeLogs(w http.ResponseWriter, r *http.Request) {
	appID := chi.URLParam(r, "id")
	
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	
	limit := 100
	offset := 0
	
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil {
			offset = o
		}
	}

	logs, err := h.logPersistence.GetLogs(r.Context(), appID, LogType("runtime"), limit, offset)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get runtime logs: %v", err))
		return
	}

	h.writeJSON(w, http.StatusOK, logs)
}

// GET /api/v1/apps/{id}/logs/runtime/stream - Stream runtime logs for an app
func (h *Handlers) StreamRuntimeLogs(w http.ResponseWriter, r *http.Request) {
	_ = chi.URLParam(r, "id") // App ID (for future use)
	containerID := r.URL.Query().Get("container_id")
	
	if containerID == "" {
		h.writeError(w, http.StatusBadRequest, "container_id query parameter required")
		return
	}

	since := r.URL.Query().Get("since")
	tail := r.URL.Query().Get("tail")
	if tail == "" {
		tail = "100" // Default to last 100 lines
	}

	follow := r.URL.Query().Get("follow") == "true"

	reader, err := h.containerLogs.StreamContainerLogs(r.Context(), containerID, since, tail, follow)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to stream logs: %v", err))
		return
	}
	defer reader.Close()

	// Set headers for streaming
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	
	if follow {
		w.Header().Set("Transfer-Encoding", "chunked")
	}

	// Stream logs to client
	_, err = io.Copy(w, reader)
	if err != nil {
		h.logger.Warn("Error streaming logs", zap.Error(err))
	}
}

// GET /health - Health check
func (h *Handlers) HealthCheck(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status: "ok",
	}
	h.writeJSON(w, http.StatusOK, response)
}

// POST /api/auth/verify-token - Verify Firebase token
func (h *Handlers) VerifyToken(w http.ResponseWriter, r *http.Request) {
	var req VerifyTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Mock response
	response := VerifyTokenResponse{
		UID:          "user-123",
		Email:        "user@example.com",
		EmailVerified: true,
	}
	h.writeJSON(w, http.StatusOK, response)
}

// GET /api/user/me - Get user profile
func (h *Handlers) GetUserProfile(w http.ResponseWriter, r *http.Request) {
	now := time.Now().Format(time.RFC3339)
	
	profile := UserProfile{
		ID:           "user-123",
		Email:        "user@example.com",
		FullName:     "John Doe",
		CompanyName:  "Acme Corp",
		EmailVerified: true,
		Plan:         "pro",
		CreatedAt:    now,
		UpdatedAt:    now,
		Quota: &Quota{
			PlanName:   "pro",
			AppCount:   2,
			TotalRAMMB: 1024,
			TotalDiskMB: 2048,
			Plan: Plan{
				Name:            "pro",
				DisplayName:     "Pro Plan",
				Price:           29,
				MaxRAMMB:        2048,
				MaxDiskMB:       4096,
				MaxApps:         10,
				AlwaysOn:        true,
				AutoDeploy:      true,
				HealthChecks:    true,
				Logs:            true,
				ZeroDowntime:    true,
				Workers:         true,
				PriorityBuilds:  true,
				ManualDeployOnly: false,
			},
		},
	}
	h.writeJSON(w, http.StatusOK, profile)
}

// POST /api/webhooks/lemon-squeezy - Handle Lemon Squeezy webhook
func (h *Handlers) HandleLemonSqueezyWebhook(w http.ResponseWriter, r *http.Request) {
	// Verify webhook signature (stub - should verify in production)
	// Lemon Squeezy signs webhooks with a secret
	signature := r.Header.Get("X-Signature")
	if signature == "" {
		h.logger.Warn("Lemon Squeezy webhook missing signature")
		// In production, reject unsigned webhooks
		// h.writeError(w, http.StatusUnauthorized, "Missing signature")
		// return
	}

	// TODO: Verify signature using Lemon Squeezy webhook secret
	// secret := os.Getenv("LEMON_SQUEEZY_WEBHOOK_SECRET")
	// if !verifySignature(r.Body, signature, secret) {
	//     h.writeError(w, http.StatusUnauthorized, "Invalid signature")
	//     return
	// }

	// Parse webhook event
	var event services.LemonSqueezyWebhookEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		h.logger.Error("Failed to decode Lemon Squeezy webhook", zap.Error(err))
		h.writeError(w, http.StatusBadRequest, "Invalid webhook payload")
		return
	}

	// Process webhook
	if err := h.billingService.ProcessLemonSqueezyWebhook(r.Context(), &event); err != nil {
		h.logger.Error("Failed to process Lemon Squeezy webhook", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "Failed to process webhook")
		return
	}

	// Return 200 OK to acknowledge receipt
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

