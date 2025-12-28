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
	"crypto/tls"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"mvp-be/internal/apps"
	"mvp-be/internal/db"
	"mvp-be/internal/deployments"
	"mvp-be/internal/dockerbuild"
	"mvp-be/internal/dockerrun"
	"mvp-be/internal/envvars"
	"mvp-be/internal/gitrepo"
	"mvp-be/internal/logs"
)

type Engine struct {
	deploymentStore *deployments.Store
	appStore        *apps.Store
	envVarStore     *envvars.Store
	cloner          *gitrepo.Cloner
	builder         *dockerbuild.Builder
	runner          *dockerrun.Runner
	baseDomain      string
	db              *sql.DB // Database connection for advisory locks
}

func NewEngine(
	deploymentStore *deployments.Store,
	appStore *apps.Store,
	envVarStore *envvars.Store,
	cloner *gitrepo.Cloner,
	builder *dockerbuild.Builder,
	runner *dockerrun.Runner,
	baseDomain string,
	database *sql.DB, // Database connection for advisory locks
) *Engine {
	return &Engine{
		deploymentStore: deploymentStore,
		appStore:        appStore,
		envVarStore:     envVarStore,
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

	// Ensure Dockerfile exists (generate if missing)
	log.Printf("[ENGINE] Step 3: Ensuring Dockerfile exists (generating if missing)...")
	if err := gitrepo.EnsureDockerfile(repoPath); err != nil {
		log.Printf("[ENGINE] ERROR - Dockerfile generation failed: %v", err)
		errorMsg := fmt.Sprintf("Failed to generate Dockerfile: %v. Please ensure your repository contains one of: package.json (Node.js), requirements.txt/pyproject.toml (Python), pom.xml/build.gradle (Java), or go.mod (Go)", err)
		e.deploymentStore.UpdateError(deploymentID, errorMsg)
		// Update app status to "Failed"
		e.appStore.UpdateStatus(deployment.AppID, "Failed")
		return fmt.Errorf("dockerfile generation failed: %w", err)
	}

	// Check if this is a multi-container app (docker-compose.yml) - not supported
	log.Printf("[ENGINE] Step 3.05: Checking for Docker Compose files (multi-container apps not supported)...")
	if err := gitrepo.CheckDockerCompose(repoPath); err != nil {
		log.Printf("[ENGINE] ERROR - Multi-container app detected: %v", err)
		errorMsg := "Multi-container applications are not supported. Stackyn supports only single-container applications. Your repository contains a docker-compose.yml (or similar) file, which indicates a multi-container setup. What you can do: • Remove the docker-compose.yml file and deploy as a single container app • Extract the main application service and deploy it as a standalone container with a Dockerfile • Consider using a different deployment platform if you require multi-container orchestration"
		e.deploymentStore.UpdateError(deploymentID, errorMsg)
		// Update app status to "Failed"
		e.appStore.UpdateStatus(deployment.AppID, "Failed")
		return fmt.Errorf("multi-container app not supported: %w", err)
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

	// Step 3: Fetch environment variables for this app
	log.Printf("[ENGINE] Step 5: Fetching environment variables for app %d...", deployment.AppID)
	envVars, err := e.envVarStore.GetAsMap(deployment.AppID)
	if err != nil {
		log.Printf("[ENGINE] WARNING - Failed to fetch environment variables: %v (continuing without env vars)", err)
		envVars = make(map[string]string) // Use empty map if fetch fails
	} else {
		log.Printf("[ENGINE] Found %d environment variable(s) for app %d", len(envVars), deployment.AppID)
	}
	
	// Always set PORT environment variable to the detected port
	// This ensures the app listens on the port that Traefik routes to
	envVars["PORT"] = strconv.Itoa(detectedPort)
	log.Printf("[ENGINE] Setting PORT environment variable to %d", detectedPort)

	// Step 4: Run container with Traefik labels and resource limits
	// Generate unique 6-character alphabetic subdomain
	subdomain, err := e.deploymentStore.GenerateUniqueSubdomain()
	if err != nil {
		log.Printf("[ENGINE] ERROR - Failed to generate unique subdomain: %v", err)
		e.deploymentStore.UpdateError(deploymentID, fmt.Sprintf("Failed to generate unique subdomain: %v", err))
		e.deploymentStore.UpdateStatus(deploymentID, deployments.StatusFailed)
		e.appStore.UpdateStatus(deployment.AppID, "Failed")
		return fmt.Errorf("failed to generate unique subdomain: %w", err)
	}
	log.Printf("[ENGINE] Step 6: Running container - Subdomain: %s, Base Domain: %s, AppID: %d, DeploymentID: %d, Port: %d", subdomain, e.baseDomain, deployment.AppID, deploymentID, detectedPort)
	containerID, err := e.runner.Run(ctx, builtImage, subdomain, e.baseDomain, deployment.AppID, deploymentID, detectedPort, envVars)
	if err != nil {
		log.Printf("[ENGINE] ERROR - Container run failed: %v", err)
		// Delete the built image since container failed to start
		log.Printf("[ENGINE] Deleting image %s since container failed to start", builtImage)
		if imageErr := e.runner.RemoveImage(ctx, builtImage); imageErr != nil {
			log.Printf("[ENGINE] WARNING - Failed to delete image %s: %v", builtImage, imageErr)
		} else {
			log.Printf("[ENGINE] Image deleted successfully: %s", builtImage)
		}
		
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

	// Step 7: Verify new container is healthy before stopping old containers
	// This ensures zero-downtime deployment - old containers keep running if new one fails
	appURL := fmt.Sprintf("https://%s.%s", subdomain, e.baseDomain)
	appURLHTTP := fmt.Sprintf("http://%s.%s", subdomain, e.baseDomain)
	log.Printf("[ENGINE] Step 6: Verifying new container health at %s...", appURL)
	
	// Wait a bit for Traefik to register the new container, for the app to start,
	// and for Let's Encrypt certificate to be generated (can take 10-30+ seconds)
	log.Printf("[ENGINE] Waiting 10 seconds for Traefik routing, app initialization, and SSL certificate generation...")
	time.Sleep(10 * time.Second)
	
	// Perform health check - try HTTPS first, fallback to HTTP if HTTPS certificate isn't ready yet
	// Let's Encrypt certificates can take time to generate, so we check HTTP as a fallback
	healthCheckPassed := verifyContainerHealth(ctx, appURL, appURLHTTP)
	
	if !healthCheckPassed {
		log.Printf("[ENGINE] ERROR - New container health check failed at %s", appURL)
		log.Printf("[ENGINE] New container failed to respond - keeping old containers running")
		
		// Clean up the failed new container and its image
		log.Printf("[ENGINE] Cleaning up failed new container: %s", containerID)
		if stopErr := e.runner.Stop(ctx, containerID); stopErr != nil {
			log.Printf("[ENGINE] WARNING - Failed to stop failed container %s: %v", containerID, stopErr)
		}
		if removeErr := e.runner.Remove(ctx, containerID); removeErr != nil {
			log.Printf("[ENGINE] WARNING - Failed to remove failed container %s: %v", containerID, removeErr)
		}
		
		// Delete the associated Docker image
		if deployment.ImageName.Valid && deployment.ImageName.String != "" {
			imageName := deployment.ImageName.String
			log.Printf("[ENGINE] Deleting failed container's image: %s", imageName)
			if imageErr := e.runner.RemoveImage(ctx, imageName); imageErr != nil {
				log.Printf("[ENGINE] WARNING - Failed to delete image %s: %v", imageName, imageErr)
			} else {
				log.Printf("[ENGINE] Failed container's image deleted successfully: %s", imageName)
			}
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

	// Step 8: Stop any previous running deployments for this app
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
			
			// Delete the associated Docker image if it exists
			if prevDeployment.ImageName.Valid && prevDeployment.ImageName.String != "" {
				imageName := prevDeployment.ImageName.String
				log.Printf("[ENGINE] Deleting associated image: %s (deployment %d)", imageName, prevDeployment.ID)
				if imageErr := e.runner.RemoveImage(ctx, imageName); imageErr != nil {
					log.Printf("[ENGINE] WARNING - Failed to delete image %s: %v", imageName, imageErr)
				} else {
					log.Printf("[ENGINE] Image deleted successfully: %s", imageName)
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
	log.Printf("[ENGINE] Step 9: Updating deployment with container info...")
	if err := e.deploymentStore.UpdateContainer(deploymentID, containerID, subdomain); err != nil {
		log.Printf("[ENGINE] ERROR - Failed to update container info: %v", err)
		return fmt.Errorf("failed to update container info: %w", err)
	}

	// Step 10: Mark as running
	log.Printf("[ENGINE] Step 10: Updating deployment status to 'running'...")
	if err := e.deploymentStore.UpdateStatus(deploymentID, deployments.StatusRunning); err != nil {
		log.Printf("[ENGINE] ERROR - Failed to update status: %v", err)
		return fmt.Errorf("failed to update status: %w", err)
	}

	// Step 11: Capture and store runtime logs
	log.Printf("[ENGINE] Step 11: Capturing initial runtime logs from container %s...", containerID)
	runtimeLogReader, runtimeLogErr := e.runner.GetLogs(ctx, containerID, "100")
	if runtimeLogErr != nil {
		log.Printf("[ENGINE] WARNING - Failed to fetch runtime logs: %v (continuing anyway)", runtimeLogErr)
	} else {
		runtimeLog, parseErr := logs.ParseRuntimeLog(runtimeLogReader)
		if parseErr != nil {
			log.Printf("[ENGINE] WARNING - Failed to parse runtime logs: %v (continuing anyway)", parseErr)
		} else {
			// Only store logs if they're not empty
			if runtimeLog != "" {
				if updateErr := e.deploymentStore.UpdateRuntimeLog(deploymentID, runtimeLog); updateErr != nil {
					log.Printf("[ENGINE] WARNING - Failed to update runtime log: %v (continuing anyway)", updateErr)
				} else {
					log.Printf("[ENGINE] Runtime logs captured and stored successfully (length: %d)", len(runtimeLog))
				}
			} else {
				log.Printf("[ENGINE] Runtime logs are empty, skipping storage")
			}
		}
	}

	// Update app status to "Healthy" and set URL
	log.Printf("[ENGINE] Step 12: Updating app status to 'Healthy' with URL: %s", appURL)
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
// It tries HTTPS first, and falls back to HTTP if HTTPS certificate isn't ready yet.
// Returns true if the container responds with any HTTP status code (even errors),
// false if it cannot be reached at all.
func verifyContainerHealth(ctx context.Context, httpsURL string, httpURL string) bool {
	log.Printf("[ENGINE] Health check: Attempting to reach %s (will fallback to %s if HTTPS cert not ready)", httpsURL, httpURL)
	
	// Create HTTP client with timeout and skip SSL verification for health checks
	// (we're just checking if the service is up, not validating SSL certificates)
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, // Skip SSL verification for health checks
		},
	}
	client := &http.Client{
		Timeout:   15 * time.Second,
		Transport: transport,
	}
	
	// Try up to 5 times with increasing delays (more retries for Let's Encrypt certificate generation)
	maxRetries := 5
	
	// First, try HTTPS up to maxRetries times
	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.Printf("[ENGINE] Health check attempt %d/%d (HTTPS) for %s", attempt, maxRetries, httpsURL)
		
		req, err := http.NewRequestWithContext(ctx, "GET", httpsURL, nil)
		if err != nil {
			log.Printf("[ENGINE] WARNING - Failed to create HTTPS health check request: %v", err)
			if attempt < maxRetries {
				time.Sleep(time.Duration(attempt) * 3 * time.Second)
			}
			continue
		}
		
		req.Header.Set("User-Agent", "Stackyn-HealthCheck/1.0")
		req.Header.Set("Accept", "*/*")
		
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("[ENGINE] HTTPS health check attempt %d failed: %v", attempt, err)
			if resp != nil {
				resp.Body.Close()
			}
			if attempt < maxRetries {
				// Exponential backoff: 3s, 6s, 12s, 24s
				waitTime := time.Duration(attempt) * 3 * time.Second
				log.Printf("[ENGINE] Waiting %v before retry...", waitTime)
				time.Sleep(waitTime)
			}
			continue
		}
		
		resp.Body.Close()
		log.Printf("[ENGINE] Health check passed (HTTPS) - Container responded with status %d", resp.StatusCode)
		return true
	}
	
	// HTTPS failed, try HTTP as fallback (certificate might not be ready yet)
	log.Printf("[ENGINE] HTTPS health check failed, trying HTTP fallback: %s", httpURL)
	for attempt := 1; attempt <= 3; attempt++ {
		log.Printf("[ENGINE] Health check attempt %d/3 (HTTP fallback) for %s", attempt, httpURL)
		
		req, err := http.NewRequestWithContext(ctx, "GET", httpURL, nil)
		if err != nil {
			log.Printf("[ENGINE] WARNING - Failed to create HTTP health check request: %v", err)
			if attempt < 3 {
				time.Sleep(3 * time.Second)
			}
			continue
		}
		
		req.Header.Set("User-Agent", "Stackyn-HealthCheck/1.0")
		req.Header.Set("Accept", "*/*")
		
		// Use regular HTTP client for HTTP (no TLS)
		httpClient := &http.Client{
			Timeout: 15 * time.Second,
		}
		
		resp, err := httpClient.Do(req)
		if err != nil {
			log.Printf("[ENGINE] HTTP health check attempt %d failed: %v", attempt, err)
			if resp != nil {
				resp.Body.Close()
			}
			if attempt < 3 {
				time.Sleep(3 * time.Second)
			}
			continue
		}
		
		resp.Body.Close()
		log.Printf("[ENGINE] Health check passed (HTTP) - Container responded with status %d (HTTPS cert may still be generating)", resp.StatusCode)
		return true
	}
	
	log.Printf("[ENGINE] Health check failed - Container did not respond after %d HTTPS attempts and 3 HTTP fallback attempts", maxRetries)
	return false
}
