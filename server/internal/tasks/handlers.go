package tasks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"
	"stackyn/server/internal/services"
)

// TaskHandler handles task processing
type TaskHandler struct {
	logger           *zap.Logger
	gitService       GitService
	dockerBuild      DockerBuildService
	runtimeDetector  RuntimeDetector
	dockerfileGen    DockerfileGenerator
	logPersister     LogPersister
	deploymentService DeploymentService
	cleanupService   CleanupService
	planEnforcement  PlanEnforcementService
	// Add dependencies here (database, etc.)
}

// PlanEnforcementService interface for plan enforcement
type PlanEnforcementService interface {
	CheckMaxRAM(ctx context.Context, userID string, requestedRAMMB int) error
	CheckMaxConcurrentBuilds(ctx context.Context, userID string) error
	GetQueuePriority(ctx context.Context, userID string) (int, error)
	IncrementBuildCount(ctx context.Context, userID string) error
	DecrementBuildCount(ctx context.Context, userID string) error
	IncrementRAMUsage(ctx context.Context, userID string, ramMB int) error
	DecrementRAMUsage(ctx context.Context, userID string, ramMB int) error
}

// DockerBuildService interface for building Docker images
// Uses services package types to avoid duplication
type DockerBuildService interface {
	BuildImage(ctx context.Context, opts services.BuildOptions, logWriter io.Writer) (*services.BuildResult, error)
	Close() error
}

// RuntimeDetector interface for detecting application runtime
type RuntimeDetector interface {
	DetectRuntime(repoPath string) (services.Runtime, error)
}

// DockerfileGenerator interface for generating Dockerfiles
type DockerfileGenerator interface {
	GenerateDockerfile(repoPath string, runtime services.Runtime) error
}

// LogPersister interface for persisting build logs
// Uses services package types to avoid duplication
type LogPersister interface {
	PersistLog(ctx context.Context, entry services.LogEntry) error
	PersistLogStream(ctx context.Context, entry interface{}, reader io.Reader) error
}

// DeploymentService interface for deploying containers
// Uses services package types to avoid duplication
type DeploymentService interface {
	DeployContainer(ctx context.Context, opts services.DeploymentOptions) (*services.DeploymentResult, error)
	RollbackDeployment(ctx context.Context, appID, previousImageName, previousImageTag string) error
	Close() error
}

// CleanupService interface for cleanup operations
type CleanupService interface {
	RunCleanup(ctx context.Context) (*services.CleanupResult, error)
	Close() error
}

// GitService interface for repository operations
// Uses services package types to avoid duplication
type GitService interface {
	ValidatePublicRepo(ctx context.Context, repoURL string) error
	Clone(ctx context.Context, opts services.CloneOptions) (*services.CloneResult, error)
	Cleanup(clonePath string) error
}

// NewTaskHandler creates a new task handler
func NewTaskHandler(
	logger *zap.Logger,
	gitService GitService,
	dockerBuild DockerBuildService,
	runtimeDetector RuntimeDetector,
	dockerfileGen DockerfileGenerator,
	logPersister LogPersister,
	deploymentService DeploymentService,
	cleanupService CleanupService,
	planEnforcement PlanEnforcementService,
) *TaskHandler {
	return &TaskHandler{
		logger:           logger,
		gitService:       gitService,
		dockerBuild:      dockerBuild,
		runtimeDetector:  runtimeDetector,
		dockerfileGen:    dockerfileGen,
		logPersister:     logPersister,
		deploymentService: deploymentService,
		cleanupService:   cleanupService,
		planEnforcement:  planEnforcement,
	}
}

// HandleBuildTask processes build tasks
func (h *TaskHandler) HandleBuildTask(ctx context.Context, t *asynq.Task) error {
	var payload BuildTaskPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal build task payload: %w", err)
	}

	h.logger.Info("Processing build task",
		zap.String("app_id", payload.AppID),
		zap.String("build_job_id", payload.BuildJobID),
		zap.String("repo_url", payload.RepoURL),
		zap.String("branch", payload.Branch),
	)

	// Step 1: Clone repository with shallow clone
	if h.gitService == nil {
		return fmt.Errorf("git service not configured")
	}

	cloneOpts := services.CloneOptions{
		RepoURL: payload.RepoURL,
		Branch:  payload.Branch,
		Shallow: true, // Always use shallow clone
		Depth:   1,    // Only clone the latest commit
	}

	cloneResult, err := h.gitService.Clone(ctx, cloneOpts)
	if err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	// Ensure cleanup happens even if build fails
	defer func() {
		if cleanupErr := h.gitService.Cleanup(cloneResult.Path); cleanupErr != nil {
			h.logger.Warn("Failed to cleanup clone directory", zap.Error(cleanupErr))
		}
	}()

	h.logger.Info("Repository cloned",
		zap.String("path", cloneResult.Path),
		zap.String("commit_sha", cloneResult.CommitSHA),
	)

	// Step 2: Detect runtime
	if h.runtimeDetector == nil {
		return fmt.Errorf("runtime detector not configured")
	}

	runtime, err := h.runtimeDetector.DetectRuntime(cloneResult.Path)
	if err != nil {
		return fmt.Errorf("failed to detect runtime: %w", err)
	}

	if runtime == services.RuntimeUnknown {
		return fmt.Errorf("could not detect application runtime")
	}

	h.logger.Info("Runtime detected",
		zap.String("runtime", string(runtime)),
	)

	// Step 3: Generate Dockerfile if missing
	if h.dockerfileGen == nil {
		return fmt.Errorf("dockerfile generator not configured")
	}

	if err := h.dockerfileGen.GenerateDockerfile(cloneResult.Path, services.Runtime(runtime)); err != nil {
		return fmt.Errorf("failed to generate Dockerfile: %w", err)
	}

	// Step 4: Build Docker image with resource constraints
	if h.dockerBuild == nil {
		return fmt.Errorf("docker build service not configured")
	}

	// Create log buffer for streaming and persistence
	var logBuffer bytes.Buffer
	logWriter := io.MultiWriter(&logBuffer, os.Stdout) // Stream to both buffer and stdout

	// Generate image name
	imageName := fmt.Sprintf("stackyn-%s", payload.AppID)
	imageTag := payload.BuildJobID

	buildOpts := services.BuildOptions{
		ContextPath: cloneResult.Path,
		ImageName:   imageName,
		Tag:         imageTag,
	}

	buildResult, err := h.dockerBuild.BuildImage(ctx, buildOpts, logWriter)
	if err != nil {
		// Persist logs even on failure
		if h.logPersister != nil {
			logEntry := services.LogEntry{
				AppID:      payload.AppID,
				BuildJobID: payload.BuildJobID,
				LogType:    string(services.LogTypeBuild),
				Timestamp:  time.Now(),
				Content:    logBuffer.String(),
				Size:       int64(logBuffer.Len()),
			}
			if persistErr := h.logPersister.PersistLog(ctx, logEntry); persistErr != nil {
				h.logger.Warn("Failed to persist build logs", zap.Error(persistErr))
			}
		}

		// Trigger cleanup on build failure
		// This ensures build artifacts and temp files are cleaned up
		if h.cleanupService != nil {
			h.logger.Info("Triggering cleanup after build failure")
			if _, cleanupErr := h.cleanupService.RunCleanup(ctx); cleanupErr != nil {
				h.logger.Warn("Cleanup after build failure failed", zap.Error(cleanupErr))
			}
		}

		return fmt.Errorf("failed to build Docker image: %w", err)
	}

	// Step 5: Persist build logs
	if h.logPersister != nil {
		logEntry := services.LogEntry{
			AppID:      payload.AppID,
			BuildJobID: payload.BuildJobID,
			LogType:    string(services.LogTypeBuild),
			Timestamp:  time.Now(),
			Content:    logBuffer.String(),
			Size:       int64(logBuffer.Len()),
		}
		if err := h.logPersister.PersistLog(ctx, logEntry); err != nil {
			h.logger.Warn("Failed to persist build logs", zap.Error(err))
		}
	}

	h.logger.Info("Build task completed",
		zap.String("app_id", payload.AppID),
		zap.String("build_job_id", payload.BuildJobID),
		zap.String("commit_sha", cloneResult.CommitSHA),
		zap.String("image_id", buildResult.ImageID),
		zap.String("image_name", buildResult.ImageName),
	)

	// TODO: Step 6: Push to registry
	// TODO: Step 7: Update build job status in database

	return nil
}

// HandleDeployTask processes deploy tasks
func (h *TaskHandler) HandleDeployTask(ctx context.Context, t *asynq.Task) error {
	var payload DeployTaskPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal deploy task payload: %w", err)
	}

	h.logger.Info("Processing deploy task",
		zap.String("app_id", payload.AppID),
		zap.String("deployment_id", payload.DeploymentID),
		zap.String("image_name", payload.ImageName),
		zap.String("build_job_id", payload.BuildJobID),
	)

	// Extract user ID from payload
	userID := payload.UserID
	if userID == "" {
		// Fallback to app ID if user ID not provided (for backward compatibility)
		userID = payload.AppID
		h.logger.Warn("UserID not provided in deploy task payload, using app ID as fallback",
			zap.String("app_id", payload.AppID),
		)
	}

	if h.deploymentService == nil {
		return fmt.Errorf("deployment service not configured")
	}

	// Extract image name and tag from payload
	imageName := payload.ImageName
	if imageName == "" {
		// Fallback: construct from app ID
		imageName = fmt.Sprintf("stackyn-%s", payload.AppID)
	}
	imageTag := payload.BuildJobID
	if imageTag == "" {
		imageTag = "latest"
	}

	// Generate subdomain if not provided
	subdomain := payload.Subdomain
	if subdomain == "" {
		subdomain = fmt.Sprintf("%s.stackyn.local", payload.AppID)
	}

	// Default port (can be overridden via env vars)
	port := 8080

	// Plan-based resource limits
	// Default values, will be overridden by plan limits if plan enforcement is enabled
	memoryMB := 512 // Default: 512 MB
	if payload.RequestedRAMMB > 0 {
		memoryMB = payload.RequestedRAMMB
	}

	// Check RAM limit if plan enforcement is enabled
	if h.planEnforcement != nil {
		if err := h.planEnforcement.CheckMaxRAM(ctx, userID, memoryMB); err != nil {
			return fmt.Errorf("plan limit exceeded: %w", err)
		}

		// Increment RAM usage
		if err := h.planEnforcement.IncrementRAMUsage(ctx, userID, memoryMB); err != nil {
			h.logger.Warn("Failed to increment RAM usage", zap.Error(err))
		}

		// Decrement RAM usage on exit (if deployment fails)
		defer func() {
			// Only decrement if deployment failed
			// Successful deployments should keep RAM usage tracked
		}()
	}

	limits := services.ResourceLimits{
		MemoryMB: int64(memoryMB),
		CPU:      0.5, // Default: 0.5 CPU
	}

	// Prepare deployment options
	deployOpts := services.DeploymentOptions{
		AppID:        payload.AppID,
		DeploymentID: payload.DeploymentID,
		ImageName:    imageName,
		ImageTag:     imageTag,
		Subdomain:    subdomain,
		Port:         port,
		Limits:       limits,
		EnvVars:      make(map[string]string), // Can be extended with additional env vars
	}

	// Deploy container
	deployResult, err := h.deploymentService.DeployContainer(ctx, deployOpts)
	if err != nil {
		// On deployment failure, attempt rollback if previous deployment exists
		h.logger.Error("Deployment failed, attempting rollback",
			zap.String("app_id", payload.AppID),
			zap.Error(err),
		)

		// TODO: Get previous image from database
		// For now, we'll just log the error
		// rollbackErr := h.deploymentService.RollbackDeployment(ctx, payload.AppID, previousImageName, previousImageTag)
		
		return fmt.Errorf("failed to deploy container: %w", err)
	}

	h.logger.Info("Deploy task completed",
		zap.String("app_id", payload.AppID),
		zap.String("deployment_id", payload.DeploymentID),
		zap.String("container_id", deployResult.ContainerID),
		zap.String("container_name", deployResult.ContainerName),
		zap.String("status", deployResult.Status),
	)

	// TODO: Update deployment status in database
	// TODO: Persist task state

	return nil
}

// HandleCleanupTask processes cleanup tasks
func (h *TaskHandler) HandleCleanupTask(ctx context.Context, t *asynq.Task) error {
	var payload CleanupTaskPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal cleanup task payload: %w", err)
	}

	h.logger.Info("Processing cleanup task",
		zap.String("app_id", payload.AppID),
		zap.Strings("container_ids", payload.ContainerIDs),
		zap.Strings("image_names", payload.ImageNames),
	)

	if h.cleanupService == nil {
		return fmt.Errorf("cleanup service not configured")
	}

	// Run cleanup operation
	result, err := h.cleanupService.RunCleanup(ctx)
	if err != nil {
		return fmt.Errorf("cleanup operation failed: %w", err)
	}

	// Log results
	h.logger.Info("Cleanup task completed",
		zap.String("app_id", payload.AppID),
		zap.Int("containers_removed", result.ContainersRemoved),
		zap.Int("images_removed", result.ImagesRemoved),
		zap.Int64("space_freed_mb", result.SpaceFreedMB),
		zap.Int("temp_dirs_pruned", result.TempDirsPruned),
		zap.Int("errors", len(result.Errors)),
	)

	// Log any errors that occurred
	if len(result.Errors) > 0 {
		for _, errMsg := range result.Errors {
			h.logger.Warn("Cleanup error", zap.String("error", errMsg))
		}
	}

	return nil
}

