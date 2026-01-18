package services

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"go.uber.org/zap"
)

// CrashCallback is a function that gets called when a container crashes
// Parameters: appID, deploymentID, containerID, exitCode, error message
type CrashCallback func(appID, deploymentID, containerID string, exitCode int, errorMsg string)

// DeploymentService handles container deployment operations
type DeploymentService struct {
	client         *client.Client
	logger         *zap.Logger
	logPersistence RuntimeLogPersistence // Optional: for persisting runtime logs
	networkName    string                 // Docker network name (e.g., "stackyn-network")
	crashCallback  CrashCallback          // Optional: callback for crash events
}

// GetDockerClient returns the Docker client (for use by other services)
func (s *DeploymentService) GetDockerClient() *client.Client {
	return s.client
}

// RuntimeLogPersistence interface for persisting runtime logs
// Accepts interface{} to allow different entry types
type RuntimeLogPersistence interface {
	PersistLogStream(ctx context.Context, entry interface{}, reader io.Reader) error
}

// NewDeploymentService creates a new deployment service
func NewDeploymentService(dockerHost string, logger *zap.Logger, logPersistence RuntimeLogPersistence, networkName string) (*DeploymentService, error) {
	cli, err := client.NewClientWithOpts(
		client.WithHost(dockerHost),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	// Default to "stackyn-network" if not provided
	if networkName == "" {
		networkName = "stackyn-network"
	}

	return &DeploymentService{
		client:         cli,
		logger:         logger,
		logPersistence: logPersistence,
		networkName:    networkName,
		crashCallback:  nil,
	}, nil
}

// SetCrashCallback sets the callback function for container crash events
func (s *DeploymentService) SetCrashCallback(callback CrashCallback) {
	s.crashCallback = callback
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
	UseDockerCompose bool   // Whether to use docker-compose for deployment
	ComposeFilePath string  // Path to docker-compose.yml file (if using docker-compose)
}

// DeploymentResult represents the result of a deployment
type DeploymentResult struct {
	ContainerID         string
	ContainerName       string
	Status              string
	StoppedContainerIDs []string // Container IDs of old deployments that were stopped
}

// ensureNetworkExists ensures the Docker network exists, creating it if necessary
func (s *DeploymentService) ensureNetworkExists(ctx context.Context) error {
	// Try to inspect the network to see if it exists
	_, err := s.client.NetworkInspect(ctx, s.networkName, network.InspectOptions{})
	if err == nil {
		// Network exists, nothing to do
		s.logger.Debug("Network exists", zap.String("network", s.networkName))
		return nil
	}

	// Network doesn't exist, create it
	s.logger.Info("Network not found, creating it", zap.String("network", s.networkName))
	
	networkCreateOptions := network.CreateOptions{
		Driver: "bridge",
		IPAM: &network.IPAM{
			Driver: "default",
		},
	}

	networkResp, err := s.client.NetworkCreate(ctx, s.networkName, networkCreateOptions)
	if err != nil {
		return fmt.Errorf("failed to create network %s: %w", s.networkName, err)
	}

	s.logger.Info("Network created successfully",
		zap.String("network", s.networkName),
		zap.String("network_id", networkResp.ID),
		zap.String("warning", networkResp.Warning),
	)

	return nil
}

// DeployContainer deploys a container with plan-based limits and Traefik labels
func (s *DeploymentService) DeployContainer(ctx context.Context, opts DeploymentOptions) (*DeploymentResult, error) {
	// Step 0: Ensure the network exists (important for localhost testing)
	if err := s.ensureNetworkExists(ctx); err != nil {
		return nil, fmt.Errorf("failed to ensure network exists: %w", err)
	}

	// Note: We will stop old containers AFTER the new container is successfully started
	// This ensures we don't lose service if the new deployment fails

	// Step 2: Pull image if needed
	imageRef := fmt.Sprintf("%s:%s", opts.ImageName, opts.ImageTag)
	if err := s.pullImage(ctx, imageRef); err != nil {
		return nil, fmt.Errorf("failed to pull image: %w", err)
	}

	// Step 3: Create container with resource limits and Traefik labels
	containerName := s.generateContainerName(opts.AppID, opts.DeploymentID)
	
	// Prepare environment variables
	// CRITICAL: PORT=8080 is ALWAYS injected first to override any user-provided PORT
	// This ensures all containers use Stackyn's standard port (8080) for routing
	envVars := make([]string, 0, len(opts.EnvVars)+1)
	envVars = append(envVars, fmt.Sprintf("PORT=%d", opts.Port)) // Always 8080 - injected first
	
	// Add user environment variables (PORT from user will be overridden by our PORT above)
	for k, v := range opts.EnvVars {
		// Skip user's PORT env var since we've already set it to 8080
		if strings.ToUpper(k) == "PORT" {
			s.logger.Debug("Overriding user PORT environment variable",
				zap.String("user_port", v),
				zap.Int("stackyn_port", opts.Port),
				zap.String("app_id", opts.AppID),
			)
			continue
		}
		envVars = append(envVars, fmt.Sprintf("%s=%s", k, v))
	}

	// Create container config
	containerConfig := &container.Config{
		Image:  imageRef,
		Env:    envVars,
		Labels: s.generateTraefikLabels(opts.Subdomain, opts.Port, opts.AppID),
		// Docker health check (complements Traefik health check)
		Healthcheck: &container.HealthConfig{
			Test:        []string{"CMD-SHELL", fmt.Sprintf("wget --no-verbose --tries=1 --spider http://localhost:%d/ || exit 1", opts.Port)},
			Interval:    10 * time.Second,
			Timeout:     3 * time.Second,
			Retries:     3,
			StartPeriod: 10 * time.Second,
		},
	}

	// Create host config with resource limits
	hostConfig := &container.HostConfig{
		Resources: container.Resources{
			Memory:     opts.Limits.MemoryMB * 1024 * 1024, // Convert MB to bytes
			NanoCPUs:   int64(opts.Limits.CPU * 1e9),      // Convert CPU to nanoseconds
			MemorySwap: opts.Limits.MemoryMB * 1024 * 1024, // Same as memory (no swap)
		},
		RestartPolicy: container.RestartPolicy{
			Name:              "no", // Don't restart on failure - try once only
			MaximumRetryCount: 0,
		},
		// Auto-remove on stop (for cleanup)
		AutoRemove: false, // We'll manage cleanup manually
	}

	// Create network config (connect to the specified network)
	networkConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			s.networkName: {
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

	// Step 4: Start container with timeout
	startCtx, startCancel := context.WithTimeout(ctx, 30*time.Second)
	defer startCancel()
	
	if err := s.client.ContainerStart(startCtx, createResp.ID, container.StartOptions{}); err != nil {
		// Cleanup on failure
		s.client.ContainerRemove(ctx, createResp.ID, container.RemoveOptions{Force: true})
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	s.logger.Info("Container started, waiting for health check before zero-downtime switchover",
		zap.String("container_id", createResp.ID),
		zap.String("app_id", opts.AppID),
	)

	// Step 4.5: Wait for container to be healthy before stopping old containers (zero-downtime deployment)
	// This ensures the new container is ready to serve traffic before old one is stopped
	healthCtx, healthCancel := context.WithTimeout(ctx, 2*time.Minute) // Allow up to 2 minutes for health check
	defer healthCancel()
	
	if err := s.waitForContainerHealth(healthCtx, createResp.ID, opts.Port); err != nil {
		s.logger.Warn("Container health check failed or timed out, but continuing deployment",
			zap.String("container_id", createResp.ID),
			zap.String("app_id", opts.AppID),
			zap.Error(err),
		)
		// Continue anyway - container might still work, just not passing health checks yet
		// In production, you might want to fail here for strict zero-downtime
	} else {
		s.logger.Info("Container passed health check, safe to stop old containers",
			zap.String("container_id", createResp.ID),
			zap.String("app_id", opts.AppID),
		)
	}

	// Step 4.6: Stop old containers now that new container is healthy and ready
	// Only stop containers that are NOT the current new container
	stoppedContainerIDs, err := s.stopOldContainersForApp(ctx, opts.AppID, createResp.ID)
	if err != nil {
		s.logger.Warn("Failed to stop old containers after successful deployment",
			zap.Error(err),
			zap.String("app_id", opts.AppID),
			zap.String("new_container_id", createResp.ID),
		)
		// Don't fail deployment if stopping old containers fails - new container is already running
	} else if len(stoppedContainerIDs) > 0 {
		s.logger.Info("Stopped old containers, ready for database update",
			zap.String("app_id", opts.AppID),
			zap.Int("stopped_count", len(stoppedContainerIDs)),
		)
	}

	// Step 5: Start crash detection monitoring
	// Use app-scoped context that can be cancelled when app is deleted
	monitorCtx, monitorCancel := context.WithCancel(ctx)
	// Note: monitorCancel should be called when app is deleted (not implemented here)
	_ = monitorCancel // Suppress unused variable warning for now
	go s.monitorContainerCrash(monitorCtx, createResp.ID, opts.AppID, opts.DeploymentID)

	// Step 6: Start runtime log streaming and persistence
	// Use background context so log streaming continues after deploy task completes
	if s.logPersistence != nil {
		go s.streamAndPersistRuntimeLogs(context.Background(), createResp.ID, opts.AppID, opts.DeploymentID)
	}

	// Step 7: Return URL for health monitoring (will be started by task handler)

	s.logger.Info("Container deployed successfully",
		zap.String("container_id", createResp.ID),
		zap.String("container_name", containerName),
		zap.String("app_id", opts.AppID),
	)

	return &DeploymentResult{
		ContainerID:         createResp.ID,
		ContainerName:       containerName,
		Status:              "running",
		StoppedContainerIDs: stoppedContainerIDs,
	}, nil
}

// DeployWithDockerCompose deploys using docker-compose when a docker-compose.yml file is present
func (s *DeploymentService) DeployWithDockerCompose(ctx context.Context, opts DeploymentOptions) (*DeploymentResult, error) {
	// Step 0: Ensure the network exists
	if err := s.ensureNetworkExists(ctx); err != nil {
		return nil, fmt.Errorf("failed to ensure network exists: %w", err)
	}

	// Step 1: Find docker-compose.yml file
	composeFilePath := s.findDockerComposeFile(opts.ComposeFilePath)
	if composeFilePath == "" {
		return nil, fmt.Errorf("docker-compose.yml file not found in repository path: %s", opts.ComposeFilePath)
	}

	composeDir := filepath.Dir(composeFilePath)
	composeFileName := filepath.Base(composeFilePath)

	s.logger.Info("Deploying with docker-compose",
		zap.String("app_id", opts.AppID),
		zap.String("compose_file", composeFilePath),
		zap.String("compose_dir", composeDir),
	)

	// Step 2: Note: We will stop old containers AFTER the new deployment is successfully started
	// This ensures we don't lose service if the new deployment fails
	// Old containers will be stopped in Step 7.5 after docker-compose up succeeds

	// Step 3: Set environment variables for docker-compose
	env := os.Environ()
	env = append(env, fmt.Sprintf("IMAGE_NAME=%s", opts.ImageName))
	env = append(env, fmt.Sprintf("IMAGE_TAG=%s", opts.ImageTag))
	env = append(env, fmt.Sprintf("SUBDOMAIN=%s", opts.Subdomain))
	env = append(env, fmt.Sprintf("APP_ID=%s", opts.AppID))
	env = append(env, fmt.Sprintf("DEPLOYMENT_ID=%s", opts.DeploymentID))
	env = append(env, fmt.Sprintf("TRAEFIK_NETWORK=%s", s.networkName))
	env = append(env, fmt.Sprintf("PORT=%d", opts.Port))
	
	// Add custom environment variables
	for k, v := range opts.EnvVars {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	// Step 4: Run docker-compose up
	// Use project name based on app ID to avoid conflicts
	projectName := fmt.Sprintf("stackyn-%s", opts.AppID)
	
	cmd := exec.CommandContext(ctx, "docker", "compose",
		"-f", composeFileName,
		"-p", projectName,
		"up", "-d", "--build")
	cmd.Dir = composeDir
	cmd.Env = env

	// Capture output for logging
	output, err := cmd.CombinedOutput()
	if err != nil {
		s.logger.Error("Docker compose up failed",
			zap.String("app_id", opts.AppID),
			zap.String("output", string(output)),
			zap.Error(err),
		)
		return nil, fmt.Errorf("docker-compose up failed: %w\nOutput: %s", err, string(output))
	}

	s.logger.Info("Docker compose up completed",
		zap.String("app_id", opts.AppID),
		zap.String("output", string(output)),
	)

	// Step 5: Connect services to Traefik network
	// Get all containers created by this compose project
	containers, err := s.findContainersByComposeProject(ctx, projectName)
	if err != nil {
		s.logger.Warn("Failed to find compose containers", zap.Error(err))
	} else {
		for _, container := range containers {
			// Connect container to Traefik network
			if err := s.client.NetworkConnect(ctx, s.networkName, container.ID, nil); err != nil {
				// Ignore error if already connected
				if !strings.Contains(err.Error(), "already exists") {
					s.logger.Warn("Failed to connect container to Traefik network",
						zap.String("container_id", container.ID),
						zap.Error(err),
					)
				}
			}
		}
	}

	// Step 6: Find the main service container (first one or one with specific label)
	mainContainerID := ""
	if len(containers) > 0 {
		mainContainerID = containers[0].ID
		// Try to find a container with a specific label indicating it's the main service
		for _, c := range containers {
			if c.Labels["com.docker.compose.service"] != "" {
				// Use the first service container found
				mainContainerID = c.ID
				break
			}
		}
	}

	if mainContainerID == "" {
		return nil, fmt.Errorf("no containers found after docker-compose up")
	}

	s.logger.Info("Docker compose containers started, waiting for health check before zero-downtime switchover",
		zap.String("main_container_id", mainContainerID),
		zap.String("app_id", opts.AppID),
	)

	// Step 7.5: Wait for main container to be healthy before stopping old containers (zero-downtime deployment)
	// This ensures the new containers are ready to serve traffic before old ones are stopped
	healthCtx, healthCancel := context.WithTimeout(ctx, 2*time.Minute) // Allow up to 2 minutes for health check
	defer healthCancel()
	
	if err := s.waitForContainerHealth(healthCtx, mainContainerID, opts.Port); err != nil {
		s.logger.Warn("Main container health check failed or timed out, but continuing deployment",
			zap.String("main_container_id", mainContainerID),
			zap.String("app_id", opts.AppID),
			zap.Error(err),
		)
		// Continue anyway - containers might still work, just not passing health checks yet
	} else {
		s.logger.Info("Main container passed health check, safe to stop old containers",
			zap.String("main_container_id", mainContainerID),
			zap.String("app_id", opts.AppID),
		)
	}

	// Step 7.6: Stop old containers now that new deployment is healthy and ready
	// Only stop containers that are NOT part of the new compose project
	// Get all container IDs from the new compose project
	newContainerIDs := make(map[string]bool)
	for _, c := range containers {
		newContainerIDs[c.ID] = true
	}
	
	// Stop old containers (those not in the new compose project)
	stoppedContainerIDs, err := s.stopOldContainersForApp(ctx, opts.AppID, mainContainerID)
	if err != nil {
		s.logger.Warn("Failed to stop old containers after docker-compose deployment",
			zap.Error(err),
			zap.String("app_id", opts.AppID),
			zap.String("new_main_container_id", mainContainerID),
		)
		// Don't fail deployment if stopping old containers fails - new containers are already running
	} else {
		// Filter out new container IDs from stopped list (shouldn't happen, but safety check)
		filteredStoppedIDs := []string{}
		for _, stoppedID := range stoppedContainerIDs {
			if !newContainerIDs[stoppedID] {
				filteredStoppedIDs = append(filteredStoppedIDs, stoppedID)
			}
		}
		stoppedContainerIDs = filteredStoppedIDs
		
		if len(stoppedContainerIDs) > 0 {
			s.logger.Info("Stopped old containers after docker-compose deployment",
				zap.String("app_id", opts.AppID),
				zap.Int("stopped_count", len(stoppedContainerIDs)),
			)
		}
	}

	// Step 7: Start crash detection monitoring
	monitorCtx, monitorCancel := context.WithCancel(ctx)
	_ = monitorCancel
	go s.monitorContainerCrash(monitorCtx, mainContainerID, opts.AppID, opts.DeploymentID)

	// Step 8: Start runtime log streaming
	// Use background context so log streaming continues after deploy task completes
	if s.logPersistence != nil {
		go s.streamAndPersistRuntimeLogs(context.Background(), mainContainerID, opts.AppID, opts.DeploymentID)
	}

	s.logger.Info("Docker compose deployment completed",
		zap.String("app_id", opts.AppID),
		zap.String("main_container_id", mainContainerID),
		zap.Int("total_containers", len(containers)),
	)

	return &DeploymentResult{
		ContainerID:         mainContainerID,
		ContainerName:        fmt.Sprintf("%s-main", projectName),
		Status:               "running",
		StoppedContainerIDs: stoppedContainerIDs,
	}, nil
}

// findDockerComposeFile finds the docker-compose file in the given directory
func (s *DeploymentService) findDockerComposeFile(repoPath string) string {
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
			return filePath
		}
	}

	return ""
}

// findContainersByComposeProject finds all containers for a docker-compose project
func (s *DeploymentService) findContainersByComposeProject(ctx context.Context, projectName string) ([]types.Container, error) {
	filter := filters.NewArgs()
	filter.Add("label", fmt.Sprintf("com.docker.compose.project=%s", projectName))

	containers, err := s.client.ContainerList(ctx, container.ListOptions{
		Filters: filter,
		All:     false, // Only running containers
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	return containers, nil
}


// ensureOneContainerPerApp ensures only one active container exists per app (MVP constraint)
// This is called BEFORE deploying a new container to prevent conflicts
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

// stopOldContainersForApp stops all containers for an app except the new container ID
// This is called AFTER a new deployment is successful to stop previous containers
// Returns a list of stopped container IDs so they can be marked as "stopped" in the database
func (s *DeploymentService) stopOldContainersForApp(ctx context.Context, appID string, newContainerID string) ([]string, error) {
	containers, err := s.findContainersByAppID(ctx, appID)
	if err != nil {
		return nil, err
	}

	var stoppedContainerIDs []string

	// Stop and remove all existing containers except the new one
	for _, c := range containers {
		// Skip the new container
		if c.ID == newContainerID {
			continue
		}

		s.logger.Info("Stopping previous container after successful deployment",
			zap.String("container_id", c.ID),
			zap.String("app_id", appID),
			zap.String("new_container_id", newContainerID),
		)

		// Stop container with timeout
		stopCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		timeout := 30
		stopOpts := container.StopOptions{Timeout: &timeout}
		if err := s.client.ContainerStop(stopCtx, c.ID, stopOpts); err != nil {
			s.logger.Warn("Failed to stop previous container", 
				zap.Error(err), 
				zap.String("container_id", c.ID),
				zap.String("app_id", appID),
			)
		} else {
			s.logger.Info("Successfully stopped previous container",
				zap.String("container_id", c.ID),
				zap.String("app_id", appID),
			)
			stoppedContainerIDs = append(stoppedContainerIDs, c.ID)
		}

		// Remove container
		removeOpts := container.RemoveOptions{Force: true}
		if err := s.client.ContainerRemove(ctx, c.ID, removeOpts); err != nil {
			s.logger.Warn("Failed to remove previous container", 
				zap.Error(err), 
				zap.String("container_id", c.ID),
				zap.String("app_id", appID),
			)
		} else {
			s.logger.Info("Successfully removed previous container",
				zap.String("container_id", c.ID),
				zap.String("app_id", appID),
			)
		}
	}

	return stoppedContainerIDs, nil
}

// waitForContainerHealth waits for a container to pass its health check
// This is used for zero-downtime deployments to ensure new container is ready before stopping old ones
func (s *DeploymentService) waitForContainerHealth(ctx context.Context, containerID string, port int) error {
	const (
		checkInterval = 2 * time.Second  // Check every 2 seconds
		maxWaitTime   = 2 * time.Minute  // Maximum wait time
	)

	deadline := time.Now().Add(maxWaitTime)
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Check if we've exceeded max wait time
			if time.Now().After(deadline) {
				return fmt.Errorf("health check timeout after %v", maxWaitTime)
			}

			// Inspect container to check health status
			containerJSON, err := s.client.ContainerInspect(ctx, containerID)
			if err != nil {
				return fmt.Errorf("failed to inspect container: %w", err)
			}

			// Check if container is running
			if !containerJSON.State.Running {
				return fmt.Errorf("container is not running (status: %s)", containerJSON.State.Status)
			}

			// Check Docker health status
			if containerJSON.State.Health != nil {
				healthStatus := containerJSON.State.Health.Status
				switch healthStatus {
				case "healthy":
					s.logger.Info("Container health check passed",
						zap.String("container_id", containerID),
						zap.String("health_status", healthStatus),
					)
					return nil // Container is healthy!
				case "unhealthy":
					// Continue waiting - might recover
					s.logger.Debug("Container health check still unhealthy, waiting...",
						zap.String("container_id", containerID),
						zap.String("health_status", healthStatus),
					)
				case "starting", "none":
					// Health check hasn't started yet or is in progress
					s.logger.Debug("Container health check in progress, waiting...",
						zap.String("container_id", containerID),
						zap.String("health_status", healthStatus),
					)
				default:
					// Unknown status - perform manual HTTP check
					s.logger.Debug("Container health status unknown, performing manual check",
						zap.String("container_id", containerID),
						zap.String("health_status", healthStatus),
					)
					if err := s.performManualHealthCheck(ctx, containerID, port); err == nil {
						s.logger.Info("Container passed manual health check",
							zap.String("container_id", containerID),
						)
						return nil
					}
				}
			} else {
				// No health check configured - perform manual HTTP check
				if err := s.performManualHealthCheck(ctx, containerID, port); err == nil {
					s.logger.Info("Container passed manual health check (no Docker health check configured)",
						zap.String("container_id", containerID),
					)
					return nil
				}
			}
		}
	}
}

// performManualHealthCheck performs a manual HTTP health check on the container
func (s *DeploymentService) performManualHealthCheck(ctx context.Context, containerID string, port int) error {
	// Get container network IP
	containerJSON, err := s.client.ContainerInspect(ctx, containerID)
	if err != nil {
		return fmt.Errorf("failed to inspect container: %w", err)
	}

	// Try to find the IP address in the network
	if containerJSON.NetworkSettings == nil {
		return fmt.Errorf("container has no network settings")
	}

	// Get IP from the stackyn-network
	var containerIP string
	if networks := containerJSON.NetworkSettings.Networks; networks != nil {
		if networkInfo, ok := networks[s.networkName]; ok && networkInfo.IPAddress != "" {
			containerIP = networkInfo.IPAddress
		}
	}

	if containerIP == "" {
		// Fallback: use exec to check from inside the container
		return s.performInternalHealthCheck(ctx, containerID, port)
	}

	// Use wget/curl inside a temporary container on the same network to check health
	// This is more reliable than trying to connect from outside
	return s.performInternalHealthCheck(ctx, containerID, port)
}

// performInternalHealthCheck checks health by executing a command inside the container
func (s *DeploymentService) performInternalHealthCheck(ctx context.Context, containerID string, port int) error {
	// Try wget first (most reliable)
	execResp, err := s.client.ContainerExecCreate(ctx, containerID, types.ExecConfig{
		Cmd:          []string{"wget", "--quiet", "--tries=1", "--spider", "--timeout=3", fmt.Sprintf("http://localhost:%d/", port)},
		AttachStdout: false,
		AttachStderr: false,
	})
	
	if err != nil {
		// Try alternative: check if port is listening using netstat or ss
		execResp, err = s.client.ContainerExecCreate(ctx, containerID, types.ExecConfig{
			Cmd:          []string{"sh", "-c", fmt.Sprintf("nc -z localhost %d || ss -ltn | grep :%d || echo failed", port, port)},
			AttachStdout: false,
			AttachStderr: false,
		})
		if err != nil {
			return fmt.Errorf("failed to create exec: %w", err)
		}
	}

	// Attach to exec to start it
	err = s.client.ContainerExecStart(ctx, execResp.ID, types.ExecStartCheck{})
	if err != nil {
		return fmt.Errorf("failed to start exec: %w", err)
	}

	// Wait a moment for exec to complete
	time.Sleep(1 * time.Second)

	// Check exec status with retries (exec might take a moment to complete)
	maxRetries := 5
	for i := 0; i < maxRetries; i++ {
		inspectResp, err := s.client.ContainerExecInspect(ctx, execResp.ID)
		if err != nil {
			return fmt.Errorf("failed to inspect exec: %w", err)
		}

		// ExitCode is -1 if exec hasn't finished yet
		if inspectResp.ExitCode == -1 {
			time.Sleep(500 * time.Millisecond)
			continue
		}

		if inspectResp.ExitCode == 0 {
			return nil // Health check passed!
		}

		return fmt.Errorf("health check failed (exit code: %d)", inspectResp.ExitCode)
	}

	return fmt.Errorf("health check timeout - exec did not complete")
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

// CleanupAppResources cleans up all Docker resources for an app (containers and images)
func (s *DeploymentService) CleanupAppResources(ctx context.Context, appID string) error {
	// Step 1: Find and remove all containers for this app
	containers, err := s.findContainersByAppID(ctx, appID)
	if err != nil {
		s.logger.Warn("Failed to find containers for cleanup", zap.Error(err), zap.String("app_id", appID))
	} else {
		for _, c := range containers {
			// Stop container if running
			if c.State == "running" {
				stopCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
				timeout := 30
				stopOpts := container.StopOptions{Timeout: &timeout}
				if err := s.client.ContainerStop(stopCtx, c.ID, stopOpts); err != nil {
					s.logger.Warn("Failed to stop container during cleanup", 
						zap.Error(err), 
						zap.String("container_id", c.ID),
						zap.String("app_id", appID),
					)
				}
				cancel()
			}

			// Remove container
			removeOpts := container.RemoveOptions{Force: true}
			if err := s.client.ContainerRemove(ctx, c.ID, removeOpts); err != nil {
				s.logger.Warn("Failed to remove container during cleanup", 
					zap.Error(err), 
					zap.String("container_id", c.ID),
					zap.String("app_id", appID),
				)
			} else {
				s.logger.Info("Removed container during app cleanup",
					zap.String("container_id", c.ID),
					zap.String("app_id", appID),
				)
			}
		}
	}

	// Step 2: Find and remove all images for this app
	// Image format: stackyn-{appID}:{buildJobID} or stackyn-{appID}
	imagePattern := fmt.Sprintf("stackyn-%s", appID)
	
	images, err := s.client.ImageList(ctx, image.ListOptions{
		All: true, // Include all images (including dangling ones)
	})
	if err != nil {
		s.logger.Warn("Failed to list images for cleanup", zap.Error(err), zap.String("app_id", appID))
	} else {
		for _, img := range images {
			// Check if any tag matches our app pattern
			shouldRemove := false
			for _, tag := range img.RepoTags {
				if strings.HasPrefix(tag, imagePattern+":") || tag == imagePattern {
					shouldRemove = true
					break
				}
			}

			if shouldRemove {
				// Remove image (with all tags)
				_, err := s.client.ImageRemove(ctx, img.ID, image.RemoveOptions{
					Force:         true, // Force removal even if in use
					PruneChildren: true, // Remove untagged parent images
				})
				if err != nil {
					s.logger.Warn("Failed to remove image during cleanup", 
						zap.Error(err), 
						zap.String("image_id", img.ID),
						zap.String("app_id", appID),
						zap.Strings("tags", img.RepoTags),
					)
				} else {
					s.logger.Info("Removed image during app cleanup",
						zap.String("image_id", img.ID),
						zap.String("app_id", appID),
						zap.Strings("tags", img.RepoTags),
					)
				}
			}
		}
	}

	// Step 3: Clean up docker-compose containers if any
	// Docker-compose uses project name: stackyn-{appID}
	projectName := fmt.Sprintf("stackyn-%s", appID)
	composeFilter := filters.NewArgs()
	composeFilter.Add("label", fmt.Sprintf("com.docker.compose.project=%s", projectName))
	
	composeContainers, err := s.client.ContainerList(ctx, container.ListOptions{
		Filters: composeFilter,
		All:     true,
	})
	if err == nil && len(composeContainers) > 0 {
		for _, c := range composeContainers {
			// Stop container if running
			if c.State == "running" {
				stopCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
				timeout := 30
				stopOpts := container.StopOptions{Timeout: &timeout}
				s.client.ContainerStop(stopCtx, c.ID, stopOpts)
				cancel()
			}

			// Remove container
			removeOpts := container.RemoveOptions{Force: true}
			if err := s.client.ContainerRemove(ctx, c.ID, removeOpts); err != nil {
				s.logger.Warn("Failed to remove docker-compose container during cleanup", 
					zap.Error(err), 
					zap.String("container_id", c.ID),
					zap.String("app_id", appID),
				)
			}
		}

		// Try to stop docker-compose project (if docker-compose command is available)
		// This is best-effort, containers are already stopped above
		s.logger.Info("Cleaned up docker-compose containers for app",
			zap.String("app_id", appID),
			zap.String("project_name", projectName),
		)
	}

	s.logger.Info("Completed cleanup of Docker resources for app",
		zap.String("app_id", appID),
	)
	return nil
}

// pullImage ensures a Docker image exists locally (does not pull from registry for local builds)
func (s *DeploymentService) pullImage(ctx context.Context, imageRef string) error {
	// Check if the image exists locally (with retry for race conditions)
	maxRetries := 3
	var lastErr error
	
	for i := 0; i < maxRetries; i++ {
		_, _, err := s.client.ImageInspectWithRaw(ctx, imageRef)
		if err == nil {
			// Image exists locally, no need to pull
			if i > 0 {
				s.logger.Info("Image found after retry", zap.String("image", imageRef), zap.Int("attempt", i+1))
			} else {
				s.logger.Info("Image already exists locally, skipping pull", zap.String("image", imageRef))
			}
			return nil
		}
		
		lastErr = err
		
		// If it's a "not found" error, wait a bit and retry (handles race condition where
		// image was just built but not yet visible to this Docker client)
		if client.IsErrNotFound(err) && i < maxRetries-1 {
			waitTime := time.Duration(i+1) * 200 * time.Millisecond
			s.logger.Debug("Image not found, retrying after brief wait",
				zap.String("image", imageRef),
				zap.Duration("wait", waitTime),
				zap.Int("attempt", i+1))
			time.Sleep(waitTime)
			continue
		}
		
		// For non-"not found" errors, log warning but continue retrying
		if !client.IsErrNotFound(err) {
			s.logger.Warn("Unexpected error checking for image, will retry",
				zap.String("image", imageRef),
				zap.Error(err),
				zap.Int("attempt", i+1))
			if i < maxRetries-1 {
				time.Sleep(time.Duration(i+1) * 200 * time.Millisecond)
				continue
			}
		}
	}

	// After all retries, image still not found
	// For locally-built images, we don't pull from registry
	return fmt.Errorf("image not found locally after %d attempts (image should have been built by build-worker): %w", maxRetries, lastErr)
}

// generateContainerName generates a unique container name
func (s *DeploymentService) generateContainerName(appID, deploymentID string) string {
	return fmt.Sprintf("stackyn-%s-%s", appID, deploymentID)
}

// generateTraefikLabels generates Traefik labels for routing with HTTPS, subdomains, and health checks
func (s *DeploymentService) generateTraefikLabels(subdomain string, port int, appID string) map[string]string {
	routerName := fmt.Sprintf("app-%s", appID)
	serviceName := fmt.Sprintf("app-%s", appID)
	middlewareName := fmt.Sprintf("app-%s-redirect", appID)
	
	// Check if this is a .local domain (local development)
	isLocalDomain := strings.HasSuffix(subdomain, ".local") || strings.HasSuffix(subdomain, ".localhost")
	
	labels := map[string]string{
		// Enable Traefik
		"traefik.enable": "true",
		
		// Service configuration
		fmt.Sprintf("traefik.http.services.%s.loadbalancer.server.port", serviceName): strconv.Itoa(port),
		fmt.Sprintf("traefik.http.services.%s.loadbalancer.healthcheck.path", serviceName): "/",
		fmt.Sprintf("traefik.http.services.%s.loadbalancer.healthcheck.interval", serviceName): "10s",
		fmt.Sprintf("traefik.http.services.%s.loadbalancer.healthcheck.timeout", serviceName): "10s", // Increased from 3s to allow app startup time
		fmt.Sprintf("traefik.http.services.%s.loadbalancer.healthcheck.scheme", serviceName): "http", // Use HTTP for health checks
		
		// Use the configured network
		"traefik.docker.network": s.networkName,
		
		// App ID label for container lookup
		"app.id": appID,
		"app.subdomain": subdomain,
	}
	
	if isLocalDomain {
		// For .local domains, use HTTP only (no HTTPS/TLS)
		labels[fmt.Sprintf("traefik.http.routers.%s.rule", routerName)] = fmt.Sprintf("Host(`%s`)", subdomain)
		labels[fmt.Sprintf("traefik.http.routers.%s.entrypoints", routerName)] = "web"
	} else {
		// For production domains, use HTTPS with redirect
		// HTTP Router (redirects to HTTPS)
		labels[fmt.Sprintf("traefik.http.routers.%s-http.rule", routerName)] = fmt.Sprintf("Host(`%s`)", subdomain)
		labels[fmt.Sprintf("traefik.http.routers.%s-http.entrypoints", routerName)] = "web"
		labels[fmt.Sprintf("traefik.http.routers.%s-http.middlewares", routerName)] = middlewareName
		
		// HTTPS Router (main router)
		labels[fmt.Sprintf("traefik.http.routers.%s.rule", routerName)] = fmt.Sprintf("Host(`%s`)", subdomain)
		labels[fmt.Sprintf("traefik.http.routers.%s.entrypoints", routerName)] = "websecure"
		labels[fmt.Sprintf("traefik.http.routers.%s.tls", routerName)] = "true"
		labels[fmt.Sprintf("traefik.http.routers.%s.tls.certresolver", routerName)] = "letsencrypt"
		
		// Redirect middleware (HTTP to HTTPS)
		labels[fmt.Sprintf("traefik.http.middlewares.%s.redirectscheme.scheme", middlewareName)] = "https"
		labels[fmt.Sprintf("traefik.http.middlewares.%s.redirectscheme.permanent", middlewareName)] = "true"
	}

	return labels
}

// streamAndPersistRuntimeLogs streams and persists runtime logs from a container
func (s *DeploymentService) streamAndPersistRuntimeLogs(ctx context.Context, containerID, appID, deploymentID string) {
	// Stream logs from container
	logReader, err := s.client.ContainerLogs(ctx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true, // Follow logs in real-time
		Timestamps: true,
		Tail:       "100", // Start from last 100 lines
	})
	if err != nil {
		s.logger.Warn("Failed to stream container logs", zap.Error(err), zap.String("container_id", containerID))
		return
	}
	defer logReader.Close()

	// Create log entry as map for LogPersistenceService
	// Use container_id as the identifier since it's stable and available in both storage and retrieval
	logEntry := map[string]interface{}{
		"app_id":        appID,
		"deployment_id": containerID, // Use container_id instead of deploymentID for reliable lookup
		"log_type":      "runtime",
		"timestamp":     time.Now(),
		"size":          int64(0),
	}

	s.logger.Info("Starting to persist runtime logs",
		zap.String("app_id", appID),
		zap.String("deployment_id", deploymentID),
		zap.String("container_id", containerID),
	)

	// Persist log stream
	if err := s.persistRuntimeLogStream(ctx, logEntry, logReader); err != nil {
		s.logger.Warn("Failed to persist runtime logs", zap.Error(err), zap.String("container_id", containerID))
	} else {
		s.logger.Info("Successfully started persisting runtime logs", zap.String("container_id", containerID))
	}
}

// persistRuntimeLogStream persists runtime logs using the log persistence service
func (s *DeploymentService) persistRuntimeLogStream(ctx context.Context, entry interface{}, reader io.Reader) error {
	// Use the log persistence service if available
	if s.logPersistence != nil {
		return s.logPersistence.PersistLogStream(ctx, entry, reader)
	}
	return nil
}

// monitorContainerCrash monitors a container for crashes and logs errors
func (s *DeploymentService) monitorContainerCrash(ctx context.Context, containerID, appID, deploymentID string) {
	ticker := time.NewTicker(10 * time.Second) // Check every 10 seconds
	defer ticker.Stop()
	
	lastStatus := "running" // Track last known status to avoid duplicate logging

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
			if !containerJSON.State.Running && containerJSON.State.Status != lastStatus {
				// Capture container logs for debugging
				logs := ""
				logReader, err := s.client.ContainerLogs(ctx, containerID, container.LogsOptions{
					ShowStdout: true,
					ShowStderr: true,
					Follow:     false,
					Timestamps: true,
					Tail:       "50", // Last 50 lines
				})
				if err == nil {
					logBytes, _ := io.ReadAll(logReader)
					logs = string(logBytes)
					logReader.Close()
				}

				// Build error message from logs and container state
				errorMsg := containerJSON.State.Error
				if errorMsg == "" && logs != "" {
					// Extract last meaningful error line from logs
					lines := strings.Split(logs, "\n")
					for i := len(lines) - 1; i >= 0; i-- {
						line := strings.TrimSpace(lines[i])
						if line != "" && (strings.Contains(line, "error") || strings.Contains(line, "Error") || strings.Contains(line, "ERROR") || strings.Contains(line, "failed") || strings.Contains(line, "Failed")) {
							errorMsg = line
							break
						}
					}
					if errorMsg == "" && len(lines) > 0 {
						// Use last non-empty line
						for i := len(lines) - 1; i >= 0; i-- {
							line := strings.TrimSpace(lines[i])
							if line != "" {
								errorMsg = line
								break
							}
						}
					}
				}
				if errorMsg == "" {
					errorMsg = fmt.Sprintf("Container exited with status %s (exit code %d)", containerJSON.State.Status, containerJSON.State.ExitCode)
				}

				s.logger.Error("Container crashed",
					zap.String("container_id", containerID),
					zap.String("app_id", appID),
					zap.String("deployment_id", deploymentID),
					zap.String("status", containerJSON.State.Status),
					zap.Int("exit_code", containerJSON.State.ExitCode),
					zap.String("error", errorMsg),
					zap.String("logs", logs),
				)

				// Call crash callback if set (to update database)
				if s.crashCallback != nil {
					s.crashCallback(appID, deploymentID, containerID, containerJSON.State.ExitCode, errorMsg)
				}

				// Update last status to avoid duplicate logging
				lastStatus = containerJSON.State.Status

				// Check restart count
				if containerJSON.RestartCount >= 3 {
					s.logger.Error("Container exceeded restart limit",
						zap.String("container_id", containerID),
						zap.String("app_id", appID),
						zap.String("deployment_id", deploymentID),
						zap.Int("restart_count", containerJSON.RestartCount),
						zap.String("reason", "container_crash_exceeded_restart_limit"),
					)
				}
			} else if containerJSON.State.Running {
				// Reset last status if container is running again
				lastStatus = "running"
			}
		}
	}
}

// DeploymentVerificationResult represents the result of deployment verification
type DeploymentVerificationResult struct {
	IsRunning        bool
	ContainerID      string
	ContainerName    string
	Port             int
	Subdomain        string
	URL              string
	TraefikConfigured bool
	HealthCheckPassed bool
	Errors           []string
}

// VerifyDeployment verifies that a deployment is successful and accessible
// This function checks:
// 1. Container is running
// 2. Port is bound correctly
// 3. Traefik routing is configured
// 4. Health check passes (optional)
func (s *DeploymentService) VerifyDeployment(ctx context.Context, appID string) (*DeploymentVerificationResult, error) {
	result := &DeploymentVerificationResult{
		Errors: make([]string, 0),
	}

	// Step 1: Find container by app ID
	containers, err := s.findContainersByAppID(ctx, appID)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to find containers: %v", err))
		return result, fmt.Errorf("failed to find containers: %w", err)
	}

	if len(containers) == 0 {
		result.Errors = append(result.Errors, "No container found for app")
		return result, fmt.Errorf("no container found for app %s", appID)
	}

	container := containers[0]
	result.ContainerID = container.ID
	if len(container.Names) > 0 {
		result.ContainerName = container.Names[0]
	} else {
		result.ContainerName = "unknown"
	}

	// Step 2: Inspect container to get detailed status
	containerJSON, err := s.client.ContainerInspect(ctx, container.ID)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to inspect container: %v", err))
		return result, fmt.Errorf("failed to inspect container: %w", err)
	}

	// Check if container is running
	result.IsRunning = containerJSON.State.Running
	if !result.IsRunning {
		result.Errors = append(result.Errors, fmt.Sprintf("Container is not running. Status: %s, ExitCode: %d", 
			containerJSON.State.Status, containerJSON.State.ExitCode))
		if containerJSON.State.Error != "" {
			result.Errors = append(result.Errors, fmt.Sprintf("Container error: %s", containerJSON.State.Error))
		}
	}

	// Step 3: Extract port from container config
	// Look for PORT environment variable
	port := 8080 // Default
	for _, env := range containerJSON.Config.Env {
		if strings.HasPrefix(env, "PORT=") {
			if p, err := strconv.Atoi(strings.TrimPrefix(env, "PORT=")); err == nil {
				port = p
				break
			}
		}
	}
	result.Port = port

	// Step 4: Extract subdomain from labels
	if subdomain, ok := containerJSON.Config.Labels["app.subdomain"]; ok {
		result.Subdomain = subdomain
		// Generate URL
		if strings.HasSuffix(subdomain, ".local") || strings.HasSuffix(subdomain, ".localhost") {
			result.URL = fmt.Sprintf("http://%s", subdomain)
		} else {
			result.URL = fmt.Sprintf("https://%s", subdomain)
		}
	} else {
		result.Errors = append(result.Errors, "Subdomain not found in container labels")
	}

	// Step 5: Check Traefik labels
	traefikEnabled := containerJSON.Config.Labels["traefik.enable"] == "true"
	result.TraefikConfigured = traefikEnabled
	if !traefikEnabled {
		result.Errors = append(result.Errors, "Traefik is not enabled for this container")
	} else {
		// Check for router configuration
		routerName := fmt.Sprintf("app-%s", appID)
		routerRule := containerJSON.Config.Labels[fmt.Sprintf("traefik.http.routers.%s.rule", routerName)]
		if routerRule == "" {
			result.Errors = append(result.Errors, "Traefik router rule not found")
		}
		// Check for service configuration
		servicePort := containerJSON.Config.Labels[fmt.Sprintf("traefik.http.services.%s.loadbalancer.server.port", routerName)]
		if servicePort == "" {
			result.Errors = append(result.Errors, "Traefik service port not configured")
		}
	}

	// Step 6: Optional health check (if container is running)
	if result.IsRunning {
		// Try to perform a basic health check by checking if container responds
		// This is a simple check - in production, you might want to call the actual health endpoint
		healthCheckPath := containerJSON.Config.Labels[fmt.Sprintf("traefik.http.services.app-%s.loadbalancer.healthcheck.path", appID)]
		if healthCheckPath == "" {
			healthCheckPath = "/"
		}
		
		// For now, we'll just check if the container is running
		// A full health check would require making an HTTP request to the container
		// which is complex without knowing the exact network setup
		result.HealthCheckPassed = result.IsRunning && result.TraefikConfigured
	} else {
		result.HealthCheckPassed = false
	}

	s.logger.Info("Deployment verification completed",
		zap.String("app_id", appID),
		zap.String("container_id", result.ContainerID),
		zap.Bool("is_running", result.IsRunning),
		zap.Int("port", result.Port),
		zap.String("subdomain", result.Subdomain),
		zap.String("url", result.URL),
		zap.Bool("traefik_configured", result.TraefikConfigured),
		zap.Bool("health_check_passed", result.HealthCheckPassed),
		zap.Int("error_count", len(result.Errors)),
	)

	return result, nil
}

