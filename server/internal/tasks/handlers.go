package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"
	"stackyn/server/internal/services"
)

// TaskHandler handles task processing
type TaskHandler struct {
	logger     *zap.Logger
	gitService GitService
	// Add dependencies here (database, docker client, etc.)
}

// GitService interface for repository operations
// Uses services package types to avoid duplication
type GitService interface {
	ValidatePublicRepo(ctx context.Context, repoURL string) error
	Clone(ctx context.Context, opts services.CloneOptions) (*services.CloneResult, error)
	Cleanup(clonePath string) error
}

// NewTaskHandler creates a new task handler
func NewTaskHandler(logger *zap.Logger, gitService GitService) *TaskHandler {
	return &TaskHandler{
		logger:     logger,
		gitService: gitService,
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

	// Clone repository with shallow clone
	var cloneResult *services.CloneResult
	if h.gitService != nil {
		cloneOpts := services.CloneOptions{
			RepoURL: payload.RepoURL,
			Branch:  payload.Branch,
			Shallow: true, // Always use shallow clone
			Depth:   1,    // Only clone the latest commit
		}

		var err error
		cloneResult, err = h.gitService.Clone(ctx, cloneOpts)
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
	} else {
		return fmt.Errorf("git service not configured")
	}

	// TODO: Implement actual build logic
	// 1. ✅ Clone repository (done)
	// 2. Build Docker image from cloned repo
	// 3. Push to registry
	// 4. Update build job status in database
	// 5. Persist task state

	// Simulate work
	time.Sleep(1 * time.Second)

	h.logger.Info("Build task completed",
		zap.String("app_id", payload.AppID),
		zap.String("build_job_id", payload.BuildJobID),
		zap.String("commit_sha", cloneResult.CommitSHA),
	)

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
	)

	// TODO: Implement actual deploy logic
	// 1. Pull Docker image
	// 2. Create/update container
	// 3. Configure networking (Traefik)
	// 4. Start container
	// 5. Update deployment status in database
	// 6. Persist task state

	// Simulate work
	time.Sleep(1 * time.Second)

	h.logger.Info("Deploy task completed",
		zap.String("app_id", payload.AppID),
		zap.String("deployment_id", payload.DeploymentID),
	)

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

