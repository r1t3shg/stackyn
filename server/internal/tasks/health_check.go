package tasks

import (
	"context"
	"time"

	"github.com/docker/docker/client"
	"go.uber.org/zap"
	"stackyn/server/internal/services"
)

// performHealthCheck performs a health check on a deployed app and updates status if it fails
func (h *TaskHandler) performHealthCheck(ctx context.Context, appID, deploymentID, containerID, url string) {
	if h.deploymentService == nil {
		return
	}

	// Get docker client from deployment service
	dockerClient := h.deploymentService.GetDockerClient()
	if dockerClient == nil {
		h.logger.Warn("Docker client not available for health check",
			zap.String("app_id", appID),
		)
		return
	}

	// Create health check service
	healthService := services.NewHealthCheckService(dockerClient, h.logger)
	
	// Set callback to update database when health check fails
	healthService.SetHealthCheckCallback(func(appID, deploymentID, errorMsg string) {
		h.logger.Warn("Health check failed, updating app status to error",
			zap.String("app_id", appID),
			zap.String("deployment_id", deploymentID),
			zap.String("error", errorMsg),
		)

		// Update deployment status to error
		if h.deploymentRepo != nil {
			err := h.deploymentRepo.UpdateDeployment(deploymentID, "error", "", containerID, "", errorMsg)
			if err != nil {
				h.logger.Error("Failed to update deployment status to error",
					zap.Error(err),
					zap.String("deployment_id", deploymentID),
				)
			}
		}

		// Update app status to error
		if h.appRepo != nil {
			err := h.appRepo.UpdateApp(appID, "error", url)
			if err != nil {
				h.logger.Error("Failed to update app status to error",
					zap.Error(err),
					zap.String("app_id", appID),
				)
			}
		}
	})

	// Perform initial health check after a delay (allow container to start)
	go func() {
		time.Sleep(15 * time.Second) // Wait 15 seconds for container to be ready
		
		err := healthService.CheckAppAccessibility(ctx, appID, deploymentID, url, containerID)
		if err != nil {
			h.logger.Warn("Initial health check failed",
				zap.String("app_id", appID),
				zap.String("deployment_id", deploymentID),
				zap.Error(err),
			)
		} else {
			h.logger.Info("Initial health check passed",
				zap.String("app_id", appID),
				zap.String("deployment_id", deploymentID),
			)
		}
		
		// Start continuous monitoring
		monitorCtx := context.Background() // Use background context for long-running monitoring
		healthService.MonitorAppHealth(monitorCtx, appID, deploymentID, url, containerID)
	}()
}

