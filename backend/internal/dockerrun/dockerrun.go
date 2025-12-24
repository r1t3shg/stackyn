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
//
// Returns:
//   - containerID: Docker container ID on success
//   - error: Detailed error if container creation/start fails
func (r *Runner) Run(ctx context.Context, imageName, subdomain, baseDomain string, appID, deploymentID int, internalPort int) (string, error) {
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

	// Create container config
	containerConfig := &container.Config{
		Image:  imageName,
		Labels: labels,
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
	time.Sleep(2 * time.Second)
	
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
		
		// Check container logs to verify the app is listening on the expected port
		// This helps catch cases where port detection failed
		logsReader, logsErr := r.client.ContainerLogs(ctx, resp.ID, container.LogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Tail:       "20", // Just check last 20 lines for port info
		})
		if logsErr == nil {
			defer logsReader.Close()
			logsData, _ := io.ReadAll(logsReader)
			if len(logsData) > 0 {
				logsStr := string(logsData)
				log.Printf("[DOCKER] Container startup logs (last 20 lines): %s", logsStr)
				// Log a warning if we see common port patterns that don't match our detected port
				// This helps debug port mismatches
				if internalPort == 8080 {
					// Check for common alternative ports in logs
					if containsPort(logsStr, "8000") {
						log.Printf("[DOCKER] WARNING - Container logs mention port 8000, but Traefik is configured for 8080. This may cause 502 errors.")
					} else if containsPort(logsStr, "5000") {
						log.Printf("[DOCKER] WARNING - Container logs mention port 5000, but Traefik is configured for 8080. This may cause 502 errors.")
					}
				}
			}
		}
	}
	
	return resp.ID, nil
}

// containsPort checks if a string contains common port patterns
func containsPort(text, port string) bool {
	// Check for common patterns: "port 8000", ":8000", "8000", "listening on 8000", etc.
	patterns := []string{
		"port " + port,
		":" + port,
		" " + port + " ",
		" " + port + "\n",
		"listening on " + port,
		"running on " + port,
		"bound to " + port,
	}
	lowerText := strings.ToLower(text)
	for _, pattern := range patterns {
		if strings.Contains(lowerText, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}

func (r *Runner) Stop(ctx context.Context, containerID string) error {
	return r.client.ContainerStop(ctx, containerID, container.StopOptions{})
}

func (r *Runner) Remove(ctx context.Context, containerID string) error {
	return r.client.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true})
}
