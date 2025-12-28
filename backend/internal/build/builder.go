// Package build provides a common interface for container image builders.
// This allows the engine to use either Dockerfile-based builds or CNB-based builds
// without being tightly coupled to a specific implementation.
package build

import (
	"context"
	"io"
)

// Builder is the interface for building container images.
// Both dockerbuild.Builder and buildpacks.BuildpacksBuilder implement this interface.
//
// The interface provides:
//   - Build: Build a container image from source code and return build logs
//   - ImageExists: Check if an image exists in the Docker daemon
type Builder interface {
	// Build builds a container image from source code.
	// It returns the image name, a stream of build logs, and any error encountered.
	Build(ctx context.Context, repoPath string, imageName string) (string, io.ReadCloser, error)

	// ImageExists checks if a Docker image exists.
	// It returns true if the image exists, false otherwise, and any error encountered.
	ImageExists(ctx context.Context, imageName string) (bool, error)
}

