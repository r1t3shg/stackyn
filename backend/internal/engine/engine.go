// Package engine provides the core deployment orchestration logic.
// The Engine coordinates the entire deployment pipeline:
//   1. Git repository cloning
//   2. Docker image building
//   3. Container creation and startup
//   4. Traefik routing configuration
//   5. Status updates and error handling
//
// The engine runs in a continuous loop, polling for pending deployments
// and processing them one at a time. It handles all state transitions
// and updates the database accordingly.
package engine

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"mvp-be/internal/apps"
	"mvp-be/internal/db"
	"mvp-be/internal/deployments"
	"mvp-be/internal/dockerbuild"
	"mvp-be/internal/dockerrun"
	"mvp-be/internal/gitrepo"
	"mvp-be/internal/logs"
)

type Engine struct {
	deploymentStore *deployments.Store
	appStore        *apps.Store
	cloner          *gitrepo.Cloner
	builder         *dockerbuild.Builder
	runner          *dockerrun.Runner
	baseDomain      string
	db              *sql.DB // Database connection for advisory locks
}

func NewEngine(
	deploymentStore *deployments.Store,
	appStore *apps.Store,
	cloner *gitrepo.Cloner,
	builder *dockerbuild.Builder,
	runner *dockerrun.Runner,
	baseDomain string,
	database *sql.DB, // Database connection for advisory locks
) *Engine {
	return &Engine{
		deploymentStore: deploymentStore,
		appStore:        appStore,
		cloner:          cloner,
		builder:         builder,
		runner:          runner,
		baseDomain:      baseDomain,
		db:              database,
	}
}

func (e *Engine) ProcessDeployment(ctx context.Context, deploymentID int) error {
	// Get deployment
	deployment, err := e.deploymentStore.GetByID(deploymentID)
	if err != nil {
		return fmt.Errorf("failed to get deployment: %w", err)
	}

	// Get app
	app, err := e.appStore.GetByID(deployment.AppID)
	if err != nil {
		return fmt.Errorf("failed to get app: %w", err)
	}

	log.Printf("[ENGINE] ===== Processing deployment %d for app %s (ID: %d) =====", deploymentID, app.Name, deployment.AppID)
	log.Printf("[ENGINE] App details - Repo: %s, Branch: %s", app.RepoURL, app.Branch)

	// Note: Deployment status is already set to "building" by DequeueNextPending(),
	// so we don't need to update it here. However, we still update app status.
	
	// Update app status to "Building"
	if err := e.appStore.UpdateStatus(deployment.AppID, "Building"); err != nil {
		log.Printf("[ENGINE] WARNING - Failed to update app status to Building: %v", err)
	}

	// Use branch from app, default to "main" only if empty
	branch := app.Branch
	log.Printf("[ENGINE] App branch from database: '%s'", branch)
	if branch == "" {
		log.Printf("[ENGINE] Branch is empty, defaulting to 'main'")
		branch = "main"
	} else {
		log.Printf("[ENGINE] Using branch: '%s'", branch)
	}

	log.Printf("[ENGINE] Step 2: Cloning repository %s (branch: %s)...", app.RepoURL, branch)
	repoPath, err := e.cloner.Clone(app.RepoURL, deploymentID, branch)
	if err != nil {
		log.Printf("[ENGINE] ERROR - Git clone failed: %v", err)
		e.deploymentStore.UpdateError(deploymentID, fmt.Sprintf("Git clone failed: %v", err))
		// Update app status to "Failed"
		e.appStore.UpdateStatus(deployment.AppID, "Failed")
		return fmt.Errorf("git clone failed: %w", err)
	}
	log.Printf("[ENGINE] Repository cloned successfully to: %s", repoPath)

	// Check if Dockerfile exists before attempting to build
	log.Printf("[ENGINE] Step 3: Checking for Dockerfile...")
	if err := gitrepo.CheckDockerfile(repoPath); err != nil {
		log.Printf("[ENGINE] ERROR - Dockerfile check failed: %v", err)
		errorMsg := "Dockerfile is not available in the repository root directory. Please ensure your repository contains a Dockerfile."
		e.deploymentStore.UpdateError(deploymentID, errorMsg)
		// Update app status to "Failed"
		e.appStore.UpdateStatus(deployment.AppID, "Failed")
		return fmt.Errorf("dockerfile check failed: %w", err)
	}

	// Check if this is a worker app (not supported)
	log.Printf("[ENGINE] Step 3.1: Checking if app is a worker/background process...")
	if gitrepo.IsWorkerApp(repoPath) {
		log.Printf("[ENGINE] ERROR - Worker app detected, deployment not supported")
		errorMsg := "Worker apps are not supported yet. Stackyn currently supports only HTTP-based applications that expose a port and serve web requests. Your app does not appear to start a web server. What you can do: • Deploy an API or web app that listens on a port • Wait for background worker support (coming soon)"
		e.deploymentStore.UpdateError(deploymentID, errorMsg)
		// Update app status to "Failed"
		e.appStore.UpdateStatus(deployment.AppID, "Failed")
		return fmt.Errorf("worker app deployment not supported: %w", fmt.Errorf(errorMsg))
	}

	// Ensure package-lock.json exists if package.json is present
	// This fixes the issue where Dockerfiles use `npm ci` but package-lock.json is missing
	log.Printf("[ENGINE] Step 3.5: Ensuring package-lock.json exists...")
	if err := gitrepo.EnsurePackageLock(repoPath); err != nil {
		log.Printf("[ENGINE] WARNING - Failed to ensure package-lock.json: %v (continuing anyway)", err)
		// Don't fail the deployment - let Docker build handle it
	}

	// Detect port from Dockerfile
	log.Printf("[ENGINE] Step 3.6: Detecting application port from Dockerfile...")
	detectedPort := gitrepo.DetectPortFromDockerfile(repoPath)
	log.Printf("[ENGINE] Using port %d for Traefik routing", detectedPort)

	// Step 2: Build Docker image
	// Sanitize app name for Docker image name (only lowercase letters, digits, hyphens, underscores, periods)
	sanitizedName := sanitizeImageName(app.Name)
	imageName := fmt.Sprintf("mvp-%s:%d", sanitizedName, deploymentID)
	log.Printf("[ENGINE] Step 4: Building Docker image: %s (from app name: %s)", imageName, app.Name)
	builtImage, buildLogReader, err := e.builder.Build(ctx, repoPath, imageName)
	if err != nil {
		log.Printf("[ENGINE] ERROR - Docker build failed: %v", err)
		e.deploymentStore.UpdateError(deploymentID, fmt.Sprintf("Docker build failed: %v", err))
		// Update app status to "Failed"
		e.appStore.UpdateStatus(deployment.AppID, "Failed")
		return fmt.Errorf("docker build failed: %w", err)
	}
	log.Printf("[ENGINE] Docker image built successfully: %s", builtImage)

	// Parse and store build log
	log.Printf("[ENGINE] Parsing and storing build logs...")
	buildLog, err := logs.ParseBuildLog(buildLogReader)
	if err != nil {
		log.Printf("[ENGINE] WARNING - Failed to parse build log: %v", err)
	} else {
		if err := e.deploymentStore.UpdateBuildLog(deploymentID, buildLog); err != nil {
			log.Printf("[ENGINE] WARNING - Failed to update build log: %v", err)
		} else {
			log.Printf("[ENGINE] Build log stored successfully")
		}
	}

	// Update image name
	log.Printf("[ENGINE] Updating deployment with image name: %s", builtImage)
	if err := e.deploymentStore.UpdateImage(deploymentID, builtImage); err != nil {
		log.Printf("[ENGINE] ERROR - Failed to update image name: %v", err)
		return fmt.Errorf("failed to update image name: %w", err)
	}

	// Step 3: Run container with Traefik labels and resource limits
	// Sanitize app name for subdomain (DNS-compliant: only lowercase letters, digits, hyphens)
	sanitizedSubdomain := sanitizeSubdomain(app.Name)
	subdomain := fmt.Sprintf("%s-%d", sanitizedSubdomain, deploymentID)
	log.Printf("[ENGINE] Step 5: Running container - Subdomain: %s, Base Domain: %s, AppID: %d, DeploymentID: %d, Port: %d", subdomain, e.baseDomain, deployment.AppID, deploymentID, detectedPort)
	containerID, err := e.runner.Run(ctx, builtImage, subdomain, e.baseDomain, deployment.AppID, deploymentID, detectedPort)
	if err != nil {
		log.Printf("[ENGINE] ERROR - Container run failed: %v", err)
		// Capture detailed error message for deployment record
		errorMsg := fmt.Sprintf("Container run failed: %v", err)
		e.deploymentStore.UpdateError(deploymentID, errorMsg)
		// Update deployment status to FAILED
		e.deploymentStore.UpdateStatus(deploymentID, deployments.StatusFailed)
		// Update app status to "Failed"
		e.appStore.UpdateStatus(deployment.AppID, "Failed")
		return fmt.Errorf("container run failed: %w", err)
	}
	log.Printf("[ENGINE] Container started successfully - ID: %s", containerID)

	// Step 6: Verify new container is healthy before stopping old containers
	// This ensures zero-downtime deployment - old containers keep running if new one fails
	appURL := fmt.Sprintf("https://%s.%s", subdomain, e.baseDomain)
	log.Printf("[ENGINE] Step 6: Verifying new container health at %s...", appURL)
	
	// Wait a bit for Traefik to register the new container and for the app to start
	log.Printf("[ENGINE] Waiting 5 seconds for Traefik routing and app initialization...")
	time.Sleep(5 * time.Second)
	
	// Perform health check - try to reach the HTTP endpoint
	healthCheckPassed := verifyContainerHealth(ctx, appURL)
	
	if !healthCheckPassed {
		log.Printf("[ENGINE] ERROR - New container health check failed at %s", appURL)
		log.Printf("[ENGINE] New container failed to respond - keeping old containers running")
		
		// Clean up the failed new container
		log.Printf("[ENGINE] Cleaning up failed new container: %s", containerID)
		if stopErr := e.runner.Stop(ctx, containerID); stopErr != nil {
			log.Printf("[ENGINE] WARNING - Failed to stop failed container %s: %v", containerID, stopErr)
		}
		if removeErr := e.runner.Remove(ctx, containerID); removeErr != nil {
			log.Printf("[ENGINE] WARNING - Failed to remove failed container %s: %v", containerID, removeErr)
		}
		
		// Mark deployment as failed
		errorMsg := fmt.Sprintf("Container health check failed - container did not respond at %s. Old deployment kept running.", appURL)
		e.deploymentStore.UpdateError(deploymentID, errorMsg)
		e.deploymentStore.UpdateStatus(deploymentID, deployments.StatusFailed)
		
		// Restore app status to previous state (if there was a running deployment, keep it as Healthy)
		previousDeployments, _ := e.deploymentStore.GetRunningByAppID(deployment.AppID)
		if len(previousDeployments) > 0 {
			// There's still a running deployment, keep app as Healthy
			log.Printf("[ENGINE] Previous deployment(s) still running - keeping app status as Healthy")
			// Get the most recent running deployment to restore its URL
			if len(previousDeployments) > 0 {
				prevDeployment := previousDeployments[0]
				if prevDeployment.Subdomain.Valid {
					prevURL := fmt.Sprintf("https://%s.%s", prevDeployment.Subdomain.String, e.baseDomain)
					e.appStore.UpdateStatusAndURL(deployment.AppID, "Healthy", prevURL)
				}
			}
		} else {
			// No previous deployment, mark app as Failed
			e.appStore.UpdateStatus(deployment.AppID, "Failed")
		}
		
		return fmt.Errorf("container health check failed: new container did not respond at %s", appURL)
	}
	
	log.Printf("[ENGINE] New container health check passed - proceeding to stop old containers")

	// Step 7: Stop any previous running deployments for this app
	// Only stop old containers after new one is verified healthy
	log.Printf("[ENGINE] Step 7: Stopping previous running deployments for app %d...", deployment.AppID)
	previousDeployments, err := e.deploymentStore.GetRunningByAppID(deployment.AppID)
	if err != nil {
		log.Printf("[ENGINE] WARNING - Failed to get previous running deployments: %v", err)
	} else if len(previousDeployments) > 0 {
		log.Printf("[ENGINE] Found %d previous running deployment(s), stopping them...", len(previousDeployments))
		for _, prevDeployment := range previousDeployments {
			// Skip the current deployment
			if prevDeployment.ID == deploymentID {
				continue
			}
			
			// Stop and remove the container if it exists
			if prevDeployment.ContainerID.Valid && prevDeployment.ContainerID.String != "" {
				prevContainerID := prevDeployment.ContainerID.String
				log.Printf("[ENGINE] Stopping previous container: %s (deployment %d)", prevContainerID, prevDeployment.ID)
				
				// Stop the container
				if stopErr := e.runner.Stop(ctx, prevContainerID); stopErr != nil {
					log.Printf("[ENGINE] WARNING - Failed to stop previous container %s: %v (may already be stopped)", prevContainerID, stopErr)
				} else {
					log.Printf("[ENGINE] Previous container stopped: %s", prevContainerID)
				}
				
				// Remove the container
				if removeErr := e.runner.Remove(ctx, prevContainerID); removeErr != nil {
					log.Printf("[ENGINE] WARNING - Failed to remove previous container %s: %v", prevContainerID, removeErr)
				} else {
					log.Printf("[ENGINE] Previous container removed: %s", prevContainerID)
				}
			}
			
			// Update deployment status to stopped
			if err := e.deploymentStore.UpdateStatus(prevDeployment.ID, deployments.StatusStopped); err != nil {
				log.Printf("[ENGINE] WARNING - Failed to update previous deployment status to stopped: %v", err)
			} else {
				log.Printf("[ENGINE] Previous deployment %d marked as stopped", prevDeployment.ID)
			}
		}
	}

	// Update container info
	log.Printf("[ENGINE] Step 8: Updating deployment with container info...")
	if err := e.deploymentStore.UpdateContainer(deploymentID, containerID, subdomain); err != nil {
		log.Printf("[ENGINE] ERROR - Failed to update container info: %v", err)
		return fmt.Errorf("failed to update container info: %w", err)
	}

	// Step 9: Mark as running
	log.Printf("[ENGINE] Step 9: Updating deployment status to 'running'...")
	if err := e.deploymentStore.UpdateStatus(deploymentID, deployments.StatusRunning); err != nil {
		log.Printf("[ENGINE] ERROR - Failed to update status: %v", err)
		return fmt.Errorf("failed to update status: %w", err)
	}

	// Update app status to "Healthy" and set URL
	log.Printf("[ENGINE] Step 10: Updating app status to 'Healthy' with URL: %s", appURL)
	if err := e.appStore.UpdateStatusAndURL(deployment.AppID, "Healthy", appURL); err != nil {
		log.Printf("[ENGINE] WARNING - Failed to update app status and URL: %v", err)
	}

	log.Printf("[ENGINE] ===== Deployment %d completed successfully =====", deploymentID)
	log.Printf("[ENGINE] Container ID: %s, Subdomain: %s.%s, URL: %s",
		containerID, subdomain, e.baseDomain, appURL)

	return nil
}

// RunLoop is the main worker loop that processes deployments one at a time.
// It uses PostgreSQL advisory locks to ensure only one build runs globally,
// even when multiple worker instances are running.
//
// The loop:
//   1. Attempts to acquire the global build lock (non-blocking)
//   2. If lock is busy, sleeps briefly and retries
//   3. If lock acquired, atomically dequeues the next pending deployment
//   4. Processes the deployment (with panic recovery)
//   5. Releases the lock (always, even on panic/failure)
//   6. Repeats
func (e *Engine) RunLoop(ctx context.Context) {
	log.Println("[ENGINE] ===== Deployment engine started =====")
	log.Println("[ENGINE] Using global build lock - only one deployment builds at a time")
	log.Println("[ENGINE] Polling for pending deployments...")

	for {
		select {
		case <-ctx.Done():
			log.Println("[ENGINE] ===== Deployment engine stopped =====")
			return
		default:
			// Try to acquire global build lock
			// This ensures only one build runs at a time across all workers
			release, ok, err := db.AcquireGlobalBuildLock(ctx, e.db)
			if err != nil {
				log.Printf("[ENGINE] ERROR - Failed to acquire build lock: %v", err)
				// Sleep before retrying on error
				select {
				case <-ctx.Done():
					return
				case <-time.After(2 * time.Second):
				}
				continue
			}

			if !ok {
				// Lock is busy - another worker is building
				log.Println("[ENGINE] Build lock busy - another worker is building, will retry...")
				// Sleep 1-3 seconds before retrying (randomized to avoid thundering herd)
				select {
				case <-ctx.Done():
					return
				case <-time.After(2 * time.Second):
				}
				continue
			}

			// Lock acquired - we can now process a deployment
			// log.Println("[ENGINE] Build lock acquired")

			// Use an anonymous function to scope the defer properly
			// This ensures the lock is always released, even on panic
			func() {
				defer release() // Always release lock when done (even on panic)

				// Atomically dequeue the next pending deployment and mark it as "building"
				// This uses FOR UPDATE SKIP LOCKED to prevent race conditions
				deployment, err := e.deploymentStore.DequeueNextPending()
				if err != nil {
					if err == sql.ErrNoRows {
						// No pending deployments - release lock and sleep briefly
						// log.Println("[ENGINE] No pending deployments found")
						return // Lock will be released by defer
					}
					// Database error
					log.Printf("[ENGINE] ERROR - Failed to dequeue deployment: %v", err)
					return // Lock will be released by defer
				}

				// Successfully dequeued a deployment
				log.Printf("[ENGINE] Picked deployment dep_%d (app_id: %d)", deployment.ID, deployment.AppID)

				// Process the deployment with panic recovery
				// This ensures the deployment is marked as failed if processing crashes
				func() {
					defer func() {
						if r := recover(); r != nil {
							// Panic occurred - mark deployment as failed and log
							log.Printf("[ENGINE] PANIC - Deployment %d crashed: %v", deployment.ID, r)
							errorMsg := fmt.Sprintf("Deployment processing crashed: %v", r)
							if err := e.deploymentStore.UpdateError(deployment.ID, errorMsg); err != nil {
								log.Printf("[ENGINE] ERROR - Failed to update deployment error: %v", err)
							}
							// App status update
							if err := e.appStore.UpdateStatus(deployment.AppID, "Failed"); err != nil {
								log.Printf("[ENGINE] WARNING - Failed to update app status: %v", err)
							}
						}
					}()

					// Process the deployment
					if err := e.ProcessDeployment(ctx, deployment.ID); err != nil {
						log.Printf("[ENGINE] ERROR - Failed to process deployment %d: %v", deployment.ID, err)
						// Error is already logged and deployment status updated by ProcessDeployment
					}
				}()
			}()

			// Lock has been released (by defer in anonymous function)
			// Sleep briefly before trying to acquire lock again
			select {
			case <-ctx.Done():
				return
			case <-time.After(1 * time.Second):
				// Brief pause before next iteration
			}
		}
	}
}

// sanitizeImageName sanitizes an app name to be a valid Docker image name.
// Docker image names must:
//   - Only contain lowercase letters, digits, underscores, periods, and hyphens
//   - Not start with a period or hyphen
//   - Not contain spaces or special characters
//
// This function:
//   - Converts to lowercase
//   - Replaces invalid characters with hyphens
//   - Removes leading/trailing hyphens and periods
//   - Ensures the result is not empty
func sanitizeImageName(name string) string {
	if name == "" {
		return "app"
	}

	// Convert to lowercase
	sanitized := strings.ToLower(name)

	// Replace invalid characters (anything that's not a-z, 0-9, underscore, period, or hyphen) with hyphens
	invalidCharRegex := regexp.MustCompile(`[^a-z0-9._-]`)
	sanitized = invalidCharRegex.ReplaceAllString(sanitized, "-")

	// Remove consecutive hyphens
	multiHyphenRegex := regexp.MustCompile(`-+`)
	sanitized = multiHyphenRegex.ReplaceAllString(sanitized, "-")

	// Remove leading and trailing hyphens and periods
	sanitized = strings.Trim(sanitized, "-.")

	// Ensure it doesn't start with a period or hyphen (Docker requirement)
	if len(sanitized) > 0 && (sanitized[0] == '.' || sanitized[0] == '-') {
		sanitized = "app" + sanitized
	}

	// If empty after sanitization, use default
	if sanitized == "" {
		return "app"
	}

	// Limit length to 128 characters (Docker image name limit)
	if len(sanitized) > 128 {
		sanitized = sanitized[:128]
		// Trim any trailing hyphens/periods after truncation
		sanitized = strings.Trim(sanitized, "-.")
	}

	return sanitized
}

// sanitizeSubdomain sanitizes an app name to be a valid DNS subdomain.
// DNS subdomains must:
//   - Only contain lowercase letters, digits, and hyphens
//   - Not start or end with a hyphen
//   - Not contain underscores, periods, or other special characters
//
// This function:
//   - Converts to lowercase
//   - Replaces invalid characters with hyphens
//   - Removes leading/trailing hyphens
//   - Ensures the result is not empty
func sanitizeSubdomain(name string) string {
	if name == "" {
		return "app"
	}

	// Convert to lowercase
	sanitized := strings.ToLower(name)

	// Replace invalid characters (anything that's not a-z, 0-9, or hyphen) with hyphens
	invalidCharRegex := regexp.MustCompile(`[^a-z0-9-]`)
	sanitized = invalidCharRegex.ReplaceAllString(sanitized, "-")

	// Remove consecutive hyphens
	multiHyphenRegex := regexp.MustCompile(`-+`)
	sanitized = multiHyphenRegex.ReplaceAllString(sanitized, "-")

	// Remove leading and trailing hyphens
	sanitized = strings.Trim(sanitized, "-")

	// If empty after sanitization, use default
	if sanitized == "" {
		return "app"
	}

	// Limit length to 63 characters (DNS label limit)
	if len(sanitized) > 63 {
		sanitized = sanitized[:63]
		// Trim any trailing hyphens after truncation
		sanitized = strings.Trim(sanitized, "-")
	}

	return sanitized
}

// verifyContainerHealth checks if the container is responding to HTTP requests.
// It attempts to reach the container's URL multiple times with retries.
// Returns true if the container responds with any HTTP status code (even errors),
// false if it cannot be reached at all.
func verifyContainerHealth(ctx context.Context, url string) bool {
	log.Printf("[ENGINE] Health check: Attempting to reach %s", url)
	
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	
	// Try up to 3 times with increasing delays
	maxRetries := 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.Printf("[ENGINE] Health check attempt %d/%d for %s", attempt, maxRetries, url)
		
		// Create request with context
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			log.Printf("[ENGINE] WARNING - Failed to create health check request: %v", err)
			if attempt < maxRetries {
				time.Sleep(2 * time.Second)
				continue
			}
			return false
		}
		
		// Set a reasonable timeout for this request
		req.Header.Set("User-Agent", "Stackyn-HealthCheck/1.0")
		
		// Make the request
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("[ENGINE] Health check attempt %d failed: %v", attempt, err)
			if attempt < maxRetries {
				// Wait before retry (exponential backoff: 2s, 4s)
				waitTime := time.Duration(attempt) * 2 * time.Second
				log.Printf("[ENGINE] Waiting %v before retry...", waitTime)
				time.Sleep(waitTime)
				continue
			}
			log.Printf("[ENGINE] Health check failed after %d attempts: %v", maxRetries, err)
			return false
		}
		
		// Close response body
		resp.Body.Close()
		
		// Any HTTP response (even 4xx/5xx) means the container is running and responding
		// We consider it healthy if we get any response
		log.Printf("[ENGINE] Health check passed - Container responded with status %d", resp.StatusCode)
		return true
	}
	
	log.Printf("[ENGINE] Health check failed - Container did not respond after %d attempts", maxRetries)
	return false
}
