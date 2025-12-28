// Package dockerbuild provides functionality for building Docker images.
// It uses the Docker API to build images from a repository path.
package dockerbuild

import (
	"context"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

// Builder handles building Docker images using the Docker API.
// It wraps the Docker client and provides a simplified interface for building images.
type Builder struct {
	// client is the Docker API client used to communicate with the Docker daemon
	client *client.Client
}

// NewBuilder creates a new Builder instance connected to the Docker daemon.
//
// Parameters:
//   - dockerHost: The Docker daemon address (e.g., "unix:///var/run/docker.sock" or "tcp://host:port")
//
// Returns:
//   - *Builder: A new Builder instance ready to build images, or nil on error
//   - error: Error if Docker client creation fails (connection issue, invalid host, etc.)
func NewBuilder(dockerHost string) (*Builder, error) {
	log.Printf("[DOCKER] Initializing Docker builder - Host: %s", dockerHost)
	// Create Docker client with host configuration and API version negotiation
	cli, err := client.NewClientWithOpts(
		client.WithHost(dockerHost),
		client.WithAPIVersionNegotiation(), // Automatically negotiate API version with daemon
	)
	if err != nil {
		log.Printf("[DOCKER] ERROR - Failed to create Docker client: %v", err)
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	log.Printf("[DOCKER] Docker builder initialized successfully")
	return &Builder{client: cli}, nil
}

// Build builds a Docker image from a repository path.
// It creates a tar archive of the repository and sends it to Docker for building.
// The build process looks for a Dockerfile in the root of the repository.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - repoPath: The local filesystem path to the cloned repository
//   - imageName: The name to tag the built image (e.g., "mvp-myapp:123")
//
// Returns:
//   - string: The image name that was built (same as input imageName)
//   - io.ReadCloser: A stream containing the Docker build output/logs (must be closed by caller)
//   - error: Error if tar creation fails, Docker build fails, or image cannot be created
func (b *Builder) Build(ctx context.Context, repoPath string, imageName string) (string, io.ReadCloser, error) {
	log.Printf("[DOCKER] Starting build - Image: %s, Context: %s", imageName, repoPath)
	// Configure Docker build options
	buildOptions := types.ImageBuildOptions{
		Tags:       []string{imageName}, // Tag the image with the provided name
		Dockerfile: "Dockerfile",         // Look for Dockerfile in the root of the build context
		Remove:    true,                 // Remove intermediate containers after build
	}

	// Create a tar archive of the repository to send as build context
	// Docker requires the build context to be a tar stream
	log.Printf("[DOCKER] Creating tar archive of build context...")
	buildContext, err := createTarContext(repoPath)
	if err != nil {
		log.Printf("[DOCKER] ERROR - Failed to create build context: %v", err)
		return "", nil, fmt.Errorf("failed to create build context: %w", err)
	}
	// Ensure the tar stream is closed when done
	defer buildContext.Close()

	// Send build request to Docker daemon
	// This starts the build process and returns a response with build logs
	log.Printf("[DOCKER] Sending build request to Docker daemon...")
	buildResponse, err := b.client.ImageBuild(ctx, buildContext, buildOptions)
	if err != nil {
		log.Printf("[DOCKER] ERROR - Build failed: %v", err)
		return "", nil, fmt.Errorf("failed to build image: %w", err)
	}

	log.Printf("[DOCKER] Build started successfully for image: %s", imageName)
	// Return the image name and the build log stream
	// The caller should read from buildResponse.Body to get build progress
	return imageName, buildResponse.Body, nil
}

// createTarContext creates a tar.gz archive of the given directory path.
// This is used to send the repository to Docker as a build context.
// The tar command is executed and its stdout is returned as a ReadCloser.
//
// Parameters:
//   - path: The directory path to archive
//
// Returns:
//   - io.ReadCloser: A stream of the tar.gz archive, or nil on error
//   - error: Error if tar command setup fails or command cannot start
//
// Note: The tar command runs in the background. In production, you might want to
// use Go's archive/tar package for better control and error handling.
func createTarContext(path string) (io.ReadCloser, error) {
	// Create tar command: tar -czf - -C {path} .
	// -c: create archive
	// -z: compress with gzip
	// -f -: write to stdout
	// -C {path}: change to directory before archiving
	// .: archive current directory contents
	cmd := exec.Command("tar", "-czf", "-", "-C", path, ".")
	
	// Get stdout pipe to read the tar stream
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	// Start the tar command (it will run in the background)
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start tar command: %w", err)
	}

	// Note: The command will run in the background. In production,
	// you'd want to ensure it completes or handle errors properly.
	// For now, we return the stdout stream which will be consumed by Docker.
	return stdout, nil
}

// ImageExists checks if a Docker image exists by inspecting it.
// This is used to verify that an image was successfully created after a build.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - imageName: The name of the image to check (e.g., "mvp-myapp:123")
//
// Returns:
//   - bool: True if the image exists, false otherwise
//   - error: Error if inspection fails (other than "not found")
func (b *Builder) ImageExists(ctx context.Context, imageName string) (bool, error) {
	_, _, err := b.client.ImageInspectWithRaw(ctx, imageName)
	if err != nil {
		// Check if error is "image not found" type
		// Docker API returns errors containing "No such image" when image doesn't exist
		errStr := err.Error()
		if strings.Contains(errStr, "No such image") || 
		   strings.Contains(errStr, "image not known") ||
		   strings.Contains(errStr, "not found") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

