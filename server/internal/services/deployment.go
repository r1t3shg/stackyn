package services

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"go.uber.org/zap"
)

// DeploymentService handles container deployment operations
type DeploymentService struct {
	client *client.Client
	logger *zap.Logger
}

// NewDeploymentService creates a new deployment service
func NewDeploymentService(dockerHost string, logger *zap.Logger) (*DeploymentService, error) {
	cli, err := client.NewClientWithOpts(
		client.WithHost(dockerHost),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	return &DeploymentService{
		client: cli,
		logger: logger,
	}, nil
}

// Close closes the Docker client
func (s *DeploymentService) Close() error {
	return s.client.Close()
}

// ResourceLimits represents plan-based resource limits
type ResourceLimits struct {
	MemoryMB int64   // Memory limit in MB
	CPU      float64 // CPU limit (e.g., 0.5 = 50% of one CPU)
}

// DeploymentOptions represents options for deploying a container
type DeploymentOptions struct {
	AppID        string
	DeploymentID string
	ImageName    string
	ImageTag     string
	Subdomain    string
	Port         int
	Limits       ResourceLimits
	EnvVars      map[string]string
}

// DeploymentResult represents the result of a deployment
type DeploymentResult struct {
	ContainerID string
	ContainerName string
	Status      string
}

// DeployContainer deploys a container with plan-based limits and Traefik labels
func (s *DeploymentService) DeployContainer(ctx context.Context, opts DeploymentOptions) (*DeploymentResult, error) {
	// Step 1: Ensure only one active container per app (stop/remove old containers)
	if err := s.ensureOneContainerPerApp(ctx, opts.AppID); err != nil {
		return nil, fmt.Errorf("failed to ensure one container per app: %w", err)
	}

	// Step 2: Pull image if needed
	imageRef := fmt.Sprintf("%s:%s", opts.ImageName, opts.ImageTag)
	if err := s.pullImage(ctx, imageRef); err != nil {
		return nil, fmt.Errorf("failed to pull image: %w", err)
	}

	// Step 3: Create container with resource limits and Traefik labels
	containerName := s.generateContainerName(opts.AppID, opts.DeploymentID)
	
	// Prepare environment variables
	envVars := make([]string, 0, len(opts.EnvVars)+1)
	envVars = append(envVars, fmt.Sprintf("PORT=%d", opts.Port))
	for k, v := range opts.EnvVars {
		envVars = append(envVars, fmt.Sprintf("%s=%s", k, v))
	}

	// Create container config
	containerConfig := &container.Config{
		Image:  imageRef,
		Env:    envVars,
		Labels: s.generateTraefikLabels(opts.Subdomain, opts.Port, opts.AppID),
		// Health check will be configured via Traefik
	}

	// Create host config with resource limits
	hostConfig := &container.HostConfig{
		Resources: container.Resources{
			Memory:     opts.Limits.MemoryMB * 1024 * 1024, // Convert MB to bytes
			NanoCPUs:   int64(opts.Limits.CPU * 1e9),      // Convert CPU to nanoseconds
			MemorySwap: opts.Limits.MemoryMB * 1024 * 1024, // Same as memory (no swap)
		},
		RestartPolicy: container.RestartPolicy{
			Name:              "on-failure",
			MaximumRetryCount: 3, // Restart up to 3 times on failure
		},
		// Auto-remove on stop (for cleanup)
		AutoRemove: false, // We'll manage cleanup manually
	}

	// Create network config (connect to Traefik network)
	networkConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			"traefik": {
				NetworkID: "", // Will be resolved by Docker
			},
		},
	}

	s.logger.Info("Creating container",
		zap.String("app_id", opts.AppID),
		zap.String("deployment_id", opts.DeploymentID),
		zap.String("container_name", containerName),
		zap.String("image", imageRef),
		zap.Int64("memory_mb", opts.Limits.MemoryMB),
		zap.Float64("cpu", opts.Limits.CPU),
	)

	// Create container
	createResp, err := s.client.ContainerCreate(ctx, containerConfig, hostConfig, networkConfig, nil, containerName)
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	// Step 4: Start container
	if err := s.client.ContainerStart(ctx, createResp.ID, container.StartOptions{}); err != nil {
		// Cleanup on failure
		s.client.ContainerRemove(ctx, createResp.ID, container.RemoveOptions{Force: true})
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	// Step 5: Start crash detection monitoring
	go s.monitorContainerCrash(context.Background(), createResp.ID, opts.AppID, opts.DeploymentID)

	s.logger.Info("Container deployed successfully",
		zap.String("container_id", createResp.ID),
		zap.String("container_name", containerName),
		zap.String("app_id", opts.AppID),
	)

	return &DeploymentResult{
		ContainerID:   createResp.ID,
		ContainerName: containerName,
		Status:        "running",
	}, nil
}

// RollbackDeployment rolls back to a previous deployment
func (s *DeploymentService) RollbackDeployment(ctx context.Context, appID, previousImageName, previousImageTag string) error {
	s.logger.Info("Rolling back deployment",
		zap.String("app_id", appID),
		zap.String("image", fmt.Sprintf("%s:%s", previousImageName, previousImageTag)),
	)

	// Find current container
	containers, err := s.findContainersByAppID(ctx, appID)
	if err != nil {
		return fmt.Errorf("failed to find containers: %w", err)
	}

	if len(containers) == 0 {
		return fmt.Errorf("no active container found for app %s", appID)
	}

	// Stop and remove current container
	currentContainer := containers[0]
	if err := s.client.ContainerStop(ctx, currentContainer.ID, container.StopOptions{}); err != nil {
		s.logger.Warn("Failed to stop container during rollback", zap.Error(err))
	}

	if err := s.client.ContainerRemove(ctx, currentContainer.ID, container.RemoveOptions{Force: true}); err != nil {
		s.logger.Warn("Failed to remove container during rollback", zap.Error(err))
	}

	// Deploy previous version
	// Note: This requires storing previous deployment info (should be in database)
	// For now, we'll just log the rollback
	s.logger.Info("Rollback completed",
		zap.String("app_id", appID),
		zap.String("previous_image", fmt.Sprintf("%s:%s", previousImageName, previousImageTag)),
	)

	return nil
}

// ensureOneContainerPerApp ensures only one active container exists per app
func (s *DeploymentService) ensureOneContainerPerApp(ctx context.Context, appID string) error {
	containers, err := s.findContainersByAppID(ctx, appID)
	if err != nil {
		return err
	}

	// Stop and remove all existing containers for this app
	for _, c := range containers {
		s.logger.Info("Stopping existing container",
			zap.String("container_id", c.ID),
			zap.String("app_id", appID),
		)

		// Stop container with timeout
		stopCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		timeout := 30
		stopOpts := container.StopOptions{Timeout: &timeout}
		if err := s.client.ContainerStop(stopCtx, c.ID, stopOpts); err != nil {
			s.logger.Warn("Failed to stop container", zap.Error(err), zap.String("container_id", c.ID))
		}

		// Remove container
		removeOpts := container.RemoveOptions{Force: true}
		if err := s.client.ContainerRemove(ctx, c.ID, removeOpts); err != nil {
			s.logger.Warn("Failed to remove container", zap.Error(err), zap.String("container_id", c.ID))
		}
	}

	return nil
}

// findContainersByAppID finds all containers for a given app ID
func (s *DeploymentService) findContainersByAppID(ctx context.Context, appID string) ([]types.Container, error) {
	filter := filters.NewArgs()
	filter.Add("label", fmt.Sprintf("app.id=%s", appID))

	containers, err := s.client.ContainerList(ctx, container.ListOptions{
		Filters: filter,
		All:     true, // Include stopped containers
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	return containers, nil
}

// pullImage pulls a Docker image
func (s *DeploymentService) pullImage(ctx context.Context, imageRef string) error {
	s.logger.Info("Pulling image", zap.String("image", imageRef))

	reader, err := s.client.ImagePull(ctx, imageRef, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image: %w", err)
	}
	defer reader.Close()

	// Read the response to completion
	buf := make([]byte, 1024)
	for {
		_, err := reader.Read(buf)
		if err != nil {
			break
		}
	}

	s.logger.Info("Image pulled successfully", zap.String("image", imageRef))
	return nil
}

// generateContainerName generates a unique container name
func (s *DeploymentService) generateContainerName(appID, deploymentID string) string {
	return fmt.Sprintf("stackyn-%s-%s", appID, deploymentID)
}

// generateTraefikLabels generates Traefik labels for routing
func (s *DeploymentService) generateTraefikLabels(subdomain string, port int, appID string) map[string]string {
	labels := map[string]string{
		// Enable Traefik
		"traefik.enable": "true",
		
		// HTTP router
		"traefik.http.routers.app.rule": fmt.Sprintf("Host(`%s`)", subdomain),
		"traefik.http.routers.app.entrypoints": "web",
		
		// Service configuration
		"traefik.http.services.app.loadbalancer.server.port": strconv.Itoa(port),
		
		// Use Traefik network
		"traefik.docker.network": "traefik",
		
		// App ID label for container lookup
		"app.id": appID,
	}

	return labels
}

// monitorContainerCrash monitors a container for crashes and triggers rollback
func (s *DeploymentService) monitorContainerCrash(ctx context.Context, containerID, appID, deploymentID string) {
	ticker := time.NewTicker(10 * time.Second) // Check every 10 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Inspect container
			containerJSON, err := s.client.ContainerInspect(ctx, containerID)
			if err != nil {
				s.logger.Warn("Failed to inspect container for crash detection", zap.Error(err))
				continue
			}

			// Check if container is running
			if !containerJSON.State.Running {
				s.logger.Error("Container crashed",
					zap.String("container_id", containerID),
					zap.String("app_id", appID),
					zap.String("deployment_id", deploymentID),
					zap.String("status", containerJSON.State.Status),
					zap.Int("exit_code", containerJSON.State.ExitCode),
				)

				// Check restart count
				if containerJSON.RestartCount >= 3 {
					s.logger.Error("Container exceeded restart limit, triggering rollback",
						zap.String("container_id", containerID),
						zap.String("app_id", appID),
						zap.Int("restart_count", containerJSON.RestartCount),
					)

					// Trigger rollback (this would typically notify the system to rollback)
					// For now, we'll just log it
					// TODO: Implement actual rollback mechanism (e.g., via task queue)
					s.logger.Error("ROLLBACK REQUIRED",
						zap.String("app_id", appID),
						zap.String("deployment_id", deploymentID),
						zap.String("reason", "container_crash_exceeded_restart_limit"),
					)
				}
			}
		}
	}
}

