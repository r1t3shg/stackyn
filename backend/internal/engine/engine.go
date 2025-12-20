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

	log.Printf("Processing deployment %d for app %s", deploymentID, app.Name)

	// Step 1: Clone repository
	if err := e.deploymentStore.UpdateStatus(deploymentID, deployments.StatusBuilding); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	// Default branch to "main" if not set
	branch := app.Branch
	if branch == "" {
		branch = "main"
	}

	repoPath, err := e.cloner.Clone(app.RepoURL, deploymentID, branch)
	if err != nil {
		e.deploymentStore.UpdateError(deploymentID, fmt.Sprintf("Git clone failed: %v", err))
		return fmt.Errorf("git clone failed: %w", err)
	}

	// Check if Dockerfile exists before attempting to build
	if err := gitrepo.CheckDockerfile(repoPath); err != nil {
		errorMsg := "Dockerfile is not available in the repository root directory. Please ensure your repository contains a Dockerfile."
		e.deploymentStore.UpdateError(deploymentID, errorMsg)
		return fmt.Errorf("dockerfile check failed: %w", err)
	}

	// Step 2: Build Docker image
	imageName := fmt.Sprintf("mvp-%s:%d", strings.ToLower(app.Name), deploymentID)
	builtImage, buildLogReader, err := e.builder.Build(ctx, repoPath, imageName)
	if err != nil {
		e.deploymentStore.UpdateError(deploymentID, fmt.Sprintf("Docker build failed: %v", err))
		return fmt.Errorf("docker build failed: %w", err)
	}

	// Parse and store build log
	buildLog, err := logs.ParseBuildLog(buildLogReader)
	if err != nil {
		log.Printf("Warning: failed to parse build log: %v", err)
	} else {
		if err := e.deploymentStore.UpdateBuildLog(deploymentID, buildLog); err != nil {
			log.Printf("Warning: failed to update build log: %v", err)
		}
	}

	// Update image name
	if err := e.deploymentStore.UpdateImage(deploymentID, builtImage); err != nil {
		return fmt.Errorf("failed to update image name: %w", err)
	}

	// Step 3: Run container with Traefik labels
	subdomain := fmt.Sprintf("%s-%d", strings.ToLower(app.Name), deploymentID)
	containerID, err := e.runner.Run(ctx, builtImage, subdomain, e.baseDomain)
	if err != nil {
		e.deploymentStore.UpdateError(deploymentID, fmt.Sprintf("Container run failed: %v", err))
		return fmt.Errorf("container run failed: %w", err)
	}

	// Update container info
	if err := e.deploymentStore.UpdateContainer(deploymentID, containerID, subdomain); err != nil {
		return fmt.Errorf("failed to update container info: %w", err)
	}

	// Step 4: Mark as running
	if err := e.deploymentStore.UpdateStatus(deploymentID, deployments.StatusRunning); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	log.Printf("Deployment %d completed successfully. Container: %s, Subdomain: %s.%s",
		deploymentID, containerID, subdomain, e.baseDomain)

	return nil
}

func (e *Engine) RunLoop(ctx context.Context) {
	log.Println("Deployment engine started")

	for {
		select {
		case <-ctx.Done():
			log.Println("Deployment engine stopped")
			return
		default:
			// Get pending deployments
			pending, err := e.deploymentStore.GetPending()
			if err != nil {
				log.Printf("Error fetching pending deployments: %v", err)
				continue
			}

			// Process each pending deployment
			for _, deployment := range pending {
				if err := e.ProcessDeployment(ctx, deployment.ID); err != nil {
					log.Printf("Error processing deployment %d: %v", deployment.ID, err)
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
