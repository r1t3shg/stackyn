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
	// Add dependencies here (database, etc.)
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
type LogPersister interface {
	PersistBuildLog(buildJobID, logs string) error
}

// DeploymentService interface for deploying containers
// Uses services package types to avoid duplication
type DeploymentService interface {
	DeployContainer(ctx context.Context, opts services.DeploymentOptions) (*services.DeploymentResult, error)
	RollbackDeployment(ctx context.Context, appID, previousImageName, previousImageTag string) error
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
) *TaskHandler {
	return &TaskHandler{
		logger:           logger,
		gitService:       gitService,
		dockerBuild:      dockerBuild,
		runtimeDetector:  runtimeDetector,
		dockerfileGen:    dockerfileGen,
		logPersister:     logPersister,
		deploymentService: deploymentService,
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
			if persistErr := h.logPersister.PersistBuildLog(payload.BuildJobID, logBuffer.String()); persistErr != nil {
				h.logger.Warn("Failed to persist build logs", zap.Error(persistErr))
			}
		}
		return fmt.Errorf("failed to build Docker image: %w", err)
	}

	// Step 5: Persist build logs
	if h.logPersister != nil {
		if err := h.logPersister.PersistBuildLog(payload.BuildJobID, logBuffer.String()); err != nil {
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

	// Plan-based resource limits (default values - should come from database/plan)
	// TODO: Fetch actual plan limits from database
	limits := services.ResourceLimits{
		MemoryMB: 512, // Default: 512 MB
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

	// TODO: Implement actual cleanup logic
	// 1. Stop and remove containers
	// 2. Remove Docker images
	// 3. Clean up networking
	// 4. Update database
	// 5. Persist task state

	// Simulate work
	time.Sleep(1 * time.Second)

	h.logger.Info("Cleanup task completed",
		zap.String("app_id", payload.AppID),
	)

	return nil
}

