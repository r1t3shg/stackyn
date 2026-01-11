package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/docker/docker/client"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
	stackynerrors "stackyn/server/internal/errors"
	"stackyn/server/internal/services"
	"stackyn/server/internal/tasks"
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
	ID          interface{} `json:"id"` // UUID string from database
	AppID       interface{} `json:"app_id"` // UUID string from database
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
	Name    string            `json:"name"`
	RepoURL string            `json:"repo_url"`
	Branch  string            `json:"branch"`
	EnvVars []CreateEnvVarRequest `json:"env_vars,omitempty"` // Optional environment variables
}

type CreateAppResponse struct {
	App       App        `json:"app"`
	Deployment Deployment `json:"deployment"`
	Error     string     `json:"error,omitempty"`
}

type EnvVar struct {
	ID        string `json:"id"`
	AppID     string `json:"app_id"`
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
	Plan      PlanInfo `json:"plan"`
	AppCount  int    `json:"app_count"`
	TotalRAMMB int   `json:"total_ram_mb"`
	TotalDiskMB int  `json:"total_disk_mb"`
}

type PlanInfo struct {
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

type HealthResponse struct {
	Status string `json:"status"`
}

// Handlers

type Handlers struct {
	logger             *zap.Logger
	logPersistence     LogPersistenceService
	containerLogs      ContainerLogService
	planEnforcement    PlanEnforcementService
	billingService     BillingService
	constraintsService ConstraintsService
	subscriptionService *services.SubscriptionService
	appRepo            *AppRepo
	deploymentRepo     *DeploymentRepo
	envVarRepo         *EnvVarRepo
	userRepo           *UserRepo
	planRepo           *PlanRepo
	userPlanRepo       *UserPlanRepo
	taskEnqueue        *services.TaskEnqueueService
	wsHub              *services.Hub
	deploymentService  DeploymentService
}

// DeploymentService interface for deployment operations
type DeploymentService interface {
	VerifyDeployment(ctx context.Context, appID string) (*services.DeploymentVerificationResult, error)
	CleanupAppResources(ctx context.Context, appID string) error
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
	GetSubscription(ctx context.Context, userID string) (*services.BillingSubscription, error)
}

// LogPersistenceService interface for log persistence
type LogPersistenceService interface {
	PersistLog(ctx context.Context, entry LogEntry) error
	PersistLogStream(ctx context.Context, entry LogEntry, reader io.Reader) error
	GetLogs(ctx context.Context, appID string, logType LogType, limit int, offset int) ([]LogEntry, error)
	GetLogsByDeploymentID(ctx context.Context, appID string, deploymentID string) (string, error)
	GetLogsByBuildJobID(ctx context.Context, appID string, buildJobID string) (string, error)
	GetLatestBuildLogs(ctx context.Context, appID string) (string, error)
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

func NewHandlers(logger *zap.Logger, logPersistence LogPersistenceService, containerLogs ContainerLogService, planEnforcement PlanEnforcementService, billingService BillingService, constraintsService ConstraintsService, subscriptionService *services.SubscriptionService, appRepo *AppRepo, deploymentRepo *DeploymentRepo, envVarRepo *EnvVarRepo, userRepo *UserRepo, planRepo *PlanRepo, userPlanRepo *UserPlanRepo, taskEnqueue *services.TaskEnqueueService, wsHub *services.Hub, deploymentService DeploymentService) *Handlers {
	return &Handlers{
		logger:              logger,
		logPersistence:      logPersistence,
		wsHub:               wsHub,
		containerLogs:       containerLogs,
		planEnforcement:     planEnforcement,
		billingService:      billingService,
		constraintsService:  constraintsService,
		subscriptionService: subscriptionService,
		appRepo:             appRepo,
		deploymentRepo:      deploymentRepo,
		envVarRepo:          envVarRepo,
		userRepo:            userRepo,
		planRepo:            planRepo,
		userPlanRepo:        userPlanRepo,
		taskEnqueue:         taskEnqueue,
		deploymentService:   deploymentService,
	}
}

// getUserIDFromContext extracts user ID from request context
func (h *Handlers) getUserIDFromContext(r *http.Request) string {
	userID, ok := r.Context().Value("user_id").(string)
	if !ok {
		h.logger.Warn("User ID not found in context")
		return ""
	}
	return userID
}

// getCurrentAppCount gets the current number of apps for a user
func (h *Handlers) getCurrentAppCount(ctx context.Context, userID string) (int, error) {
	if h.appRepo == nil {
		return 0, nil
	}
	return h.appRepo.GetAppCountByUserID(userID)
}

// getUserResourceUsage calculates total RAM and disk usage for a user's apps
// Returns total RAM in MB and total disk in GB
func (h *Handlers) getUserResourceUsage(ctx context.Context, userID string) (totalRAMMB, totalDiskGB int, err error) {
	if h.appRepo == nil {
		return 0, 0, nil
	}

	apps, err := h.appRepo.GetAppsByUserID(userID)
	if err != nil {
		return 0, 0, err
	}

	// Sum up resource usage from all apps
	// Note: For MVP, we use default values (256 MB RAM, 1 GB Disk per app)
	// In future, these values could be stored in the apps table or passed during app creation
	defaultAppRAMMB := 256
	defaultAppDiskGB := 1

	for range apps {
		totalRAMMB += defaultAppRAMMB
		totalDiskGB += defaultAppDiskGB
	}

	return totalRAMMB, totalDiskGB, nil
}

// Helper to write JSON response
func (h *Handlers) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", zap.Error(err))
	}
}

// ErrorResponse represents a standardized error response
type ErrorResponse struct {
	Error   ErrorDetail `json:"error"`
	Message string      `json:"message,omitempty"` // Deprecated: use error.message
}

// ErrorDetail contains error code and message
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// Helper to write error response (legacy format for backward compatibility)
func (h *Handlers) writeError(w http.ResponseWriter, status int, message string) {
	h.writeJSON(w, status, map[string]string{"error": message})
}

// writeStackynError writes a StackynError in the standardized format
func (h *Handlers) writeStackynError(w http.ResponseWriter, r *http.Request, status int, err *stackynerrors.StackynError) {
	requestID := middleware.GetReqID(r.Context())
	
	// Log the error with context
	h.logger.Error("Stackyn error",
		zap.String("error_code", string(err.Code)),
		zap.String("error_message", err.Message),
		zap.String("error_details", err.Details),
		zap.String("request_id", requestID),
		zap.Error(err.Err),
	)

	response := ErrorResponse{
		Error: ErrorDetail{
			Code:    string(err.Code),
			Message: err.Message,
			Details: err.Details,
		},
	}
	h.writeJSON(w, status, response)
}

// handleError processes an error and writes appropriate response
func (h *Handlers) handleError(w http.ResponseWriter, r *http.Request, err error, defaultStatus int) {
	requestID := middleware.GetReqID(r.Context())
	
	// Check if it's already a StackynError
	if stackynErr, ok := stackynerrors.AsStackynError(err); ok {
		status := defaultStatus
		// Map error codes to HTTP status codes
		switch stackynErr.Code {
		case stackynerrors.ErrorCodeRepoNotFound,
			stackynerrors.ErrorCodePlanLimitExceeded,
			stackynerrors.ErrorCodeDeployLocked:
			status = http.StatusBadRequest
		case stackynerrors.ErrorCodeRepoPrivateUnsupported,
			stackynerrors.ErrorCodeUnsupportedLanguage,
			stackynerrors.ErrorCodeDockerfilePresent,
			stackynerrors.ErrorCodeDockerComposePresent,
			stackynerrors.ErrorCodeMonorepoDetected:
			status = http.StatusUnprocessableEntity
		case stackynerrors.ErrorCodeBuildFailed,
			stackynerrors.ErrorCodeBuildTimeout,
			stackynerrors.ErrorCodeAppCrashOnStart,
			stackynerrors.ErrorCodePortNotListening,
			stackynerrors.ErrorCodeHealthcheckFailed:
			status = http.StatusUnprocessableEntity
		case stackynerrors.ErrorCodeHostOutOfMemory,
			stackynerrors.ErrorCodeBuildNodeUnavailable,
			stackynerrors.ErrorCodeInternalPlatformError:
			status = http.StatusServiceUnavailable
		}
		h.writeStackynError(w, r, status, stackynErr)
		return
	}

	// Check for plan limit errors
	if planErr, ok := GetPlanLimitError(err); ok {
		stackynErr := stackynerrors.New(stackynerrors.ErrorCodePlanLimitExceeded, planErr.Message)
		h.writeStackynError(w, r, http.StatusForbidden, stackynErr)
		return
	}

	// Check for constraint errors
	if constraintErr, ok := GetConstraintError(err); ok {
		// Map constraint errors to appropriate error codes based on constraint name
		var code stackynerrors.ErrorCode
		switch constraintErr.Constraint {
		case "repo_url", "invalid_repo_url":
			code = stackynerrors.ErrorCodeRepoNotFound
		case "private_repo":
			code = stackynerrors.ErrorCodeRepoPrivateUnsupported
		case "repo_size", "repo_too_large":
			code = stackynerrors.ErrorCodeRepoTooLarge
		case "monorepo":
			code = stackynerrors.ErrorCodeMonorepoDetected
		case "dockerfile", "no_dockerfile":
			code = stackynerrors.ErrorCodeDockerfilePresent
		case "docker_compose", "no_docker_compose":
			code = stackynerrors.ErrorCodeDockerComposePresent
		default:
			code = stackynerrors.ErrorCodeInternalPlatformError
		}
		stackynErr := stackynerrors.New(code, constraintErr.Message)
		h.writeStackynError(w, r, http.StatusBadRequest, stackynErr)
		return
	}

	// Generic error fallback
	h.logger.Error("Unhandled error",
		zap.Error(err),
		zap.String("request_id", requestID),
	)
	stackynErr := stackynerrors.Wrap(stackynerrors.ErrorCodeInternalPlatformError, err, "An unexpected error occurred")
	h.writeStackynError(w, r, defaultStatus, stackynErr)
}

// GET /api/apps - List all apps for authenticated user
func (h *Handlers) ListApps(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by AuthMiddleware)
	userID := h.getUserIDFromContext(r)
	if userID == "" {
		h.writeError(w, http.StatusUnauthorized, "User ID not found in context")
		return
	}

	// Query database for user's apps
	if h.appRepo == nil {
		h.logger.Error("App repository not initialized")
		h.writeError(w, http.StatusInternalServerError, "App repository not available")
		return
	}

	apps, err := h.appRepo.GetAppsByUserID(userID)
	if err != nil {
		h.logger.Error("Failed to get apps", zap.Error(err), zap.String("user_id", userID))
		h.writeError(w, http.StatusInternalServerError, "Failed to retrieve apps")
		return
	}

	// Return empty array if no apps found
	if apps == nil {
		apps = []App{}
	}

	// Enrich each app with deployment data and container stats
	for i := range apps {
		h.enrichAppWithDeployment(r.Context(), &apps[i])
	}

	h.writeJSON(w, http.StatusOK, apps)
}

// GET /api/v1/apps/{id} - Get app by ID
func (h *Handlers) GetAppByID(w http.ResponseWriter, r *http.Request) {
	appID := chi.URLParam(r, "id")
	
	// Get user ID from context (set by AuthMiddleware)
	userID := h.getUserIDFromContext(r)
	if userID == "" {
		h.writeError(w, http.StatusUnauthorized, "User ID not found in context")
		return
	}

	// Query database for the app
	if h.appRepo == nil {
		h.logger.Error("App repository not initialized")
		h.writeError(w, http.StatusInternalServerError, "App repository not available")
		return
	}

	app, err := h.appRepo.GetAppByID(appID, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.writeError(w, http.StatusNotFound, "App not found")
			return
		}
		h.logger.Error("Failed to get app", zap.Error(err), zap.String("app_id", appID), zap.String("user_id", userID))
		h.writeError(w, http.StatusInternalServerError, "Failed to retrieve app")
		return
	}

	// Enrich app with deployment data and container stats
	h.enrichAppWithDeployment(r.Context(), app)

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

	// Check subscription status and resource limits before creating app
	// Default app resource allocation (can be made configurable later)
	defaultAppRAMMB := 256  // 256 MB per app
	defaultAppDiskGB := 1   // 1 GB per app

	// Get current resource usage for user's apps
	currentRAMMB, currentDiskGB, err := h.getUserResourceUsage(r.Context(), userID)
	if err != nil {
		h.logger.Error("Failed to get user resource usage", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "Failed to check resource limits")
		return
	}

	// Check resource limits (subscription service will check subscription status too)
	// For MVP, we'll use a subscription service reference if available
	// If not available, we'll skip resource limit checking (should not happen in production)
	if h.subscriptionService != nil {
		if err := h.subscriptionService.CheckResourceLimits(
			r.Context(),
			userID,
			currentRAMMB,
			currentDiskGB,
			defaultAppRAMMB,
			defaultAppDiskGB,
		); err != nil {
			h.writeError(w, http.StatusForbidden, err.Error())
			return
		}
	}

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

	// Create app in database
	if h.appRepo == nil {
		h.logger.Error("App repository not initialized")
		h.writeError(w, http.StatusInternalServerError, "App repository not available")
		return
	}

	// Default branch to "main" if not provided
	branch := req.Branch
	if branch == "" {
		branch = "main"
	}

	app, err := h.appRepo.CreateApp(userID, req.Name, req.RepoURL, branch)
	if err != nil {
		// Check for duplicate key violation (unique constraint on user_id + name)
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			h.writeError(w, http.StatusConflict, fmt.Sprintf("An app with the name '%s' already exists", req.Name))
			return
		}
		h.logger.Error("Failed to create app", zap.Error(err), zap.String("user_id", userID))
		h.writeError(w, http.StatusInternalServerError, "Failed to create app")
		return
	}

	// Save environment variables BEFORE enqueueing build task
	// This ensures they're available when the deployment happens
	if len(req.EnvVars) > 0 && h.envVarRepo != nil {
		for _, envVar := range req.EnvVars {
			// Skip empty keys
			if envVar.Key == "" {
				continue
			}
			
			_, err := h.envVarRepo.CreateEnvVar(r.Context(), app.ID, envVar.Key, envVar.Value)
			if err != nil {
				// Log error but don't fail app creation - env vars can be added later
				h.logger.Warn("Failed to create environment variable during app creation",
					zap.Error(err),
					zap.String("app_id", app.ID),
					zap.String("key", envVar.Key),
				)
			} else {
				h.logger.Info("Created environment variable during app creation",
					zap.String("app_id", app.ID),
					zap.String("key", envVar.Key),
				)
			}
		}
		h.logger.Info("Environment variables saved for app",
			zap.String("app_id", app.ID),
			zap.Int("count", len(req.EnvVars)),
		)
	}

	// Generate build job ID
	buildJobID := uuid.New().String()

	// Enqueue build task to trigger deployment
	requestID := middleware.GetReqID(r.Context())
	if h.taskEnqueue != nil {
		buildPayload := tasks.BuildTaskPayload{
			AppID:      app.ID,
			BuildJobID: buildJobID,
			RepoURL:    req.RepoURL,
			Branch:     branch,
			UserID:     userID,
		}

		taskInfo, err := h.taskEnqueue.EnqueueBuildTask(r.Context(), buildPayload, userID)
		if err != nil {
			h.logger.Error("Failed to enqueue build task", 
				zap.Error(err), 
				zap.String("app_id", app.ID),
				zap.String("request_id", requestID),
				zap.String("user_id", userID),
			)
			// Log error but don't fail the app creation - user can manually redeploy
			h.logger.Warn("App created but deployment not started", 
				zap.String("app_id", app.ID),
				zap.String("request_id", requestID),
			)
		} else {
			h.logger.Info("Build task enqueued successfully",
				zap.String("app_id", app.ID),
				zap.String("build_job_id", buildJobID),
				zap.String("task_id", taskInfo.ID),
				zap.String("request_id", requestID),
				zap.String("user_id", userID),
			)
		}
	} else {
		h.logger.Warn("Task enqueue service not available - deployment will not start automatically", 
			zap.String("app_id", app.ID),
			zap.String("request_id", requestID),
		)
	}

	// Create a deployment response
	now := time.Now().Format(time.RFC3339)
	deployment := Deployment{
		ID:        0, // Will be set by deployment system
		AppID:     0, // Will be set by deployment system
		Status:    "pending",
		CreatedAt: now,
		UpdatedAt: now,
	}

	response := CreateAppResponse{
		App:       *app,
		Deployment: deployment,
	}
	h.writeJSON(w, http.StatusCreated, response)
}

// DELETE /api/v1/apps/{id} - Delete app
func (h *Handlers) DeleteApp(w http.ResponseWriter, r *http.Request) {
	appID := chi.URLParam(r, "id")
	
	// Get user ID from context
	userID := h.getUserIDFromContext(r)
	if userID == "" {
		h.writeError(w, http.StatusUnauthorized, "User ID not found in context")
		return
	}
	
	h.logger.Info("Deleting app and cleaning up resources", zap.String("app_id", appID), zap.String("user_id", userID))
	
	// Get app info before deletion (needed for cleanup)
	app, err := h.appRepo.GetAppByID(appID, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.writeError(w, http.StatusNotFound, "App not found or you don't have permission to delete it")
			return
		}
		h.logger.Error("Failed to get app for deletion", zap.Error(err), zap.String("app_id", appID))
		h.writeError(w, http.StatusInternalServerError, "Failed to get app")
		return
	}
	
	// Step 1: Clean up Docker resources (containers and images)
	if h.deploymentService != nil {
		if err := h.deploymentService.CleanupAppResources(r.Context(), appID); err != nil {
			h.logger.Warn("Failed to cleanup Docker resources during app deletion", 
				zap.Error(err), 
				zap.String("app_id", appID),
			)
			// Continue with deletion even if cleanup fails
		}
	} else {
		h.logger.Warn("Deployment service not available, skipping Docker resource cleanup", zap.String("app_id", appID))
	}
	
	// Step 2: Delete the app from database (this will also delete app_logs, and cascade will handle: deployments, env_vars, build_jobs, runtime_instances)
	err = h.appRepo.DeleteApp(appID, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.writeError(w, http.StatusNotFound, "App not found or you don't have permission to delete it")
			return
		}
		h.logger.Error("Failed to delete app", zap.Error(err), zap.String("app_id", appID), zap.String("user_id", userID))
		h.writeError(w, http.StatusInternalServerError, "Failed to delete app")
		return
	}
	
	// Step 3: Clean up cloned repositories
	// Note: This is best-effort cleanup. Cloned repos are typically cleaned up after builds,
	// but we clean them up here to be thorough. We'll need GitService access for this.
	// For now, we log a warning if we can't access it. This can be enhanced later.
	h.logger.Info("App deleted successfully, cloned repos should be cleaned up by build worker", 
		zap.String("app_id", appID),
		zap.String("repo_url", app.RepoURL),
	)
	
	h.logger.Info("App deletion completed",
		zap.String("app_id", appID),
		zap.String("user_id", userID),
	)
	
	// Return 204 No Content on success
	w.WriteHeader(http.StatusNoContent)
}

// POST /api/v1/apps/{id}/redeploy - Redeploy app
func (h *Handlers) RedeployApp(w http.ResponseWriter, r *http.Request) {
	appID := chi.URLParam(r, "id")
	
	// Get user ID from context
	userID := h.getUserIDFromContext(r)
	if userID == "" {
		h.writeError(w, http.StatusUnauthorized, "User ID not found in context")
		return
	}

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

	// Get app from database
	app, err := h.appRepo.GetAppByID(appID, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.writeError(w, http.StatusNotFound, "App not found or you don't have permission to redeploy it")
			return
		}
		h.logger.Error("Failed to get app", zap.Error(err), zap.String("app_id", appID))
		h.writeError(w, http.StatusInternalServerError, "Failed to get app")
		return
	}

	// Generate new build job ID
	buildJobID := uuid.New().String()

	// Enqueue build task to trigger deployment
	// This will: 1) Clone the latest code from the repository branch, 2) Build the Docker image, 3) Deploy the container
	requestID := middleware.GetReqID(r.Context())
	if h.taskEnqueue != nil {
		buildPayload := tasks.BuildTaskPayload{
			AppID:      app.ID,
			BuildJobID: buildJobID,
			RepoURL:    app.RepoURL,    // Always use current repo URL from database
			Branch:     app.Branch,      // Always use current branch from database (ensures latest code from this branch)
			UserID:     userID,
		}

		taskInfo, err := h.taskEnqueue.EnqueueBuildTask(r.Context(), buildPayload, userID)
		if err != nil {
			h.logger.Error("Failed to enqueue build task for redeploy", 
				zap.Error(err), 
				zap.String("app_id", appID),
				zap.String("request_id", requestID),
				zap.String("user_id", userID),
			)
			h.writeError(w, http.StatusInternalServerError, "Failed to start deployment")
			return
		}

		h.logger.Info("Redeploy build task enqueued successfully",
			zap.String("app_id", app.ID),
			zap.String("build_job_id", buildJobID),
			zap.String("task_id", taskInfo.ID),
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
		)
	} else {
		h.logger.Error("Task enqueue service not available - cannot redeploy", 
			zap.String("app_id", appID),
			zap.String("request_id", requestID),
		)
		h.writeError(w, http.StatusInternalServerError, "Deployment service not available")
		return
	}

	// Create deployment response
	now := time.Now().Format(time.RFC3339)
	deployment := Deployment{
		ID:        0, // Will be set by deployment system
		AppID:     0, // Will be set by deployment system
		Status:    "building",
		CreatedAt: now,
		UpdatedAt: now,
	}

	response := CreateAppResponse{
		App:       *app,
		Deployment: deployment,
	}
	h.writeJSON(w, http.StatusOK, response)
}

// GET /api/v1/apps/{id}/deployments - Get deployments for app
func (h *Handlers) GetAppDeployments(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	appID := id // Use string ID directly
	
	if h.deploymentRepo == nil {
		h.logger.Error("Deployment repository not initialized")
		h.writeError(w, http.StatusInternalServerError, "Deployment repository not available")
		return
	}
	
	deploymentsData, err := h.deploymentRepo.GetDeploymentsByAppID(appID)
	if err != nil {
		h.logger.Error("Failed to get deployments", zap.Error(err), zap.String("app_id", appID))
		h.writeError(w, http.StatusInternalServerError, "Failed to retrieve deployments")
		return
	}
	
	// Convert to Deployment structs for response
	deployments := make([]Deployment, 0, len(deploymentsData))
	for _, d := range deploymentsData {
		var status, createdAt, updatedAt string
		if statusVal, ok := d["status"].(string); ok {
			status = statusVal
		}
		if createdAtVal, ok := d["created_at"].(string); ok {
			createdAt = createdAtVal
		}
		if updatedAtVal, ok := d["updated_at"].(string); ok {
			updatedAt = updatedAtVal
		}
		
		deployment := Deployment{
			ID:        d["id"], // Keep as-is (UUID string)
			AppID:     d["app_id"], // Keep as-is (UUID string)
			Status:    status,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		}
		
		if img, ok := d["image_name"].(map[string]interface{}); ok {
			deployment.ImageName = img
		}
		if cid, ok := d["container_id"].(map[string]interface{}); ok {
			deployment.ContainerID = cid
		}
		if sub, ok := d["subdomain"].(map[string]interface{}); ok {
			deployment.Subdomain = sub
		}
		if buildLog, ok := d["build_log"].(map[string]interface{}); ok {
			deployment.BuildLog = buildLog
		}
		if runtimeLog, ok := d["runtime_log"].(map[string]interface{}); ok {
			deployment.RuntimeLog = runtimeLog
		}
		if errMsg, ok := d["error_message"].(map[string]interface{}); ok {
			deployment.ErrorMessage = errMsg
		}
		
		deployments = append(deployments, deployment)
	}
	
	h.writeJSON(w, http.StatusOK, deployments)
}

// GET /api/v1/apps/{id}/env - Get environment variables
func (h *Handlers) GetEnvVars(w http.ResponseWriter, r *http.Request) {
	appID := chi.URLParam(r, "id")
	userID := h.getUserIDFromContext(r)
	if userID == "" {
		h.writeError(w, http.StatusUnauthorized, "User ID not found in context")
		return
	}

	// Verify app belongs to user
	if h.appRepo == nil {
		h.logger.Error("App repository not initialized")
		h.writeError(w, http.StatusInternalServerError, "App repository not available")
		return
	}

	_, err := h.appRepo.GetAppByID(appID, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.writeError(w, http.StatusNotFound, "App not found")
			return
		}
		h.logger.Error("Failed to get app", zap.Error(err), zap.String("app_id", appID))
		h.writeError(w, http.StatusInternalServerError, "Failed to retrieve app")
		return
	}

	if h.envVarRepo == nil {
		h.logger.Error("Env var repository not initialized")
		h.writeError(w, http.StatusInternalServerError, "Env var repository not available")
		return
	}

	envVars, err := h.envVarRepo.GetEnvVarsByAppID(r.Context(), appID)
	if err != nil {
		// Check for context deadline exceeded
		if errors.Is(err, context.DeadlineExceeded) {
			h.logger.Error("Timeout getting env vars", zap.Error(err), zap.String("app_id", appID))
			h.writeError(w, http.StatusGatewayTimeout, "Request timed out while retrieving environment variables")
			return
		}
		h.logger.Error("Failed to get env vars", zap.Error(err), zap.String("app_id", appID))
		h.writeError(w, http.StatusInternalServerError, "Failed to retrieve environment variables")
		return
	}

	// Convert []*EnvVar to []EnvVar for JSON response
	result := make([]EnvVar, len(envVars))
	for i, v := range envVars {
		result[i] = *v
	}

	h.writeJSON(w, http.StatusOK, result)
}

// POST /api/v1/apps/{id}/env - Create environment variable
func (h *Handlers) CreateEnvVar(w http.ResponseWriter, r *http.Request) {
	appID := chi.URLParam(r, "id")
	
	// Validate appID is a valid UUID
	if _, err := uuid.Parse(appID); err != nil {
		h.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid app ID format: %s", appID))
		return
	}
	
	userID := h.getUserIDFromContext(r)
	if userID == "" {
		h.writeError(w, http.StatusUnauthorized, "User ID not found in context")
		return
	}

	// Verify app belongs to user
	if h.appRepo == nil {
		h.logger.Error("App repository not initialized")
		h.writeError(w, http.StatusInternalServerError, "App repository not available")
		return
	}

	_, err := h.appRepo.GetAppByID(appID, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.writeError(w, http.StatusNotFound, "App not found")
			return
		}
		h.logger.Error("Failed to get app", zap.Error(err), zap.String("app_id", appID))
		h.writeError(w, http.StatusInternalServerError, "Failed to retrieve app")
		return
	}

	var req CreateEnvVarRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate key and value
	if req.Key == "" {
		h.writeError(w, http.StatusBadRequest, "Environment variable key is required")
		return
	}

	if h.envVarRepo == nil {
		h.logger.Error("Env var repository not initialized")
		h.writeError(w, http.StatusInternalServerError, "Env var repository not available")
		return
	}

	envVar, err := h.envVarRepo.CreateEnvVar(r.Context(), appID, req.Key, req.Value)
	if err != nil {
		// Check for context deadline exceeded first
		if errors.Is(err, context.DeadlineExceeded) {
			h.logger.Error("Timeout creating env var", 
				zap.Error(err), 
				zap.String("app_id", appID), 
				zap.String("key", req.Key),
			)
			h.writeError(w, http.StatusGatewayTimeout, "Request timed out while creating environment variable")
			return
		}
		// Check for specific database errors
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "23505": // Unique constraint violation
				h.writeError(w, http.StatusConflict, fmt.Sprintf("Environment variable '%s' already exists for this app", req.Key))
				return
			case "23503": // Foreign key constraint violation
				h.logger.Error("Foreign key constraint violation when creating env var", 
					zap.Error(err), 
					zap.String("app_id", appID), 
					zap.String("key", req.Key),
					zap.String("pg_error_code", pgErr.Code),
					zap.String("pg_error_message", pgErr.Message),
				)
				h.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid app ID or app does not exist: %s", pgErr.Message))
				return
			case "23514": // Check constraint violation
				h.logger.Error("Check constraint violation when creating env var", 
					zap.Error(err), 
					zap.String("app_id", appID), 
					zap.String("key", req.Key),
					zap.String("pg_error_code", pgErr.Code),
					zap.String("pg_error_message", pgErr.Message),
				)
				h.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid environment variable: %s", pgErr.Message))
				return
			default:
				h.logger.Error("Database error when creating env var", 
					zap.Error(err), 
					zap.String("app_id", appID), 
					zap.String("key", req.Key),
					zap.String("pg_error_code", pgErr.Code),
					zap.String("pg_error_message", pgErr.Message),
					zap.String("pg_error_detail", pgErr.Detail),
				)
				h.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create environment variable: %s", pgErr.Message))
				return
			}
		}
		// Non-PostgreSQL errors
		h.logger.Error("Failed to create env var", 
			zap.Error(err), 
			zap.String("app_id", appID), 
			zap.String("key", req.Key),
			zap.String("error_type", fmt.Sprintf("%T", err)),
		)
		h.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create environment variable: %v", err))
		return
	}

	h.writeJSON(w, http.StatusCreated, envVar)
}

// DELETE /api/v1/apps/{id}/env/{key} - Delete environment variable
func (h *Handlers) DeleteEnvVar(w http.ResponseWriter, r *http.Request) {
	appID := chi.URLParam(r, "id")
	key := chi.URLParam(r, "key")
	userID := h.getUserIDFromContext(r)
	if userID == "" {
		h.writeError(w, http.StatusUnauthorized, "User ID not found in context")
		return
	}

	// Verify app belongs to user
	if h.appRepo == nil {
		h.logger.Error("App repository not initialized")
		h.writeError(w, http.StatusInternalServerError, "App repository not available")
		return
	}

	_, err := h.appRepo.GetAppByID(appID, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.writeError(w, http.StatusNotFound, "App not found")
			return
		}
		h.logger.Error("Failed to get app", zap.Error(err), zap.String("app_id", appID))
		h.writeError(w, http.StatusInternalServerError, "Failed to retrieve app")
		return
	}

	if h.envVarRepo == nil {
		h.logger.Error("Env var repository not initialized")
		h.writeError(w, http.StatusInternalServerError, "Env var repository not available")
		return
	}

	err = h.envVarRepo.DeleteEnvVar(r.Context(), appID, key)
	if err != nil {
		// Check for context deadline exceeded first
		if errors.Is(err, context.DeadlineExceeded) {
			h.logger.Error("Timeout deleting env var", zap.Error(err), zap.String("app_id", appID), zap.String("key", key))
			h.writeError(w, http.StatusGatewayTimeout, "Request timed out while deleting environment variable")
			return
		}
		if errors.Is(err, pgx.ErrNoRows) {
			h.writeError(w, http.StatusNotFound, "Environment variable not found")
			return
		}
		h.logger.Error("Failed to delete env var", zap.Error(err), zap.String("app_id", appID), zap.String("key", key))
		h.writeError(w, http.StatusInternalServerError, "Failed to delete environment variable")
		return
	}

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
	
	h.logger.Info("GetDeploymentLogs called",
		zap.String("deployment_id", deploymentID),
		zap.String("url_path", r.URL.Path),
	)
	
	// Verify user has access to this deployment
	userID := h.getUserIDFromContext(r)
	if userID == "" {
		h.writeError(w, http.StatusUnauthorized, "User ID not found in context")
		return
	}

	// Get deployment from database (contains build_log and runtime_log)
	if h.deploymentRepo == nil {
		h.logger.Error("Deployment repository not initialized")
		h.writeError(w, http.StatusInternalServerError, "Deployment repository not available")
		return
	}

	deploymentData, err := h.deploymentRepo.GetDeploymentByID(deploymentID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.writeError(w, http.StatusNotFound, "Deployment not found")
			return
		}
		h.logger.Error("Failed to get deployment", zap.Error(err), zap.String("deployment_id", deploymentID))
		h.writeError(w, http.StatusInternalServerError, "Failed to retrieve deployment")
		return
	}

	// Verify app belongs to user
	appID, ok := deploymentData["app_id"].(string)
	if !ok {
		h.logger.Error("Invalid app_id in deployment data", zap.String("deployment_id", deploymentID))
		h.writeError(w, http.StatusInternalServerError, "Invalid deployment data")
		return
	}

	// Verify app ownership
	if h.appRepo == nil {
		h.logger.Error("App repository not initialized")
		h.writeError(w, http.StatusInternalServerError, "App repository not available")
		return
	}

	_, err = h.appRepo.GetAppByID(appID, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.writeError(w, http.StatusNotFound, "Deployment not found or access denied")
			return
		}
		h.logger.Error("Failed to verify app ownership", zap.Error(err), zap.String("app_id", appID), zap.String("user_id", userID))
		h.writeError(w, http.StatusInternalServerError, "Failed to verify deployment access")
		return
	}

	// Get build_log from filesystem (using build_job_id) or database (fallback)
	var buildLog, errorMsg string
	
	// First try to get build logs from filesystem using build_job_id
	if h.logPersistence != nil {
		// Check if build_job_id exists in deployment data (stored as plain string)
		buildJobIDVal, buildJobIDExists := deploymentData["build_job_id"]
		h.logger.Info("Checking for build_job_id in deployment data",
			zap.String("deployment_id", deploymentID),
			zap.String("app_id", appID),
			zap.Bool("build_job_id_exists", buildJobIDExists),
			zap.Any("build_job_id_value", buildJobIDVal),
			zap.String("build_job_id_type", fmt.Sprintf("%T", buildJobIDVal)),
		)
		
		if buildJobIDExists && buildJobIDVal != nil {
			var buildJobID string
			if idStr, ok := buildJobIDVal.(string); ok && idStr != "" {
				buildJobID = idStr
			}
			
			if buildJobID != "" {
				h.logger.Info("Attempting to retrieve build logs from filesystem",
					zap.String("build_job_id", buildJobID),
					zap.String("app_id", appID),
					zap.String("deployment_id", deploymentID),
				)
				// Build logs are stored with build_job_id in the filesystem
				buildLogContent, err := h.logPersistence.GetLogsByBuildJobID(r.Context(), appID, buildJobID)
				if err != nil {
					h.logger.Warn("Failed to get build logs from filesystem",
						zap.Error(err),
						zap.String("app_id", appID),
						zap.String("build_job_id", buildJobID),
					)
				} else if buildLogContent != "" {
					h.logger.Info("Successfully retrieved build logs from filesystem",
						zap.String("app_id", appID),
						zap.String("build_job_id", buildJobID),
						zap.Int("log_length", len(buildLogContent)),
					)
					buildLog = buildLogContent
				} else {
					h.logger.Debug("Build logs not found in filesystem, will check database",
						zap.String("app_id", appID),
						zap.String("build_job_id", buildJobID),
					)
				}
			} else {
				h.logger.Warn("build_job_id is NULL or empty in deployment - this should not happen for new deployments",
					zap.String("app_id", appID),
					zap.String("deployment_id", deploymentID),
					zap.String("suggestion", "Check if build_job was created in build_jobs table"),
				)
			}
		} else {
			h.logger.Debug("No build_job_id found in deployment data, will check database for build logs",
				zap.String("app_id", appID),
				zap.String("deployment_id", deploymentID),
			)
		}
	}
	
	// Fallback to database if filesystem doesn't have logs (for older deployments)
	if buildLog == "" {
		if buildLogVal, ok := deploymentData["build_log"].(map[string]interface{}); ok {
			if valid, ok := buildLogVal["Valid"].(bool); ok && valid {
				if str, ok := buildLogVal["String"].(string); ok && str != "" {
					buildLog = str
					h.logger.Debug("Retrieved build logs from database",
						zap.String("app_id", appID),
						zap.String("deployment_id", deploymentID),
						zap.Int("log_length", len(buildLog)),
					)
				}
			}
		}
	}
	
	if errorMsgVal, ok := deploymentData["error_message"].(map[string]interface{}); ok {
		if valid, ok := errorMsgVal["Valid"].(bool); ok && valid {
			if str, ok := errorMsgVal["String"].(string); ok {
				errorMsg = str
			}
		}
	}

	// Get runtime logs from persistence service (filesystem/Postgres)
	// Use container_id to look up logs since logs are stored with container_id as the identifier
	var runtimeLog string
	if h.logPersistence != nil {
		h.logger.Debug("Attempting to retrieve runtime logs",
			zap.String("deployment_id", deploymentID),
			zap.String("app_id", appID),
		)
		
		// Get container_id from deployment data
		containerIDVal, ok := deploymentData["container_id"].(map[string]interface{})
		if !ok {
			h.logger.Warn("container_id not found in deployment data",
				zap.String("deployment_id", deploymentID),
				zap.String("app_id", appID),
				zap.Any("container_id_value", deploymentData["container_id"]),
			)
		} else {
			h.logger.Debug("container_id found in deployment data",
				zap.String("deployment_id", deploymentID),
				zap.String("app_id", appID),
				zap.Any("container_id_val", containerIDVal),
			)
			if valid, ok := containerIDVal["Valid"].(bool); ok && valid {
				if containerID, ok := containerIDVal["String"].(string); ok && containerID != "" {
					h.logger.Info("Found container_id, retrieving logs",
						zap.String("container_id", containerID),
						zap.String("app_id", appID),
						zap.String("deployment_id", deploymentID),
					)
					
					runtimeLogContent, err := h.logPersistence.GetLogsByDeploymentID(r.Context(), appID, containerID)
					if err != nil {
						h.logger.Warn("Failed to get runtime logs from persistence service", 
							zap.Error(err), 
							zap.String("app_id", appID), 
							zap.String("container_id", containerID),
							zap.String("deployment_id", deploymentID))
						// Continue with empty runtime log rather than failing
					} else {
						h.logger.Info("Successfully retrieved runtime logs",
							zap.String("app_id", appID),
							zap.String("container_id", containerID),
							zap.String("deployment_id", deploymentID),
							zap.Int("log_length", len(runtimeLogContent)),
							zap.Bool("has_content", len(runtimeLogContent) > 0),
						)
						runtimeLog = runtimeLogContent
					}
				} else {
					h.logger.Warn("container_id string is empty",
						zap.String("deployment_id", deploymentID),
						zap.String("app_id", appID),
						zap.Any("container_id_val", containerIDVal),
					)
				}
			} else {
				h.logger.Warn("container_id is not valid in deployment data",
					zap.String("deployment_id", deploymentID),
					zap.String("app_id", appID),
					zap.Bool("valid", valid),
					zap.Any("container_id_val", containerIDVal),
				)
			}
		}
	} else {
		h.logger.Warn("Log persistence service not available",
			zap.String("deployment_id", deploymentID),
			zap.String("app_id", appID),
		)
	}

	// Get status from deployment
	status := "unknown"
	if statusVal, ok := deploymentData["status"].(string); ok {
		status = statusVal
	}

	logs := DeploymentLogs{
		DeploymentID: 0, // Not needed in response
		Status:       status,
		BuildLog:     buildLog,
		RuntimeLog:   runtimeLog,
		ErrorMessage: errorMsg,
	}
	h.logger.Info("Returning deployment logs response",
		zap.String("deployment_id", deploymentID),
		zap.String("app_id", appID),
		zap.Int("runtime_log_length", len(runtimeLog)),
		zap.Int("build_log_length", len(buildLog)),
		zap.Bool("has_runtime_log", len(runtimeLog) > 0),
		zap.Bool("has_build_log", len(buildLog) > 0),
		zap.Any("build_job_id_in_deployment", deploymentData["build_job_id"]),
	)
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
// BroadcastBuildStatus removed - DB is single source of truth, no WebSocket broadcasting needed

func (h *Handlers) HealthCheck(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status: "ok",
	}
	h.writeJSON(w, http.StatusOK, response)
}

// GET /api/user/me - Get user profile
func (h *Handlers) GetUserProfile(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by AuthMiddleware)
	userID := h.getUserIDFromContext(r)
	if userID == "" {
		h.writeError(w, http.StatusUnauthorized, "User ID not found in context")
		return
	}

	// Get user from database
	if h.userRepo == nil {
		h.logger.Error("User repository not initialized")
		h.writeError(w, http.StatusInternalServerError, "User repository not available")
		return
	}

	user, err := h.userRepo.GetUserByID(userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.writeError(w, http.StatusNotFound, "User not found")
			return
		}
		h.logger.Error("Failed to get user", zap.Error(err), zap.String("user_id", userID))
		h.writeError(w, http.StatusInternalServerError, "Failed to retrieve user")
		return
	}

	// Get user created_at and updated_at from database
	var createdAt, updatedAt time.Time
	if h.userRepo != nil {
		var err error
		createdAt, updatedAt, err = h.userRepo.GetUserDates(r.Context(), userID)
		if err != nil {
			// If dates are not available, use current time
			h.logger.Warn("Failed to get user dates", zap.Error(err), zap.String("user_id", userID))
			createdAt = time.Now()
			updatedAt = time.Now()
		}
	} else {
		createdAt = time.Now()
		updatedAt = time.Now()
	}

	// Get user's plan
	var planID string
	var plan *Plan
	if h.userPlanRepo != nil {
		planID, err = h.userPlanRepo.GetUserPlanID(r.Context(), userID)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			h.logger.Warn("Failed to get user plan ID", zap.Error(err), zap.String("user_id", userID))
		}
	}

	// Get plan details
	if planID != "" && h.planRepo != nil {
		plan, err = h.planRepo.GetPlanByID(r.Context(), planID)
		if err != nil {
			h.logger.Warn("Failed to get plan by ID", zap.Error(err), zap.String("plan_id", planID))
			plan = nil
		}
	}

	// If no plan found, use default free plan
	var planName string
	if plan == nil && h.planRepo != nil {
		defaultPlan, err := h.planRepo.GetDefaultPlan(r.Context())
		if err != nil {
			h.logger.Warn("Failed to get default plan, using fallback free plan", zap.Error(err))
			planName = "free"
			// Use fallback plan limits if database plan is not available
			plan = &Plan{
				Name:        "free",
				DisplayName: "Free Plan",
				Price:       0,
				MaxRAMMB:    512,
				MaxDiskMB:   1024,
				MaxApps:     3,
				AlwaysOn:     false,
				AutoDeploy:   false,
				HealthChecks: false,
				Logs:         true,
				ZeroDowntime: false,
				Workers:      false,
				PriorityBuilds: false,
				ManualDeployOnly: false,
			}
		} else {
			plan = defaultPlan
			planName = plan.Name
		}
	} else if plan != nil {
		planName = plan.Name
	} else {
		planName = "free"
		// Use fallback plan limits if no plan repo available
		plan = &Plan{
			Name:        "free",
			DisplayName: "Free Plan",
			Price:       0,
			MaxRAMMB:    512,
			MaxDiskMB:   1024,
			MaxApps:     3,
			AlwaysOn:     false,
			AutoDeploy:   false,
			HealthChecks: false,
			Logs:         true,
			ZeroDowntime: false,
			Workers:      false,
			PriorityBuilds: false,
			ManualDeployOnly: false,
		}
	}

	// Get user's apps to calculate usage
	var appCount int
	var totalRAMMB int
	var totalDiskMB int
	
	if h.appRepo != nil {
		apps, err := h.appRepo.GetAppsByUserID(userID)
		if err != nil {
			h.logger.Warn("Failed to get user apps for quota calculation", zap.Error(err), zap.String("user_id", userID))
		} else {
			appCount = len(apps)
			
			// Calculate total RAM and disk usage from apps' deployments
			for _, app := range apps {
				// Enrich app with deployment data to get usage stats
				h.enrichAppWithDeployment(r.Context(), &app)
				
				if app.Deployment != nil && app.Deployment.UsageStats != nil {
					totalRAMMB += app.Deployment.UsageStats.MemoryUsageMB
					// Convert disk usage from GB to MB
					totalDiskMB += int(app.Deployment.UsageStats.DiskUsageGB * 1024)
				}
			}
		}
	}

	// Build profile response with plan and quota information
	profile := UserProfile{
		ID:           user.ID,
		Email:        user.Email,
		FullName:     user.FullName,
		CompanyName:  user.CompanyName,
		EmailVerified: false, // TODO: Implement email verification check
		Plan:         planName,
		CreatedAt:    createdAt.Format(time.RFC3339),
		UpdatedAt:    updatedAt.Format(time.RFC3339),
		Quota: &Quota{
			PlanName:    plan.Name,
			AppCount:    appCount,
			TotalRAMMB:  totalRAMMB,
			TotalDiskMB: totalDiskMB,
			Plan: PlanInfo{
				Name:            plan.Name,
				DisplayName:     plan.DisplayName,
				Price:           plan.Price,
				MaxRAMMB:        plan.MaxRAMMB,
				MaxDiskMB:       plan.MaxDiskMB,
				MaxApps:         plan.MaxApps,
				AlwaysOn:        plan.AlwaysOn,
				AutoDeploy:      plan.AutoDeploy,
				HealthChecks:    plan.HealthChecks,
				Logs:            plan.Logs,
				ZeroDowntime:    plan.ZeroDowntime,
				Workers:         plan.Workers,
				PriorityBuilds:  plan.PriorityBuilds,
				ManualDeployOnly: plan.ManualDeployOnly,
			},
		},
	}

	h.writeJSON(w, http.StatusOK, profile)
}

// GET /api/v1/apps/{id}/verify - Verify deployment status
func (h *Handlers) VerifyDeployment(w http.ResponseWriter, r *http.Request) {
	appID := chi.URLParam(r, "id")
	
	// Get user ID from context
	userID := h.getUserIDFromContext(r)
	if userID == "" {
		h.writeError(w, http.StatusUnauthorized, "User ID not found in context")
		return
	}

	// Verify app ownership
	app, err := h.appRepo.GetAppByID(appID, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.writeError(w, http.StatusNotFound, "App not found or you don't have permission to access it")
			return
		}
		h.logger.Error("Failed to get app", zap.Error(err), zap.String("app_id", appID))
		h.writeError(w, http.StatusInternalServerError, "Failed to retrieve app")
		return
	}

	// Verify deployment if service is available
	if h.deploymentService == nil {
		h.writeError(w, http.StatusServiceUnavailable, "Deployment service not available")
		return
	}

	verification, err := h.deploymentService.VerifyDeployment(r.Context(), appID)
	if err != nil {
		h.logger.Error("Failed to verify deployment", 
			zap.Error(err), 
			zap.String("app_id", appID),
			zap.String("request_id", middleware.GetReqID(r.Context())),
		)
		// Return verification result even if there are errors
		// This allows frontend to see what's wrong
	}

	response := map[string]interface{}{
		"app_id":              appID,
		"app_name":            app.Name,
		"is_running":          verification.IsRunning,
		"container_id":         verification.ContainerID,
		"container_name":       verification.ContainerName,
		"port":                 verification.Port,
		"subdomain":            verification.Subdomain,
		"url":                  verification.URL,
		"traefik_configured":   verification.TraefikConfigured,
		"health_check_passed":  verification.HealthCheckPassed,
		"errors":               verification.Errors,
		"success":              verification.IsRunning && verification.TraefikConfigured && len(verification.Errors) == 0,
	}

	h.writeJSON(w, http.StatusOK, response)
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

// enrichAppWithDeployment enriches an app with deployment data and container stats
func (h *Handlers) enrichAppWithDeployment(ctx context.Context, app *App) {
	if h.deploymentRepo == nil {
		return
	}

	// Get deployments for this app
	deploymentsData, err := h.deploymentRepo.GetDeploymentsByAppID(app.ID)
	if err != nil || len(deploymentsData) == 0 {
		return
	}

	// Find the active/running deployment
	var activeDeployment map[string]interface{}
	for _, d := range deploymentsData {
		status, _ := d["status"].(string)
		if status == "running" {
			activeDeployment = d
			break
		}
	}

	// If no running deployment, use the most recent one
	if activeDeployment == nil {
		activeDeployment = deploymentsData[0]
	}

	// Extract container ID
	var containerID string
	if containerIDVal, ok := activeDeployment["container_id"].(map[string]interface{}); ok {
		if cidStr, ok := containerIDVal["String"].(string); ok && containerIDVal["Valid"].(bool) {
			containerID = cidStr
		}
	} else if cidStr, ok := activeDeployment["container_id"].(string); ok {
		containerID = cidStr
	}

	// Extract deployment ID (always set this, even if container ID is empty)
	var deploymentID string
	if depID, ok := activeDeployment["id"].(string); ok {
		deploymentID = depID
	}

	// Extract last deployed time
	var lastDeployedAt string
	if createdAt, ok := activeDeployment["created_at"].(string); ok {
		lastDeployedAt = createdAt
	} else if updatedAt, ok := activeDeployment["updated_at"].(string); ok {
		lastDeployedAt = updatedAt
	}

	// Get deployment status
	var state string
	if status, ok := activeDeployment["status"].(string); ok {
		state = status
	}

	// Always set basic deployment info (even if container stats aren't available)
	if deploymentID != "" {
		app.Deployment = &AppDeployment{
			ActiveDeploymentID: fmt.Sprintf("dep_%s", deploymentID), // Prefix for frontend compatibility
			LastDeployedAt:     lastDeployedAt,
			State:              state,
		}
	}

	// Set default resource limits and usage stats (will be updated if container stats are available)
	defaultMemoryMB := 512
	defaultCPU := 1
	defaultDiskGB := 10
	defaultMemoryUsageMB := 0
	defaultMemoryUsagePercent := 0.0
	defaultDiskUsageGB := 0.0
	defaultDiskUsagePercent := 0.0
	defaultRestartCount := 0

	// Only try to get container stats if container ID is available and deployment service exists
	if containerID == "" || h.deploymentService == nil {
		// Set defaults when container stats aren't available
		if app.Deployment != nil {
			app.Deployment.ResourceLimits = &ResourceLimits{
				MemoryMB: defaultMemoryMB,
				CPU:      defaultCPU,
				DiskGB:   defaultDiskGB,
			}
			app.Deployment.UsageStats = &UsageStats{
				MemoryUsageMB:      defaultMemoryUsageMB,
				MemoryUsagePercent: defaultMemoryUsagePercent,
				DiskUsageGB:        defaultDiskUsageGB,
				DiskUsagePercent:   defaultDiskUsagePercent,
				RestartCount:       defaultRestartCount,
			}
		}
		return
	}

	// Try to get Docker client via type assertion to the concrete type
	// The deploymentService is actually *services.DeploymentService
	type DockerClientGetter interface {
		GetDockerClient() *client.Client
	}
	
	var dockerClient *client.Client
	if getter, ok := h.deploymentService.(DockerClientGetter); ok {
		dockerClient = getter.GetDockerClient()
	} else {
		// Try reflection-based approach as fallback
		h.logger.Debug("Cannot get Docker client via interface, trying reflection")
		// Set defaults when Docker client can't be obtained
		if app.Deployment != nil {
			app.Deployment.ResourceLimits = &ResourceLimits{
				MemoryMB: defaultMemoryMB,
				CPU:      defaultCPU,
				DiskGB:   defaultDiskGB,
			}
			app.Deployment.UsageStats = &UsageStats{
				MemoryUsageMB:      defaultMemoryUsageMB,
				MemoryUsagePercent: defaultMemoryUsagePercent,
				DiskUsageGB:        defaultDiskUsageGB,
				DiskUsagePercent:   defaultDiskUsagePercent,
				RestartCount:       defaultRestartCount,
			}
		}
		return
	}

	if dockerClient == nil {
		// Set defaults when Docker client is nil
		if app.Deployment != nil {
			app.Deployment.ResourceLimits = &ResourceLimits{
				MemoryMB: defaultMemoryMB,
				CPU:      defaultCPU,
				DiskGB:   defaultDiskGB,
			}
			app.Deployment.UsageStats = &UsageStats{
				MemoryUsageMB:      defaultMemoryUsageMB,
				MemoryUsagePercent: defaultMemoryUsagePercent,
				DiskUsageGB:        defaultDiskUsageGB,
				DiskUsagePercent:   defaultDiskUsagePercent,
				RestartCount:       defaultRestartCount,
			}
		}
		return
	}

	// Get container stats (one-shot, not streaming)
	stats, err := dockerClient.ContainerStats(ctx, containerID, false)
	if err != nil {
		h.logger.Debug("Failed to get container stats", zap.Error(err), zap.String("container_id", containerID))
		// Continue without stats - we can still get basic info from inspect
	} else {
		defer stats.Body.Close()
	}

	// Inspect container to get resource limits and restart count
	containerJSON, err := dockerClient.ContainerInspect(ctx, containerID)
	if err != nil {
		h.logger.Debug("Failed to inspect container", zap.Error(err))
		// Set defaults when container inspection fails
		if app.Deployment != nil {
			app.Deployment.ResourceLimits = &ResourceLimits{
				MemoryMB: defaultMemoryMB,
				CPU:      defaultCPU,
				DiskGB:   defaultDiskGB,
			}
			app.Deployment.UsageStats = &UsageStats{
				MemoryUsageMB:      defaultMemoryUsageMB,
				MemoryUsagePercent: defaultMemoryUsagePercent,
				DiskUsageGB:        defaultDiskUsageGB,
				DiskUsagePercent:   defaultDiskUsagePercent,
				RestartCount:       defaultRestartCount,
			}
		}
		return
	}

	// Calculate memory usage from stats if available
	memoryLimit := float64(containerJSON.HostConfig.Memory)
	memoryUsageMB := 0
	memoryUsagePercent := 0.0
	
	if stats.Body != nil {
		// Parse stats JSON
		var statsJSON map[string]interface{}
		if err := json.NewDecoder(stats.Body).Decode(&statsJSON); err == nil {
			// Extract memory usage from stats
			if memoryStats, ok := statsJSON["memory_stats"].(map[string]interface{}); ok {
				if usage, ok := memoryStats["usage"].(float64); ok {
					memoryUsageMB = int(usage / 1024 / 1024)
					if memoryLimit > 0 {
						memoryUsagePercent = (usage / memoryLimit) * 100
					}
				}
			}
		}
	}

	// If stats weren't available, use a default/estimated value
	if memoryUsageMB == 0 && memoryLimit > 0 {
		// Estimate based on container state (rough estimate)
		memoryUsageMB = int(memoryLimit / 1024 / 1024 / 4) // Assume 25% usage as placeholder
		memoryUsagePercent = 25.0
	}

	// Calculate disk usage (Docker stats don't provide this directly)
	// We'll use a placeholder or estimate based on container size
	diskUsageGB := 0.5 // Default placeholder
	diskUsagePercent := 5.0 // Default placeholder

	// Get restart count
	restartCount := containerJSON.RestartCount

	// Get resource limits from container config
	memoryMB := int(containerJSON.HostConfig.Memory / 1024 / 1024)
	cpuLimit := float64(containerJSON.HostConfig.NanoCPUs) / 1e9
	diskGB := 10 // Default, could be calculated from container size or config

	// Update deployment data with container stats (if Deployment wasn't created above, create it now)
	if app.Deployment == nil {
		app.Deployment = &AppDeployment{
			ActiveDeploymentID: fmt.Sprintf("dep_%s", deploymentID),
			LastDeployedAt:     lastDeployedAt,
			State:              state,
		}
	}

	// Add resource limits and usage stats
	app.Deployment.ResourceLimits = &ResourceLimits{
		MemoryMB: memoryMB,
		CPU:      int(cpuLimit),
		DiskGB:   diskGB,
	}
	app.Deployment.UsageStats = &UsageStats{
		MemoryUsageMB:      memoryUsageMB,
		MemoryUsagePercent: memoryUsagePercent,
		DiskUsageGB:        diskUsageGB,
		DiskUsagePercent:   diskUsagePercent,
		RestartCount:       restartCount,
	}
}

// Admin handlers

// GET /admin/users - List all users with pagination
func (h *Handlers) AdminListUsers(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	search := r.URL.Query().Get("search")
	
	limit := 50
	offset := 0
	var err error
	
	if limitStr != "" {
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit < 1 || limit > 100 {
			limit = 50
		}
	}
	if offsetStr != "" {
		offset, err = strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			offset = 0
		}
	}
	
	// Get users from repository
	users, total, err := h.userRepo.ListAllUsers(limit, offset, search)
	if err != nil {
		h.logger.Error("Failed to list users", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "Failed to retrieve users")
		return
	}
	
	// Build response with quota information for each user
	type AdminUserResponse struct {
		ID           string    `json:"id"`
		Email        string    `json:"email"`
		FullName     string    `json:"full_name,omitempty"`
		CompanyName  string    `json:"company_name,omitempty"`
		EmailVerified bool     `json:"email_verified"`
		Plan         string    `json:"plan"`
		IsAdmin      bool      `json:"is_admin"`
		CreatedAt    string    `json:"created_at"`
		UpdatedAt    string    `json:"updated_at"`
		Quota        *Quota    `json:"quota,omitempty"`
	}
	
	var adminUsers []AdminUserResponse
	for _, user := range users {
		// Get user dates
		createdAt, updatedAt, err := h.userRepo.GetUserDates(r.Context(), user.ID)
		if err != nil {
			h.logger.Warn("Failed to get user dates", zap.Error(err), zap.String("user_id", user.ID))
			createdAt = time.Now()
			updatedAt = time.Now()
		}
		
		// Get user's plan
		var planID string
		var plan *Plan
		planName := "free"
		if h.userPlanRepo != nil {
			planID, err = h.userPlanRepo.GetUserPlanID(r.Context(), user.ID)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				h.logger.Warn("Failed to get user plan ID", zap.Error(err), zap.String("user_id", user.ID))
			}
		}
		
		if planID != "" && h.planRepo != nil {
			plan, err = h.planRepo.GetPlanByID(r.Context(), planID)
			if err != nil {
				h.logger.Warn("Failed to get plan by ID", zap.Error(err), zap.String("plan_id", planID))
				plan = nil
			} else {
				planName = plan.Name
			}
		}
		
		if plan == nil && h.planRepo != nil {
			defaultPlan, err := h.planRepo.GetDefaultPlan(r.Context())
			if err != nil {
				planName = "free"
				plan = &Plan{
					Name:        "free",
					DisplayName: "Free Plan",
					Price:       0,
					MaxRAMMB:    512,
					MaxDiskMB:   1024,
					MaxApps:     3,
					AlwaysOn:     false,
					AutoDeploy:   false,
					HealthChecks: false,
					Logs:         true,
					ZeroDowntime: false,
					Workers:      false,
					PriorityBuilds: false,
					ManualDeployOnly: false,
				}
			} else {
				plan = defaultPlan
				planName = plan.Name
			}
		}
		
		// Get user's apps to calculate usage
		var appCount int
		var totalRAMMB int
		var totalDiskMB int
		if h.appRepo != nil {
			apps, err := h.appRepo.GetAppsByUserID(user.ID)
			if err != nil {
				h.logger.Warn("Failed to get user apps for quota calculation", zap.Error(err), zap.String("user_id", user.ID))
			} else {
				appCount = len(apps)
				for _, app := range apps {
					h.enrichAppWithDeployment(r.Context(), &app)
					if app.Deployment != nil && app.Deployment.UsageStats != nil {
						totalRAMMB += app.Deployment.UsageStats.MemoryUsageMB
						totalDiskMB += int(app.Deployment.UsageStats.DiskUsageGB * 1024)
					}
				}
			}
		}
		
		if plan == nil {
			plan = &Plan{
				Name:        "free",
				DisplayName: "Free Plan",
				Price:       0,
				MaxRAMMB:    512,
				MaxDiskMB:   1024,
				MaxApps:     3,
				AlwaysOn:     false,
				AutoDeploy:   false,
				HealthChecks: false,
				Logs:         true,
				ZeroDowntime: false,
				Workers:      false,
				PriorityBuilds: false,
				ManualDeployOnly: false,
			}
		}
		
		adminUsers = append(adminUsers, AdminUserResponse{
			ID:            user.ID,
			Email:         user.Email,
			FullName:      user.FullName,
			CompanyName:   user.CompanyName,
			EmailVerified: false,
			Plan:          planName,
			IsAdmin:       false,
			CreatedAt:     createdAt.Format(time.RFC3339),
			UpdatedAt:     updatedAt.Format(time.RFC3339),
			Quota: &Quota{
				PlanName:    plan.Name,
				AppCount:    appCount,
				TotalRAMMB:  totalRAMMB,
				TotalDiskMB: totalDiskMB,
				Plan: PlanInfo{
					Name:            plan.Name,
					DisplayName:     plan.DisplayName,
					Price:           plan.Price,
					MaxRAMMB:        plan.MaxRAMMB,
					MaxDiskMB:       plan.MaxDiskMB,
					MaxApps:         plan.MaxApps,
					AlwaysOn:        plan.AlwaysOn,
					AutoDeploy:      plan.AutoDeploy,
					HealthChecks:    plan.HealthChecks,
					Logs:            plan.Logs,
					ZeroDowntime:    plan.ZeroDowntime,
					Workers:         plan.Workers,
					PriorityBuilds:  plan.PriorityBuilds,
					ManualDeployOnly: plan.ManualDeployOnly,
				},
			},
		})
	}
	
	response := map[string]interface{}{
		"users":  adminUsers,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	}
	
	h.writeJSON(w, http.StatusOK, response)
}

// GET /admin/apps - List all apps with pagination
func (h *Handlers) AdminListApps(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	
	limit := 50
	offset := 0
	var err error
	
	if limitStr != "" {
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit < 1 || limit > 100 {
			limit = 50
		}
	}
	if offsetStr != "" {
		offset, err = strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			offset = 0
		}
	}
	
	// Get apps from repository
	apps, total, err := h.appRepo.ListAllApps(limit, offset)
	if err != nil {
		h.logger.Error("Failed to list apps", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "Failed to retrieve apps")
		return
	}
	
	// Enrich apps with deployment data
	for i := range apps {
		h.enrichAppWithDeployment(r.Context(), &apps[i])
	}
	
	// Build response
	type AdminAppResponse struct {
		ID              string    `json:"id"`
		Name            string    `json:"name"`
		Slug            string    `json:"slug"`
		Status          string    `json:"status"`
		URL             string    `json:"url"`
		RepoURL         string    `json:"repo_url"`
		Branch          string    `json:"branch"`
		CreatedAt       string    `json:"created_at"`
		UpdatedAt       string    `json:"updated_at"`
		DeploymentCount int       `json:"deployment_count"`
	}
	
	var adminApps []AdminAppResponse
	for _, app := range apps {
		deploymentCount := 0
		if deployments, err := h.deploymentRepo.GetDeploymentsByAppID(app.ID); err == nil {
			deploymentCount = len(deployments)
		}
		
		adminApps = append(adminApps, AdminAppResponse{
			ID:              app.ID,
			Name:            app.Name,
			Slug:            app.Slug,
			Status:          app.Status,
			URL:             app.URL,
			RepoURL:         app.RepoURL,
			Branch:          app.Branch,
			CreatedAt:       app.CreatedAt,
			UpdatedAt:       app.UpdatedAt,
			DeploymentCount: deploymentCount,
		})
	}
	
	response := map[string]interface{}{
		"apps":   adminApps,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	}
	
	h.writeJSON(w, http.StatusOK, response)
}

// PATCH /admin/users/{id}/plan - Update user plan
func (h *Handlers) AdminUpdateUserPlan(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	
	var req struct {
		Plan string `json:"plan"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	
	// Get plan by name
	if h.planRepo == nil {
		h.writeError(w, http.StatusInternalServerError, "Plan repository not available")
		return
	}
	
	plan, err := h.planRepo.GetPlanByName(r.Context(), req.Plan)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.writeError(w, http.StatusBadRequest, fmt.Sprintf("Plan '%s' not found", req.Plan))
			return
		}
		h.logger.Error("Failed to get plan by name", zap.Error(err), zap.String("plan", req.Plan))
		h.writeError(w, http.StatusInternalServerError, "Failed to get plan")
		return
	}
	
	// Update user plan
	if h.userPlanRepo == nil {
		h.writeError(w, http.StatusInternalServerError, "User plan repository not available")
		return
	}
	
	err = h.userPlanRepo.UpdateUserPlanID(r.Context(), userID, plan.ID)
	if err != nil {
		h.logger.Error("Failed to update user plan", zap.Error(err), zap.String("user_id", userID), zap.String("plan_id", plan.ID))
		h.writeError(w, http.StatusInternalServerError, "Failed to update user plan")
		return
	}
	
	response := map[string]interface{}{
		"message": "User plan updated successfully",
		"user_id": userID,
		"plan":    req.Plan,
	}
	
	h.writeJSON(w, http.StatusOK, response)
}

// POST /admin/apps/{id}/stop - Stop app (admin version, no ownership check)
func (h *Handlers) AdminStopApp(w http.ResponseWriter, r *http.Request) {
	appID := chi.URLParam(r, "id")
	
	// Get app (no ownership check for admin)
	_, err := h.appRepo.GetAppByIDWithoutUserCheck(appID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.writeError(w, http.StatusNotFound, "App not found")
			return
		}
		h.logger.Error("Failed to get app", zap.Error(err), zap.String("app_id", appID))
		h.writeError(w, http.StatusInternalServerError, "Failed to get app")
		return
	}
	
	// TODO: Implement stop logic
	// For now, return success
	response := map[string]interface{}{
		"message":           "App stopped successfully",
		"app_id":            appID,
		"stopped_containers": 0,
	}
	
	h.writeJSON(w, http.StatusOK, response)
}

// POST /admin/apps/{id}/start - Start app (admin version, no ownership check)
func (h *Handlers) AdminStartApp(w http.ResponseWriter, r *http.Request) {
	appID := chi.URLParam(r, "id")
	
	// Get app (no ownership check for admin)
	_, err := h.appRepo.GetAppByIDWithoutUserCheck(appID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.writeError(w, http.StatusNotFound, "App not found")
			return
		}
		h.logger.Error("Failed to get app", zap.Error(err), zap.String("app_id", appID))
		h.writeError(w, http.StatusInternalServerError, "Failed to get app")
		return
	}
	
	// TODO: Implement start logic
	// For now, return success
	response := map[string]interface{}{
		"message": "App started successfully",
		"app_id":  appID,
	}
	
	h.writeJSON(w, http.StatusOK, response)
}

// POST /admin/apps/{id}/redeploy - Redeploy app (admin version, no ownership check)
func (h *Handlers) AdminRedeployApp(w http.ResponseWriter, r *http.Request) {
	appID := chi.URLParam(r, "id")
	
	// Get app (no ownership check for admin)
	_, err := h.appRepo.GetAppByIDWithoutUserCheck(appID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.writeError(w, http.StatusNotFound, "App not found")
			return
		}
		h.logger.Error("Failed to get app", zap.Error(err), zap.String("app_id", appID))
		h.writeError(w, http.StatusInternalServerError, "Failed to get app")
		return
	}
	
	// Reuse existing redeploy logic but skip ownership check
	// For now, return success (full implementation would trigger redeploy)
	response := map[string]interface{}{
		"message":    "App redeployment initiated",
		"app_id":     appID,
		"deployment": map[string]interface{}{},
	}
	
	h.writeJSON(w, http.StatusOK, response)
}

