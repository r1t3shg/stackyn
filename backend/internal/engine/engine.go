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
	"fmt"
	"log"
	"strings"
	"time"

	"mvp-be/internal/apps"
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
}

func NewEngine(
	deploymentStore *deployments.Store,
	appStore *apps.Store,
	cloner *gitrepo.Cloner,
	builder *dockerbuild.Builder,
	runner *dockerrun.Runner,
	baseDomain string,
) *Engine {
	return &Engine{
		deploymentStore: deploymentStore,
		appStore:        appStore,
		cloner:          cloner,
		builder:         builder,
		runner:          runner,
		baseDomain:      baseDomain,
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

	// Step 1: Clone repository
	log.Printf("[ENGINE] Step 1: Updating deployment status to 'building'...")
	if err := e.deploymentStore.UpdateStatus(deploymentID, deployments.StatusBuilding); err != nil {
		log.Printf("[ENGINE] ERROR - Failed to update status: %v", err)
		return fmt.Errorf("failed to update status: %w", err)
	}
	
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

	// Step 2: Build Docker image
	imageName := fmt.Sprintf("mvp-%s:%d", strings.ToLower(app.Name), deploymentID)
	log.Printf("[ENGINE] Step 4: Building Docker image: %s", imageName)
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
	subdomain := fmt.Sprintf("%s-%d", strings.ToLower(app.Name), deploymentID)
	log.Printf("[ENGINE] Step 5: Running container - Subdomain: %s, Base Domain: %s, AppID: %d, DeploymentID: %d", subdomain, e.baseDomain, deployment.AppID, deploymentID)
	containerID, err := e.runner.Run(ctx, builtImage, subdomain, e.baseDomain, deployment.AppID, deploymentID)
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

	// Update container info
	log.Printf("[ENGINE] Step 6: Updating deployment with container info...")
	if err := e.deploymentStore.UpdateContainer(deploymentID, containerID, subdomain); err != nil {
		log.Printf("[ENGINE] ERROR - Failed to update container info: %v", err)
		return fmt.Errorf("failed to update container info: %w", err)
	}

	// Step 4: Mark as running
	log.Printf("[ENGINE] Step 7: Updating deployment status to 'running'...")
	if err := e.deploymentStore.UpdateStatus(deploymentID, deployments.StatusRunning); err != nil {
		log.Printf("[ENGINE] ERROR - Failed to update status: %v", err)
		return fmt.Errorf("failed to update status: %w", err)
	}

	// Update app status to "Healthy" and set URL
	appURL := fmt.Sprintf("https://%s.%s", subdomain, e.baseDomain)
	log.Printf("[ENGINE] Step 8: Updating app status to 'Healthy' with URL: %s", appURL)
	if err := e.appStore.UpdateStatusAndURL(deployment.AppID, "Healthy", appURL); err != nil {
		log.Printf("[ENGINE] WARNING - Failed to update app status and URL: %v", err)
	}

	log.Printf("[ENGINE] ===== Deployment %d completed successfully =====", deploymentID)
	log.Printf("[ENGINE] Container ID: %s, Subdomain: %s.%s, URL: %s",
		containerID, subdomain, e.baseDomain, appURL)

	return nil
}

func (e *Engine) RunLoop(ctx context.Context) {
	log.Println("[ENGINE] ===== Deployment engine started =====")
	log.Println("[ENGINE] Polling for pending deployments every 2 seconds...")

	for {
		select {
		case <-ctx.Done():
			log.Println("[ENGINE] ===== Deployment engine stopped =====")
			return
		default:
			// Get pending deployments
			pending, err := e.deploymentStore.GetPending()
			if err != nil {
				log.Printf("[ENGINE] ERROR - Failed to fetch pending deployments: %v", err)
				continue
			}

			if len(pending) > 0 {
				log.Printf("[ENGINE] Found %d pending deployment(s)", len(pending))
			}

			// Process each pending deployment
			for _, deployment := range pending {
				if err := e.ProcessDeployment(ctx, deployment.ID); err != nil {
					log.Printf("[ENGINE] ERROR - Failed to process deployment %d: %v", deployment.ID, err)
				}
			}

			// Simple polling - in production, use a better mechanism
			// Sleep for a short duration before checking again
			select {
			case <-ctx.Done():
				return
			case <-time.After(2 * time.Second):
				// Poll every 2 seconds
			}
		}
	}
}
