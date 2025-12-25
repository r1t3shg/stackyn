// Package dockerrun provides functionality for running Docker containers.
// It handles container creation, startup, and Traefik label configuration
// for automatic routing. Containers are configured with:
//   - Traefik labels for automatic service discovery
//   - SSL/TLS support via Let's Encrypt
//   - Network configuration for Traefik routing
//   - Port mapping and health checks
//   - Resource limits (memory, CPU, disk, process limits)
package dockerrun

import (
	"context"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)

type Runner struct {
	client *client.Client
}

func NewRunner(dockerHost string) (*Runner, error) {
	log.Printf("[DOCKER] Initializing Docker runner - Host: %s", dockerHost)
	cli, err := client.NewClientWithOpts(
		client.WithHost(dockerHost),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		log.Printf("[DOCKER] ERROR - Failed to create Docker client: %v", err)
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	log.Printf("[DOCKER] Docker runner initialized successfully")
	return &Runner{client: cli}, nil
}

// Run starts a Docker container with enforced resource limits and Traefik routing.
// It applies hard limits for memory, CPU, process count, and logging.
//
// Resource Limits Applied:
//   - Memory: 256 MB (hard limit, no swap)
//   - CPU: 0.25 vCPU (250000000 nano CPUs)
//   - Process limit: 128 PIDs
//   - Logging: JSON file driver with 10MB max size, 3 file rotation
//   - Restart policy: unless-stopped
//
// Disk Limit Strategy:
//   - MVP: Uses Docker's default volume management without explicit size limits
//   - TODO: Implement filesystem quota enforcement (requires host-level quota support)
//     Options: XFS project quotas, btrfs quotas, or Docker volume size limits
//
// Parameters:
//   - ctx: Context for cancellation/timeout
//   - imageName: Docker image name (already built)
//   - subdomain: Subdomain for Traefik routing
//   - baseDomain: Base domain for FQDN construction
//   - appID: Application ID for container naming
//   - deploymentID: Deployment ID for container naming
//   - internalPort: Port the application listens on inside the container
//   - envVars: Map of environment variables to inject into the container (key-value pairs)
//
// Returns:
//   - containerID: Docker container ID on success
//   - error: Detailed error if container creation/start fails
func (r *Runner) Run(ctx context.Context, imageName, subdomain, baseDomain string, appID, deploymentID int, internalPort int, envVars map[string]string) (string, error) {
	// Build FQDN and determine router/service names
	fqdn := fmt.Sprintf("%s.%s", subdomain, baseDomain)
	routerName := subdomain
	serviceName := subdomain
	// Container name format: app-<appID>-<deploymentID>
	containerName := fmt.Sprintf("app-%d-%d", appID, deploymentID)

	log.Printf("[DOCKER] Running container - Image: %s, Subdomain: %s, FQDN: %s, Name: %s", imageName, subdomain, fqdn, containerName)

	// Create Traefik labels with HTTPS/TLS support
	labels := map[string]string{
		"traefik.enable": "true",
		"traefik.docker.network": "stackyn-network",
		// HTTPS Router
		"traefik.http.routers." + routerName + ".rule":                       fmt.Sprintf("Host(`%s`)", fqdn),
		"traefik.http.routers." + routerName + ".entrypoints":                "websecure",
		"traefik.http.routers." + routerName + ".tls":                        "true",
		"traefik.http.routers." + routerName + ".tls.certresolver":           "letsencrypt",
		"traefik.http.routers." + routerName + ".service":                    serviceName,
		// HTTP Router (redirects to HTTPS using inline redirect middleware)
		"traefik.http.routers." + routerName + "-redirect.rule":              fmt.Sprintf("Host(`%s`)", fqdn),
		"traefik.http.routers." + routerName + "-redirect.entrypoints":       "web",
		"traefik.http.routers." + routerName + "-redirect.middlewares":       routerName + "-redirect",
		// Redirect middleware (inline)
		"traefik.http.middlewares." + routerName + "-redirect.redirectscheme.scheme": "https",
		"traefik.http.middlewares." + routerName + "-redirect.redirectscheme.permanent": "true",
		// Service definition
		"traefik.http.services." + serviceName + ".loadbalancer.server.port": strconv.Itoa(internalPort),
	}

	// Convert environment variables map to slice of strings in KEY=VALUE format
	envSlice := make([]string, 0, len(envVars))
	for key, value := range envVars {
		envSlice = append(envSlice, fmt.Sprintf("%s=%s", key, value))
	}
	log.Printf("[DOCKER] Injecting %d environment variable(s) into container", len(envSlice))

	// Create container config
	containerConfig := &container.Config{
		Image:      imageName,
		Labels:     labels,
		Env:        envSlice,
	}

	// Resource limits constants
	// Memory limit: 256 MB (256 * 1024 * 1024 bytes)
	// This is a hard limit - container cannot exceed this memory usage
	memoryLimitBytes := int64(256 * 1024 * 1024)
	// Memory swap: 256 MB (same as memory limit to disable swap)
	// Setting swap equal to memory effectively disables swap usage
	memorySwapBytes := int64(256 * 1024 * 1024)
	// CPU limit: 0.25 vCPU
	// Using CPU quota and period: quota = 25000, period = 100000
	// This gives us 0.25 vCPU (25000/100000 = 0.25)
	cpuQuota := int64(25000)  // 25% of CPU
	cpuPeriod := int64(100000) // Standard period
	// Process limit: 128 PIDs
	// Prevents fork bombs and excessive process creation
	pidsLimit := int64(128)
	pidsLimitPtr := &pidsLimit

	// Create host config with resource limits
	hostConfig := &container.HostConfig{
		AutoRemove: false,
		// Restart policy: unless-stopped
		// Container will automatically restart on failure, unless manually stopped
		RestartPolicy: container.RestartPolicy{
			Name: "unless-stopped",
		},
		// Resource limits: Memory, CPU, and process limits
		Resources: container.Resources{
			// Memory limit: Hard limit of 256 MB
			// Container will be OOM killed if it exceeds this limit
			Memory: memoryLimitBytes,
			// Memory swap: Set to same as memory to disable swap
			// This ensures total memory usage (RAM + swap) cannot exceed 256 MB
			MemorySwap: memorySwapBytes,
			// CPU limit: 0.25 vCPU using quota/period
			// Container can use at most 25% of one CPU core
			CPUQuota: cpuQuota,
			CPUPeriod: cpuPeriod,
			// Process limit: Maximum 128 processes
			// Prevents fork bombs and resource exhaustion attacks
			PidsLimit: pidsLimitPtr,
		},
		// Logging configuration: JSON file driver with rotation
		// Prevents log files from consuming unlimited disk space
		LogConfig: container.LogConfig{
			Type: "json-file",
			Config: map[string]string{
				// Maximum size per log file: 10 MB
				// When a log file reaches this size, it rotates
				"max-size": "10m",
				// Maximum number of log files to keep: 3
				// Total log storage per container: ~30 MB (3 files * 10 MB)
				"max-file": "3",
			},
		},
	}

	// Disk limit enforcement strategy:
	// MVP: Docker volumes are created without explicit size limits.
	// The container's writable layer and volumes are managed by Docker's storage driver.
	// TODO: Implement filesystem quota enforcement for 1 GB disk limit per app.
	// Options for future implementation:
	//   1. XFS project quotas: Set quota on per-app volume directories
	//   2. btrfs quotas: Use btrfs subvolume quotas if using btrfs storage driver
	//   3. Docker volume size limits: Use volume plugins that support size limits
	//   4. Periodic cleanup: Monitor disk usage and clean up old data
	// Implementation would require:
	//   - Host filesystem support for quotas (XFS or btrfs)
	//   - Volume creation with quota settings
	//   - Monitoring and enforcement logic

	// Create network config to connect to stackyn-network
	networkConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			"stackyn-network": {},
		},
	}

	// Create container
	log.Printf("[DOCKER] Creating container: %s (Memory: 256MB, CPU: 0.25, PIDs: 128)", containerName)
	resp, err := r.client.ContainerCreate(ctx, containerConfig, hostConfig, networkConfig, nil, containerName)
	if err != nil {
		// Capture Docker error details for debugging
		errorDetails := err.Error()
		log.Printf("[DOCKER] ERROR - Failed to create container: %s", errorDetails)
		
		return "", fmt.Errorf("failed to create container: %w", err)
	}
	log.Printf("[DOCKER] Container created - ID: %s", resp.ID)

	// Start container
	log.Printf("[DOCKER] Starting container: %s", resp.ID)
	if err := r.client.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		// Capture Docker error details
		errorDetails := err.Error()
		log.Printf("[DOCKER] ERROR - Failed to start container: %s", errorDetails)
		
		// Try to get container logs for additional context
		logsReader, logsErr := r.client.ContainerLogs(ctx, resp.ID, container.LogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Tail:       "50",
		})
		if logsErr == nil {
			defer logsReader.Close()
			logsData, _ := io.ReadAll(logsReader)
			if len(logsData) > 0 {
				log.Printf("[DOCKER] Container logs (last 50 lines): %s", string(logsData))
				errorDetails = fmt.Sprintf("%s\nContainer logs: %s", errorDetails, string(logsData))
			}
		}
		
		// Clean up the failed container
		removeErr := r.client.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true})
		if removeErr != nil {
			log.Printf("[DOCKER] WARNING - Failed to remove failed container %s: %v", resp.ID, removeErr)
		}
		
		return "", fmt.Errorf("failed to start container: %w", err)
	}

	log.Printf("[DOCKER] Container started successfully - ID: %s, Name: %s, URL: https://%s", resp.ID, containerName, fqdn)
	
	// Wait a moment for the container to initialize
	// Then check if it's still running (basic health check)
	time.Sleep(3 * time.Second)
	
	// Check container status
	containerInfo, err := r.client.ContainerInspect(ctx, resp.ID)
	if err != nil {
		log.Printf("[DOCKER] WARNING - Failed to inspect container: %v", err)
	} else {
		if !containerInfo.State.Running {
			// Container stopped - get logs for debugging
			logsReader, logsErr := r.client.ContainerLogs(ctx, resp.ID, container.LogsOptions{
				ShowStdout: true,
				ShowStderr: true,
				Tail:       "100",
			})
			if logsErr == nil {
				defer logsReader.Close()
				logsData, _ := io.ReadAll(logsReader)
				if len(logsData) > 0 {
					log.Printf("[DOCKER] Container stopped after startup. Logs: %s", string(logsData))
					return "", fmt.Errorf("container stopped after startup. Exit code: %d. Logs: %s", 
						containerInfo.State.ExitCode, string(logsData))
				}
			}
			return "", fmt.Errorf("container stopped after startup. Exit code: %d", containerInfo.State.ExitCode)
		}
		log.Printf("[DOCKER] Container health check passed - Status: %s", containerInfo.State.Status)
		
		// Check container logs to verify it's a web server, not a worker
		logsReader, logsErr := r.client.ContainerLogs(ctx, resp.ID, container.LogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Tail:       "30",
		})
		if logsErr == nil {
			defer logsReader.Close()
			logsData, _ := io.ReadAll(logsReader)
			if len(logsData) > 0 {
				logsStr := string(logsData)
				// Check if this is a worker app based on logs
				if isWorkerAppFromLogs(logsStr) {
					return "", fmt.Errorf("worker apps are not supported yet. Stackyn currently supports only HTTP-based applications that expose a port and serve web requests. Your app does not appear to start a web server. What you can do: • Deploy an API or web app that listens on a port • Wait for background worker support (coming soon)")
				}
			}
		}
	}
	
	return resp.ID, nil
}

// isWorkerAppFromLogs checks if container logs indicate this is a worker/background process
// This is a fallback check - primary detection happens in gitrepo.IsWorkerApp
// Returns true only if logs clearly indicate a worker with no web server indicators
func isWorkerAppFromLogs(logs string) bool {
	lowerLogs := strings.ToLower(logs)
	
	// First, check for positive web server indicators
	// If we find these, it's definitely NOT a worker
	webServerPatterns := []string{
		"listening on",
		"running on http",
		"serving on",
		"bound to",
		"uvicorn running",
		"gunicorn",
		"http server",
		"web server",
		"started server",
		"server listening",
		"server started",
		"listening on port",
		"listening at",
		"ready to accept connections",
	}
	
	hasWebServer := false
	for _, pattern := range webServerPatterns {
		if strings.Contains(lowerLogs, pattern) {
			hasWebServer = true
			log.Printf("[DOCKER] Found web server indicator '%s' in logs - not a worker", pattern)
			break
		}
	}
	
	// If we found web server indicators, it's NOT a worker
	if hasWebServer {
		return false
	}
	
	// Only check for worker patterns if no web server indicators found
	// Use more specific patterns to avoid false positives
	workerPatterns := []string{
		"celery worker",
		"celery@",
		"sidekiq",
		"bull queue",
		"queue:work",
		"queue:listen",
		"worker:start",
		"background worker started",
		"worker process started",
	}
	
	// Check for specific worker indicators
	for _, pattern := range workerPatterns {
		if strings.Contains(lowerLogs, pattern) {
			log.Printf("[DOCKER] Detected worker pattern '%s' in container logs", pattern)
			return true
		}
	}
	
	return false
}

func (r *Runner) Stop(ctx context.Context, containerID string) error {
	log.Printf("[DOCKER] Stopping container: %s", containerID)
	
	// First, try to inspect the container to check its status
	inspectCtx, cancelInspect := context.WithTimeout(ctx, 10*time.Second)
	containerInfo, inspectErr := r.client.ContainerInspect(inspectCtx, containerID)
	cancelInspect()
	
	if inspectErr != nil {
		// Container might not exist
		if strings.Contains(inspectErr.Error(), "No such container") {
			log.Printf("[DOCKER] Container %s does not exist, skipping stop", containerID)
			return nil
		}
		log.Printf("[DOCKER] WARNING - Failed to inspect container %s: %v (will try to stop anyway)", containerID, inspectErr)
	} else {
		// Check if container is already stopped
		if !containerInfo.State.Running {
			log.Printf("[DOCKER] Container %s is already stopped (Status: %s)", containerID, containerInfo.State.Status)
			return nil
		}
		log.Printf("[DOCKER] Container %s is running, stopping it...", containerID)
	}
	
	// Use a timeout of 30 seconds for stopping the container
	stopCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	
	// Stop the container with a 10 second timeout (in seconds)
	timeout := 10
	err := r.client.ContainerStop(stopCtx, containerID, container.StopOptions{
		Timeout: &timeout,
	})
	if err != nil {
		// Check if container doesn't exist or is already stopped
		errStr := err.Error()
		if strings.Contains(errStr, "No such container") || 
		   strings.Contains(errStr, "is not running") ||
		   strings.Contains(errStr, "already stopped") ||
		   strings.Contains(errStr, "not found") {
			log.Printf("[DOCKER] Container %s is already stopped or doesn't exist: %v", containerID, err)
			return nil // Not an error if already stopped
		}
		log.Printf("[DOCKER] ERROR - Failed to stop container %s: %v", containerID, err)
		return fmt.Errorf("failed to stop container %s: %w", containerID, err)
	}
	
	log.Printf("[DOCKER] Container stopped successfully: %s", containerID)
	return nil
}

func (r *Runner) Remove(ctx context.Context, containerID string) error {
	log.Printf("[DOCKER] Removing container: %s", containerID)
	
	// First, try to inspect the container to check if it exists
	inspectCtx, cancelInspect := context.WithTimeout(ctx, 10*time.Second)
	_, inspectErr := r.client.ContainerInspect(inspectCtx, containerID)
	cancelInspect()
	
	if inspectErr != nil {
		// Container doesn't exist
		if strings.Contains(inspectErr.Error(), "No such container") || 
		   strings.Contains(inspectErr.Error(), "not found") {
			log.Printf("[DOCKER] Container %s does not exist, skipping remove", containerID)
			return nil
		}
		log.Printf("[DOCKER] WARNING - Failed to inspect container %s: %v (will try to remove anyway)", containerID, inspectErr)
	}
	
	// Use a timeout of 30 seconds for removing the container
	removeCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	
	err := r.client.ContainerRemove(removeCtx, containerID, container.RemoveOptions{
		Force:         true, // Force removal even if running
		RemoveVolumes: true, // Also remove volumes
	})
	if err != nil {
		// Check if container doesn't exist
		errStr := err.Error()
		if strings.Contains(errStr, "No such container") || 
		   strings.Contains(errStr, "not found") {
			log.Printf("[DOCKER] Container %s doesn't exist (may already be removed): %v", containerID, err)
			return nil // Not an error if already removed
		}
		log.Printf("[DOCKER] ERROR - Failed to remove container %s: %v", containerID, err)
		return fmt.Errorf("failed to remove container %s: %w", containerID, err)
	}
	
	log.Printf("[DOCKER] Container removed successfully: %s", containerID)
	return nil
}

// RemoveImage removes a Docker image by name
func (r *Runner) RemoveImage(ctx context.Context, imageName string) error {
	log.Printf("[DOCKER] Removing image: %s", imageName)
	
	// Use a timeout of 60 seconds for removing the image
	removeCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	
	_, err := r.client.ImageRemove(removeCtx, imageName, image.RemoveOptions{
		Force:         true, // Force removal even if in use
		PruneChildren: true, // Remove all untagged parents
	})
	if err != nil {
		// Check if image doesn't exist
		if strings.Contains(err.Error(), "No such image") || 
		   strings.Contains(err.Error(), "image not known") {
			log.Printf("[DOCKER] Image %s doesn't exist (may already be removed): %v", imageName, err)
			return nil // Not an error if already removed
		}
		log.Printf("[DOCKER] ERROR - Failed to remove image %s: %v", imageName, err)
		return fmt.Errorf("failed to remove image %s: %w", imageName, err)
	}
	log.Printf("[DOCKER] Image removed successfully: %s", imageName)
	return nil
}

// GetLogs retrieves the runtime logs from a Docker container.
// It returns an io.ReadCloser that contains the container's stdout and stderr logs.
// The caller is responsible for closing the reader.
//
// Parameters:
//   - ctx: Context for cancellation/timeout
//   - containerID: The Docker container ID to get logs from
//   - tail: Number of lines to fetch from the end (e.g., "100" for last 100 lines, "all" for all logs)
//
// Returns:
//   - io.ReadCloser: A reader containing the container logs (must be closed by caller)
//   - error: Error if log retrieval fails (container doesn't exist, etc.)
func (r *Runner) GetLogs(ctx context.Context, containerID string, tail string) (io.ReadCloser, error) {
	log.Printf("[DOCKER] Fetching logs for container: %s (tail: %s)", containerID, tail)
	
	// Use a timeout of 30 seconds for fetching logs
	logsCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	
	// Fetch container logs
	logsReader, err := r.client.ContainerLogs(logsCtx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       tail,
		Follow:     false, // Don't follow, just get current logs
	})
	if err != nil {
		log.Printf("[DOCKER] ERROR - Failed to fetch logs for container %s: %v", containerID, err)
		return nil, fmt.Errorf("failed to fetch container logs: %w", err)
	}
	
	log.Printf("[DOCKER] Logs fetched successfully for container: %s", containerID)
	return logsReader, nil
}