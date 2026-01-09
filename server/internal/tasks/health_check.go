package tasks

import (
	"context"
	"time"

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
		h.logger.Warn("Health check failed, checking if deployment is still active",
			zap.String("app_id", appID),
			zap.String("deployment_id", deploymentID),
			zap.String("error", errorMsg),
		)

		// Update deployment status to error (always update the specific deployment)
		if h.deploymentRepo != nil {
			err := h.deploymentRepo.UpdateDeployment(deploymentID, "error", "", containerID, "", errorMsg)
			if err != nil {
				h.logger.Error("Failed to update deployment status to error",
					zap.Error(err),
					zap.String("deployment_id", deploymentID),
				)
			}
		}

		// Only update app status to error if this is still the latest/active deployment
		// Check if there's a newer running deployment
		if h.deploymentRepo != nil && h.appRepo != nil {
			// Get all deployments for this app
			deployments, err := h.deploymentRepo.GetDeploymentsByAppID(appID)
			if err != nil {
				h.logger.Warn("Failed to get deployments to check if failed deployment is still active",
					zap.Error(err),
					zap.String("app_id", appID),
				)
				// If we can't check, err on the side of caution and don't update app status
				return
			}

			// Find the latest deployment (first one in the list, sorted by created_at DESC)
			// Check if there's a newer running deployment
			hasNewerRunningDeployment := false
			for _, dep := range deployments {
				depID, _ := dep["id"].(string)
				depStatus, _ := dep["status"].(string)
				
				// If we find a deployment that's newer and running, don't update app status
				if depID != deploymentID && depStatus == "running" {
					hasNewerRunningDeployment = true
					h.logger.Info("Found newer running deployment, not updating app status to error",
						zap.String("app_id", appID),
						zap.String("failed_deployment_id", deploymentID),
						zap.String("newer_deployment_id", depID),
					)
					break
				}
				
				// If we've reached the failed deployment, no newer running deployment exists
				if depID == deploymentID {
					break
				}
			}

			// Only update app status to error if there's no newer running deployment
			if !hasNewerRunningDeployment {
				err := h.appRepo.UpdateApp(appID, "error", url)
				if err != nil {
					h.logger.Error("Failed to update app status to error",
						zap.Error(err),
						zap.String("app_id", appID),
					)
				} else {
					h.logger.Info("Updated app status to error (no newer running deployment)",
						zap.String("app_id", appID),
						zap.String("deployment_id", deploymentID),
					)
				}
			}
		}
	})

	// Perform initial health check after a delay (allow container to start and SSL cert to be issued)
	go func() {
		time.Sleep(60 * time.Second) // Wait 60 seconds for container and SSL cert to be ready (Let's Encrypt takes time)
		
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

