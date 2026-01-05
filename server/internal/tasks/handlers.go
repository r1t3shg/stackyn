package tasks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/docker/docker/client"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"go.uber.org/zap"
	stackynerrors "stackyn/server/internal/errors"
	"stackyn/server/internal/services"
)

// TaskEnqueueService interface for enqueueing tasks
type TaskEnqueueService interface {
	EnqueueDeployTask(ctx context.Context, payload interface{}, userID string) (*asynq.TaskInfo, error)
}

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
	constraintsService ConstraintsService
	taskEnqueue      TaskEnqueueService
	wsBroadcast      *services.WebSocketBroadcastClient
	deploymentRepo   DeploymentRepository // For storing deployment status in DB
	appRepo          AppRepository        // For updating app status and URL
}

// ConstraintsService interface for constraint enforcement
type ConstraintsService interface {
	ValidateAllConstraints(ctx context.Context, repoURL, repoPath string) error
	ValidateBuildTime(ctx context.Context, buildTimeMinutes int) error
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
	DeployWithDockerCompose(ctx context.Context, opts services.DeploymentOptions) (*services.DeploymentResult, error)
	RollbackDeployment(ctx context.Context, appID, previousImageName, previousImageTag string) error
	GetDockerClient() *client.Client
	Close() error
}

// DeploymentRepository interface for deployment database operations
type DeploymentRepository interface {
	CreateDeployment(appID, buildJobID, status, imageName, containerID, subdomain string) (string, error)
	UpdateDeployment(deploymentID, status, imageName, containerID, subdomain, errorMsg string) error
}

// AppRepository interface for app database operations
type AppRepository interface {
	UpdateApp(appID, status, url string) error
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
	constraintsService ConstraintsService,
	taskEnqueue TaskEnqueueService,
	wsBroadcast *services.WebSocketBroadcastClient, // Deprecated - not used, DB is single source of truth
	deploymentRepo DeploymentRepository, // For storing deployment status in DB
	appRepo AppRepository, // For updating app status and URL
) *TaskHandler {
	return &TaskHandler{
		logger:           logger,
		gitService:       gitService,
		dockerBuild:      dockerBuild,
		runtimeDetector:  runtimeDetector,
		dockerfileGen:    dockerfileGen,
		logPersister:     logPersister,
		deploymentService: deploymentService,
		deploymentRepo:   deploymentRepo,
		cleanupService:   cleanupService,
		planEnforcement:  planEnforcement,
		constraintsService: constraintsService,
		taskEnqueue:      taskEnqueue,
		wsBroadcast:      nil, // Not used - DB is single source of truth
		appRepo:          appRepo,
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

	// Update app status to "building" when build starts
	if h.appRepo != nil {
		if err := h.appRepo.UpdateApp(payload.AppID, "building", ""); err != nil {
			h.logger.Warn("Failed to update app status to building",
				zap.Error(err),
				zap.String("app_id", payload.AppID),
			)
		}
	}

	// Step 1: Clone repository with shallow clone
	if h.gitService == nil {
		return fmt.Errorf("git service not configured")
	}

	cloneOpts := services.CloneOptions{
		RepoURL:  payload.RepoURL,
		Branch:   payload.Branch,
		Shallow:  true,              // Always use shallow clone
		Depth:    1,                // Only clone the latest commit
		UniqueID: payload.BuildJobID, // Use build job ID to avoid concurrent clone conflicts
	}

	cloneResult, err := h.gitService.Clone(ctx, cloneOpts)
	if err != nil {
		// Check if it's a StackynError and log it properly
		var errorMsg string
		if stackynErr, ok := stackynerrors.AsStackynError(err); ok {
			h.logger.Error("Repository clone failed",
				zap.String("app_id", payload.AppID),
				zap.String("build_job_id", payload.BuildJobID),
				zap.String("error_code", string(stackynErr.Code)),
				zap.String("error_message", stackynErr.Message),
				zap.String("error_details", stackynErr.Details),
				zap.Error(stackynErr.Err),
			)
			errorMsg = stackynErr.Message
			if stackynErr.Details != "" {
				errorMsg = fmt.Sprintf("%s: %s", stackynErr.Message, stackynErr.Details)
			}
		} else {
			errorMsg = fmt.Sprintf("Failed to clone repository: %v", err)
			h.logger.Error("Repository clone failed",
				zap.String("app_id", payload.AppID),
				zap.String("build_job_id", payload.BuildJobID),
				zap.Error(err),
			)
		}
		
		// Update app status to "failed" when clone fails
		if h.appRepo != nil {
			if updateErr := h.appRepo.UpdateApp(payload.AppID, "failed", ""); updateErr != nil {
				h.logger.Warn("Failed to update app status to failed",
					zap.Error(updateErr),
					zap.String("app_id", payload.AppID),
				)
			}
		}

		// Create a failed deployment with error message
		// Check if app still exists (it might have been deleted)
		if h.deploymentRepo != nil {
			h.logger.Info("Creating failed deployment for clone error",
				zap.String("app_id", payload.AppID),
				zap.String("build_job_id", payload.BuildJobID),
			)
			deploymentID, createErr := h.deploymentRepo.CreateDeployment(
				payload.AppID,
				payload.BuildJobID,
				"failed",
				"",
				"",
				"",
			)
			if createErr == nil && deploymentID != "" {
				// Update the deployment with error message
				if updateErr := h.deploymentRepo.UpdateDeployment(deploymentID, "", "", "", "", errorMsg); updateErr != nil {
					h.logger.Warn("Failed to update deployment error message",
						zap.Error(updateErr),
						zap.String("app_id", payload.AppID),
						zap.String("deployment_id", deploymentID),
					)
				} else {
					h.logger.Info("Created failed deployment with error message",
						zap.String("app_id", payload.AppID),
						zap.String("deployment_id", deploymentID),
						zap.String("error", errorMsg),
					)
				}
			} else {
				// Check if error is due to app being deleted (foreign key constraint)
				if createErr != nil && strings.Contains(createErr.Error(), "foreign key constraint") {
					h.logger.Info("App was deleted, skipping deployment creation",
						zap.String("app_id", payload.AppID),
						zap.String("build_job_id", payload.BuildJobID),
					)
				} else if createErr != nil {
					h.logger.Warn("Failed to create failed deployment",
						zap.Error(createErr),
						zap.String("app_id", payload.AppID),
						zap.String("build_job_id", payload.BuildJobID),
					)
				}
			}
		} else {
			h.logger.Warn("Deployment repository not available - cannot create failed deployment",
				zap.String("app_id", payload.AppID),
			)
		}
		
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	h.logger.Info("Repository cloned",
		zap.String("path", cloneResult.Path),
		zap.String("commit_sha", cloneResult.CommitSHA),
	)

	// Repository cloned - status will be stored in DB

	// MVP constraints validation removed - allowing all repository types

	// Check for docker-compose.yml file (must be before defer to be in scope)
	hasDockerCompose := h.hasDockerComposeFile(cloneResult.Path)
	h.logger.Info("Docker Compose detection",
		zap.String("app_id", payload.AppID),
		zap.String("build_job_id", payload.BuildJobID),
		zap.Bool("has_docker_compose", hasDockerCompose),
		zap.String("repo_path", cloneResult.Path),
	)

	// Ensure cleanup happens even if build fails
	// BUT: Skip cleanup if docker-compose is detected (deploy task needs the repo path)
	defer func() {
		if !hasDockerCompose {
			// Only cleanup if docker-compose is not being used
			// For docker-compose deployments, cleanup will happen after deployment completes
			if cleanupErr := h.gitService.Cleanup(cloneResult.Path); cleanupErr != nil {
				h.logger.Warn("Failed to cleanup clone directory", zap.Error(cleanupErr))
			}
		} else {
			h.logger.Info("Skipping repo cleanup - docker-compose deployment will use this path",
				zap.String("app_id", payload.AppID),
				zap.String("repo_path", cloneResult.Path),
			)
		}
	}()

	// Step 2: Detect runtime
	if h.runtimeDetector == nil {
		return fmt.Errorf("runtime detector not configured")
	}

	runtime, err := h.runtimeDetector.DetectRuntime(cloneResult.Path)
	if err != nil {
		h.logger.Error("Runtime detection failed",
			zap.String("app_id", payload.AppID),
			zap.String("build_job_id", payload.BuildJobID),
			zap.Error(err),
		)
		return stackynerrors.Wrap(stackynerrors.ErrorCodeRuntimeNotDetected, err, "Failed to detect runtime")
	}

	if runtime == services.RuntimeUnknown {
		h.logger.Error("Runtime not detected",
			zap.String("app_id", payload.AppID),
			zap.String("build_job_id", payload.BuildJobID),
		)
		return stackynerrors.New(stackynerrors.ErrorCodeRuntimeNotDetected, "Could not detect a supported runtime")
	}
	
	// Check for unsupported runtimes (if any)
	// This would be handled by the runtime detector, but we can add explicit checks here
	if runtime != services.RuntimeNodeJS && runtime != services.RuntimePython && 
		runtime != services.RuntimeGo && runtime != services.RuntimeJava {
		h.logger.Error("Unsupported runtime detected",
			zap.String("app_id", payload.AppID),
			zap.String("build_job_id", payload.BuildJobID),
			zap.String("runtime", string(runtime)),
		)
		return stackynerrors.New(stackynerrors.ErrorCodeUnsupportedLanguage, fmt.Sprintf("Runtime '%s' is not supported", runtime))
	}

	h.logger.Info("Runtime detected",
		zap.String("runtime", string(runtime)),
	)

	// Runtime detected - status will be stored in DB

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

	// Building Docker image - status will be stored in DB

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

		// Extract meaningful error message from build logs
		errorMsg := h.extractBuildError(logBuffer.String(), err)
		
		// Determine error code based on error type
		var errorCode stackynerrors.ErrorCode = stackynerrors.ErrorCodeBuildFailed
		if strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "deadline exceeded") {
			errorCode = stackynerrors.ErrorCodeBuildTimeout
		}
		
		stackynErr := stackynerrors.New(errorCode, errorMsg)
		h.logger.Error("Build failed",
			zap.String("app_id", payload.AppID),
			zap.String("build_job_id", payload.BuildJobID),
			zap.String("error_code", string(errorCode)),
			zap.String("error_message", errorMsg),
			zap.Error(err),
		)
		
		// Use errorMsg variable for deployment record
		errorMsgForDB := fmt.Sprintf("[%s] %s", string(errorCode), errorMsg)

		// Update app status to "failed" when build fails
		if h.appRepo != nil {
			if updateErr := h.appRepo.UpdateApp(payload.AppID, "failed", ""); updateErr != nil {
				h.logger.Warn("Failed to update app status to failed",
					zap.Error(updateErr),
					zap.String("app_id", payload.AppID),
				)
			}
		} else {
			h.logger.Warn("App repository not available - cannot update app status",
				zap.String("app_id", payload.AppID),
			)
		}

		// Create a failed deployment with error message
		// Note: App might have been deleted, so we handle foreign key errors gracefully
		if h.deploymentRepo != nil {
			h.logger.Info("Creating failed deployment",
				zap.String("app_id", payload.AppID),
				zap.String("build_job_id", payload.BuildJobID),
			)
			deploymentID, createErr := h.deploymentRepo.CreateDeployment(
				payload.AppID,
				payload.BuildJobID,
				"failed",
				"",
				"",
				"",
			)
			if createErr == nil && deploymentID != "" {
				h.logger.Info("Deployment created, updating with error message",
					zap.String("app_id", payload.AppID),
					zap.String("deployment_id", deploymentID),
				)
				// Update the deployment with error message (include error code)
				if updateErr := h.deploymentRepo.UpdateDeployment(deploymentID, "", "", "", "", errorMsgForDB); updateErr != nil {
					h.logger.Warn("Failed to update deployment error message",
						zap.Error(updateErr),
						zap.String("app_id", payload.AppID),
						zap.String("deployment_id", deploymentID),
					)
				} else {
					h.logger.Info("Created failed deployment with error message",
						zap.String("app_id", payload.AppID),
						zap.String("deployment_id", deploymentID),
						zap.String("error", errorMsg),
					)
				}
			} else {
				// Check if error is due to app being deleted (foreign key constraint)
				if createErr != nil && strings.Contains(createErr.Error(), "foreign key constraint") {
					h.logger.Info("App was deleted, skipping deployment creation",
						zap.String("app_id", payload.AppID),
						zap.String("build_job_id", payload.BuildJobID),
					)
				} else {
					h.logger.Warn("Failed to create failed deployment",
						zap.Error(createErr),
						zap.String("app_id", payload.AppID),
						zap.String("deployment_id", deploymentID),
					)
				}
			}
		} else {
			h.logger.Warn("Deployment repository not available - cannot create failed deployment",
				zap.String("app_id", payload.AppID),
			)
		}

		// Trigger cleanup on build failure
		// This ensures build artifacts and temp files are cleaned up
		if h.cleanupService != nil {
			h.logger.Info("Triggering cleanup after build failure")
			if _, cleanupErr := h.cleanupService.RunCleanup(ctx); cleanupErr != nil {
				h.logger.Warn("Cleanup after build failure failed", zap.Error(cleanupErr))
			}
		}

		// Return clear error message - will be stored in DB by task persistence
		// Return StackynError for proper error handling
		return stackynErr
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

	// Build completed - status will be stored in DB

	// Step 6: Enqueue deploy task after successful build
	if h.taskEnqueue != nil {
		// Generate deployment ID
		deploymentID := uuid.New().String()

		// Extract image name without tag (deploy handler will add tag from BuildJobID)
		// buildResult.ImageName is in format "imageName:tag", we need just "imageName"
		imageName := buildResult.ImageName
		if idx := strings.LastIndex(imageName, ":"); idx > 0 {
			imageName = imageName[:idx]
		}

		// Prepare deploy task payload
		deployPayload := DeployTaskPayload{
			AppID:        payload.AppID,
			DeploymentID: deploymentID,
			BuildJobID:   payload.BuildJobID,
			ImageName:    imageName,
			UserID:       payload.UserID,
			// Default RAM request (can be overridden by plan limits)
			RequestedRAMMB: 512,
			UseDockerCompose: hasDockerCompose,
			RepoPath:      cloneResult.Path, // Pass repo path for docker-compose deployment
		}

		// Enqueue deploy task
		taskInfo, err := h.taskEnqueue.EnqueueDeployTask(ctx, deployPayload, payload.UserID)
		if err != nil {
			h.logger.Error("Failed to enqueue deploy task",
				zap.String("app_id", payload.AppID),
				zap.String("build_job_id", payload.BuildJobID),
				zap.Error(err),
			)
			// Don't fail the build task if deployment enqueue fails
			// The deployment can be triggered manually later
		} else {
			h.logger.Info("Deploy task enqueued successfully",
				zap.String("app_id", payload.AppID),
				zap.String("build_job_id", payload.BuildJobID),
				zap.String("deployment_id", deploymentID),
				zap.String("task_id", taskInfo.ID),
			)
			// Deployment started - status will be stored in DB
		}
	} else {
		h.logger.Warn("Task enqueue service not available - deployment not started",
			zap.String("app_id", payload.AppID),
			zap.String("build_job_id", payload.BuildJobID),
		)
	}

	// TODO: Step 7: Push to registry
	// TODO: Step 8: Update build job status in database

	return nil
}

// broadcastStatus removed - DB is single source of truth

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

	// Update app status to "deploying" when deployment starts
	if h.appRepo != nil {
		if err := h.appRepo.UpdateApp(payload.AppID, "deploying", ""); err != nil {
			h.logger.Warn("Failed to update app status to deploying",
				zap.Error(err),
				zap.String("app_id", payload.AppID),
			)
		}
	}

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
		// Get base domain from environment variable
		baseDomain := os.Getenv("APP_BASE_DOMAIN")
		if baseDomain == "" {
			// Default to .local for local development
			baseDomain = "stackyn.local"
		}
		subdomain = fmt.Sprintf("%s.%s", payload.AppID, baseDomain)
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
	// TODO: Re-enable RAM limit checks in production
	if h.planEnforcement != nil {
		// RAM limit check disabled for now
		// if err := h.planEnforcement.CheckMaxRAM(ctx, userID, memoryMB); err != nil {
		// 	return fmt.Errorf("plan limit exceeded: %w", err)
		// }

		// Increment RAM usage (still track usage, but don't enforce limits)
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
		UseDockerCompose: payload.UseDockerCompose,
		ComposeFilePath: payload.RepoPath, // Path to repository containing docker-compose.yml
	}

	// Deploy container (using docker-compose if detected)
	var deployResult *services.DeploymentResult
	var err error
	
	if payload.UseDockerCompose {
		// If docker-compose is needed, ensure we have the repo path
		repoPath := payload.RepoPath
		if repoPath == "" || !h.pathExists(repoPath) {
			// Repo path not available or cleaned up
			h.logger.Error("Repo path not available for docker-compose deployment",
				zap.String("app_id", payload.AppID),
				zap.String("repo_path", repoPath),
			)
			return fmt.Errorf("repository path not available for docker-compose deployment: %s", repoPath)
		}
		
		h.logger.Info("Deploying with docker-compose",
			zap.String("app_id", payload.AppID),
			zap.String("repo_path", repoPath),
		)
		deployOpts.ComposeFilePath = repoPath
		deployResult, err = h.deploymentService.DeployWithDockerCompose(ctx, deployOpts)
		
		// Cleanup repo path after docker-compose deployment completes (success or failure)
		if h.gitService != nil {
			if cleanupErr := h.gitService.Cleanup(repoPath); cleanupErr != nil {
				h.logger.Warn("Failed to cleanup clone directory after docker-compose deployment",
					zap.String("app_id", payload.AppID),
					zap.String("repo_path", repoPath),
					zap.Error(cleanupErr),
				)
			} else {
				h.logger.Info("Cleaned up repo path after docker-compose deployment",
					zap.String("app_id", payload.AppID),
					zap.String("repo_path", repoPath),
				)
			}
		}
	} else {
		deployResult, err = h.deploymentService.DeployContainer(ctx, deployOpts)
	}
	if err != nil {
		// On deployment failure, store error in database
		if h.deploymentRepo != nil {
			fullImageName := fmt.Sprintf("%s:%s", imageName, imageTag)
			errorMsg := err.Error()
			// Create failed deployment record
			deploymentID, createErr := h.deploymentRepo.CreateDeployment(
				payload.AppID,
				payload.BuildJobID,
				"failed",
				fullImageName,
				"",
				subdomain,
			)
		if createErr == nil && deploymentID != "" {
			// Update with error message
			updateErr := h.deploymentRepo.UpdateDeployment(deploymentID, "", "", "", "", errorMsg)
				if updateErr != nil {
					h.logger.Warn("Failed to update deployment error message", zap.Error(updateErr))
				} else {
				h.logger.Debug("Failed deployment recorded in database",
					zap.String("app_id", payload.AppID),
					zap.String("deployment_id", deploymentID),
				)
				}
			} else {
				h.logger.Warn("Failed to store failed deployment in database", zap.Error(createErr))
			}
		}
		
		h.logger.Error("Deployment failed",
			zap.String("app_id", payload.AppID),
			zap.String("deployment_id", payload.DeploymentID),
			zap.Error(err),
		)
		
		return fmt.Errorf("failed to deploy container: %w", err)
	}

	h.logger.Info("Deploy task completed",
		zap.String("app_id", payload.AppID),
		zap.String("deployment_id", payload.DeploymentID),
		zap.String("container_id", deployResult.ContainerID),
		zap.String("container_name", deployResult.ContainerName),
		zap.String("status", deployResult.Status),
	)

	// Update deployment status in database
	var dbDeploymentID string
	if h.deploymentRepo != nil {
		// Try to create deployment record (if it doesn't exist) or update existing one
		// For now, we'll try to find deployment by matching container name pattern or create new
		// In production, deployment ID should be stored when build completes
		
		fullImageName := fmt.Sprintf("%s:%s", imageName, imageTag)
		// Use subdomain from deployment options (it's set earlier in the function)
		// subdomain variable is already defined above
		
		// Try to create deployment - if deployment ID format allows lookup, we could update instead
		// For simplicity, always create new deployment record
		// Use subdomain from deployOpts (already defined above)
		dbDeploymentID, err = h.deploymentRepo.CreateDeployment(
			payload.AppID,
			payload.BuildJobID,
			deployResult.Status,
			fullImageName,
			deployResult.ContainerID,
			deployOpts.Subdomain,
		)
		if err != nil {
			// If creation fails (e.g., duplicate), try to update by finding existing deployment
			// For now, just log the error
			h.logger.Warn("Failed to create deployment record", 
				zap.Error(err),
				zap.String("app_id", payload.AppID),
				zap.String("deployment_id", payload.DeploymentID),
			)
		} else {
			h.logger.Info("Deployment record created in database",
				zap.String("db_deployment_id", dbDeploymentID),
				zap.String("app_id", payload.AppID),
				zap.String("deployment_id", payload.DeploymentID),
			)
		}
	} else {
		h.logger.Warn("Deployment repository not available - deployment not stored in DB")
	}

	// Update app status and URL after successful deployment
	if h.appRepo != nil && deployResult.Status == "running" {
		// Generate URL from subdomain
		// Use HTTP for .local domains, HTTPS for production domains
		var appURL string
		if strings.HasSuffix(deployOpts.Subdomain, ".local") || strings.HasSuffix(deployOpts.Subdomain, ".localhost") {
			appURL = fmt.Sprintf("http://%s", deployOpts.Subdomain)
		} else {
			appURL = fmt.Sprintf("https://%s", deployOpts.Subdomain)
		}
		
		// First, set status to "running" (will be updated to "error" if health check fails)
		if err := h.appRepo.UpdateApp(payload.AppID, "running", appURL); err != nil {
			h.logger.Warn("Failed to update app status and URL",
				zap.Error(err),
				zap.String("app_id", payload.AppID),
				zap.String("status", "running"),
				zap.String("url", appURL),
			)
		} else {
			h.logger.Info("App status and URL updated successfully",
				zap.String("app_id", payload.AppID),
				zap.String("status", "running"),
				zap.String("url", appURL),
			)

			// Wait a bit for container to fully start and Traefik to configure
			// Then run initial health check (use DB deployment ID for health check)
			// Give extra time for SSL certificate issuance (Let's Encrypt can take 1-2 minutes)
			if dbDeploymentID != "" {
				go func() {
					time.Sleep(60 * time.Second) // Wait 60 seconds for container and SSL cert to be ready
					h.performHealthCheck(context.Background(), payload.AppID, dbDeploymentID, deployResult.ContainerID, appURL)
				}()
			}
		}
	} else if h.appRepo == nil {
		h.logger.Warn("App repository not available - app status not updated")
	}

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

// extractBuildError extracts meaningful, user-friendly error messages from build logs
func (h *TaskHandler) extractBuildError(logs string, buildErr error) string {
	if logs == "" {
		return "Build failed. Please check your application configuration and try again."
	}

	// Clean and normalize the logs
	cleanedLogs := h.cleanBuildLogs(logs)
	lines := strings.Split(cleanedLogs, "\n")
	
	// Try to extract specific error types with user-friendly messages
	errorMsg := h.extractSpecificError(lines)
	if errorMsg != "" {
		return errorMsg
	}
	
	// Fallback: extract general error message
	return h.extractGeneralError(lines, buildErr)
}

// cleanBuildLogs removes Docker build artifacts, escape sequences, and formatting
func (h *TaskHandler) cleanBuildLogs(logs string) string {
	// Remove JSON escaping
	logs = strings.ReplaceAll(logs, "\\n", "\n")
	logs = strings.ReplaceAll(logs, "\\\"", "\"")
	logs = strings.ReplaceAll(logs, "\\u003c", "<")
	logs = strings.ReplaceAll(logs, "\\u003e", ">")
	logs = strings.ReplaceAll(logs, "\\u0026", "&")
	logs = strings.ReplaceAll(logs, "\\u001b", "")
	
	// Remove Docker build log artifacts
	logs = strings.ReplaceAll(logs, "\"}", "")
	logs = strings.ReplaceAll(logs, "{\"", "")
	logs = strings.ReplaceAll(logs, "\"", "")
	
	// Remove ANSI color codes
	logs = h.removeANSICodes(logs)
	
	return logs
}

// extractSpecificError extracts user-friendly messages for specific error types
func (h *TaskHandler) extractSpecificError(lines []string) string {
	logsText := strings.Join(lines, "\n")
	lowerLogs := strings.ToLower(logsText)
	
	// Poetry errors
	if strings.Contains(lowerLogs, "poetry") {
		if strings.Contains(lowerLogs, "group(s) not found") {
			// Extract the group name if possible
			if match := h.findPattern(lines, `Group\(s\) not found: (\w+)`); match != "" {
				return fmt.Sprintf("Poetry configuration error: The dependency group '%s' specified in your pyproject.toml was not found. Please check your pyproject.toml file and ensure the group exists, or remove the --only flag from your build configuration.", match)
			}
			return "Poetry configuration error: A dependency group specified in your build configuration was not found in pyproject.toml. Please check your pyproject.toml file and ensure all referenced groups exist."
		}
		if strings.Contains(lowerLogs, "poetry install failed") || strings.Contains(lowerLogs, "poetry sync failed") {
			return "Poetry dependency installation failed. Please check your pyproject.toml file for errors and ensure all dependencies are correctly specified."
		}
		if strings.Contains(lowerLogs, "solver problem") || strings.Contains(lowerLogs, "dependency resolution") {
			return "Poetry dependency resolution failed. Some dependencies in your pyproject.toml have conflicting version requirements. Please review and update your dependency versions."
		}
	}
	
	// npm/Node.js errors
	if strings.Contains(lowerLogs, "npm") || strings.Contains(lowerLogs, "node") {
		if strings.Contains(lowerLogs, "package.json") && strings.Contains(lowerLogs, "not found") {
			return "Node.js application error: package.json file is missing or invalid. Please ensure your repository contains a valid package.json file."
		}
		if strings.Contains(lowerLogs, "npm install failed") || strings.Contains(lowerLogs, "npm err") {
			return "npm dependency installation failed. Please check your package.json file and ensure all dependencies are correctly specified and compatible."
		}
		if strings.Contains(lowerLogs, "peer dependency") {
			return "npm peer dependency conflict. Some packages require incompatible versions of other packages. Please update your package.json to resolve version conflicts."
		}
	}
	
	// Python/pip errors
	if strings.Contains(lowerLogs, "pip") && !strings.Contains(lowerLogs, "poetry") {
		if strings.Contains(lowerLogs, "requirements.txt") && strings.Contains(lowerLogs, "not found") {
			return "Python application error: requirements.txt file is missing. Please add a requirements.txt file with your Python dependencies, or use pyproject.toml with Poetry."
		}
		if strings.Contains(lowerLogs, "no matching distribution") {
			return "Python dependency error: One or more packages in your requirements.txt cannot be found or installed. Please check package names and versions."
		}
	}
	
	// Go errors
	if strings.Contains(lowerLogs, "go mod") || strings.Contains(lowerLogs, "go:") {
		if strings.Contains(lowerLogs, "go.mod") && strings.Contains(lowerLogs, "not found") {
			return "Go application error: go.mod file is missing. Please ensure your Go application has a valid go.mod file."
		}
		if strings.Contains(lowerLogs, "cannot find module") {
			return "Go module error: One or more Go modules cannot be found. Please check your go.mod file and ensure all module paths are correct."
		}
	}
	
	// Java/Maven errors
	if strings.Contains(lowerLogs, "maven") || strings.Contains(lowerLogs, "pom.xml") {
		if strings.Contains(lowerLogs, "pom.xml") && strings.Contains(lowerLogs, "not found") {
			return "Java application error: pom.xml file is missing. Please ensure your Java application has a valid pom.xml file."
		}
		if strings.Contains(lowerLogs, "dependency resolution") {
			return "Maven dependency resolution failed. Please check your pom.xml file for dependency conflicts or missing repositories."
		}
	}
	
	// Buildpack errors
	if strings.Contains(lowerLogs, "paketo buildpacks build failed") {
		return "Buildpack build failed. Please ensure your application has the required configuration files (package.json for Node.js, requirements.txt or pyproject.toml for Python, go.mod for Go, pom.xml for Java)."
	}
	
	// Docker errors
	if strings.Contains(lowerLogs, "dockerfile") && strings.Contains(lowerLogs, "not found") {
		return "Dockerfile not found. Please ensure your repository contains a Dockerfile, or use a supported runtime (Node.js, Python, Go, Java) with the required configuration files."
	}
	
	return ""
}

// extractGeneralError extracts a general user-friendly error message
func (h *TaskHandler) extractGeneralError(lines []string, buildErr error) string {
	// Look for ERROR lines from the end (most recent errors first)
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		
		// Skip Docker build step lines
		if strings.HasPrefix(line, "Step ") && strings.Contains(line, "/") {
			continue
		}
		
		// Skip Docker build output markers
		if strings.HasPrefix(line, "} --->") || strings.HasPrefix(line, "}") || strings.HasPrefix(line, "---") {
			continue
		}
		
		// Skip buildpack output markers
		if strings.HasPrefix(line, "Paketo Buildpack") || strings.HasPrefix(line, "======== Output:") {
			continue
		}
		
		// Look for meaningful error messages
		lowerLine := strings.ToLower(line)
		if strings.Contains(lowerLine, "error:") {
			// Extract text after "error:"
			if idx := strings.Index(lowerLine, "error:"); idx >= 0 {
				errorText := strings.TrimSpace(line[idx+6:])
				if errorText != "" && len(errorText) < 300 {
					return fmt.Sprintf("Build error: %s", errorText)
				}
			}
		}
		if strings.Contains(lowerLine, "failed:") {
			if idx := strings.Index(lowerLine, "failed:"); idx >= 0 {
				errorText := strings.TrimSpace(line[idx+7:])
				if errorText != "" && len(errorText) < 300 {
					return fmt.Sprintf("Build failed: %s", errorText)
				}
			}
		}
		if strings.Contains(lowerLine, "cannot") && len(line) < 300 {
			return fmt.Sprintf("Build error: %s", line)
		}
	}
	
	// Final fallback
	return "Build failed. Please check your application configuration, dependencies, and ensure all required files are present in your repository."
}

// findPattern finds a pattern in lines and returns the first match group
func (h *TaskHandler) findPattern(lines []string, pattern string) string {
	re := regexp.MustCompile(pattern)
	for _, line := range lines {
		matches := re.FindStringSubmatch(line)
		if len(matches) > 1 {
			return matches[1]
		}
	}
	return ""
}

// hasDockerComposeFile checks if a docker-compose file exists in the repository
func (h *TaskHandler) hasDockerComposeFile(repoPath string) bool {
	dockerComposeFiles := []string{
		"docker-compose.yml",
		"docker-compose.yaml",
		"compose.yml",
		"compose.yaml",
		".docker-compose.yml",
		".docker-compose.yaml",
	}
	
	for _, filename := range dockerComposeFiles {
		filePath := filepath.Join(repoPath, filename)
		if _, err := os.Stat(filePath); err == nil {
			h.logger.Info("Found docker-compose file",
				zap.String("file", filename),
				zap.String("path", filePath),
			)
			return true
		}
	}
	
	return false
}

// pathExists checks if a path exists
func (h *TaskHandler) pathExists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

// removeANSICodes removes ANSI escape codes from a string
func (h *TaskHandler) removeANSICodes(s string) string {
	// Remove ANSI escape sequences (e.g., \u001b[31;1m, \u001b[0m)
	var result strings.Builder
	inEscape := false
	
	for _, r := range s {
		if r == '\u001b' || r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if r == 'm' {
				inEscape = false
			}
			continue
		}
		if !inEscape {
			result.WriteRune(r)
		}
	}
	
	return result.String()
}

