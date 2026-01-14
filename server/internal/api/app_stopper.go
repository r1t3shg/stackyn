package api

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

// AppStopperImpl implements services.AppStopper to stop all apps for a user
type AppStopperImpl struct {
	appRepo          *AppRepo
	deploymentService DeploymentService
	logger           *zap.Logger
}

// NewAppStopper creates a new app stopper
func NewAppStopper(appRepo *AppRepo, deploymentService DeploymentService, logger *zap.Logger) *AppStopperImpl {
	return &AppStopperImpl{
		appRepo:          appRepo,
		deploymentService: deploymentService,
		logger:           logger,
	}
}

// StopAllUserApps stops all running apps for a user
func (s *AppStopperImpl) StopAllUserApps(ctx context.Context, userID string) error {
	// Get all apps for the user
	apps, err := s.appRepo.GetAppsByUserID(userID)
	if err != nil {
		return fmt.Errorf("failed to get user apps: %w", err)
	}

	if len(apps) == 0 {
		s.logger.Info("No apps to stop for user", zap.String("user_id", userID))
		return nil
	}

	s.logger.Info("Stopping all apps for user",
		zap.String("user_id", userID),
		zap.Int("app_count", len(apps)),
	)

	// Stop each app and mark as disabled
	for _, app := range apps {
		// Mark app as disabled in database (idempotent - safe to run multiple times)
		if err := s.appRepo.UpdateApp(app.ID, "disabled", ""); err != nil {
			s.logger.Warn("Failed to mark app as disabled",
				zap.Error(err),
				zap.String("app_id", app.ID),
				zap.String("app_name", app.Name),
				zap.String("user_id", userID),
			)
			// Continue stopping other apps even if marking fails
		}

		if s.deploymentService != nil {
			// Cleanup app resources (stops containers)
			if err := s.deploymentService.CleanupAppResources(ctx, app.ID); err != nil {
				s.logger.Warn("Failed to stop app containers",
					zap.Error(err),
					zap.String("app_id", app.ID),
					zap.String("app_name", app.Name),
					zap.String("user_id", userID),
				)
				// Continue stopping other apps even if one fails
			} else {
				s.logger.Info("Stopped app containers",
					zap.String("app_id", app.ID),
					zap.String("app_name", app.Name),
					zap.String("user_id", userID),
				)
			}
		} else {
			s.logger.Warn("Deployment service not available, cannot stop app containers",
				zap.String("app_id", app.ID),
				zap.String("user_id", userID),
			)
		}

		s.logger.Info("App disabled and stopped",
			zap.String("app_id", app.ID),
			zap.String("app_name", app.Name),
			zap.String("user_id", userID),
		)
	}

	return nil
}

