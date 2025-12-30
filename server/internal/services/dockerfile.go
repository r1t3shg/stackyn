package services

import (
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/zap"
)

// DockerfileGenerator generates Dockerfiles using Paketo Buildpacks
type DockerfileGenerator struct {
	logger *zap.Logger
}

// NewDockerfileGenerator creates a new Dockerfile generator
func NewDockerfileGenerator(logger *zap.Logger) *DockerfileGenerator {
	return &DockerfileGenerator{
		logger: logger,
	}
}

// GenerateDockerfile generates a Dockerfile using Paketo Buildpacks for the given runtime
func (g *DockerfileGenerator) GenerateDockerfile(repoPath string, runtime Runtime) error {
	// Check if Dockerfile already exists
	dockerfilePath := filepath.Join(repoPath, "Dockerfile")
	if _, err := os.Stat(dockerfilePath); err == nil {
		g.logger.Info("Dockerfile already exists, skipping generation", zap.String("path", dockerfilePath))
		return nil
	}

	var content string

	switch runtime {
	case RuntimeNodeJS:
		content = g.generateNodeJSDockerfile()
	case RuntimePython:
		content = g.generatePythonDockerfile()
	case RuntimeGo:
		content = g.generateGoDockerfile()
	case RuntimeJava:
		content = g.generateJavaDockerfile()
	case RuntimeRuby, RuntimePHP, RuntimeStatic:
		// These runtimes are not supported by Paketo Buildpacks
		return fmt.Errorf("runtime '%s' is not supported by Paketo Buildpacks. Supported runtimes: Node.js, Python, Go, Java", runtime)
	case RuntimeUnknown:
		return fmt.Errorf("could not detect runtime. Supported runtimes: Node.js, Python, Go, Java")
	default:
		return fmt.Errorf("unsupported runtime: %s. Supported runtimes: Node.js, Python, Go, Java", runtime)
	}

	// Write Dockerfile
	if err := os.WriteFile(dockerfilePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write Dockerfile: %w", err)
	}

	g.logger.Info("Generated Dockerfile using Paketo Buildpacks",
		zap.String("path", dockerfilePath),
		zap.String("runtime", string(runtime)),
	)

	return nil
}

// generateNodeJSDockerfile generates a Dockerfile for Node.js using Paketo Buildpacks
func (g *DockerfileGenerator) generateNodeJSDockerfile() string {
	return `# syntax=docker/dockerfile:1
# Multi-stage build using Paketo Buildpacks for Node.js

# Build stage - Use Paketo Node.js builder
FROM paketobuildpacks/builder-jammy-base:latest AS builder

# Set working directory
WORKDIR /workspace

# Copy application source
COPY --chown=cnb:cnb . .

# Build using Paketo Buildpacks lifecycle
# The builder will automatically detect Node.js and install dependencies
# Note: This requires the CNB lifecycle tools to be available in the builder
RUN /cnb/lifecycle/creator \
    -app=/workspace \
    -layers=/layers \
    -platform=/platform \
    -cache-dir=/cache \
    -log-level=info \
    || (echo "ERROR: Paketo Buildpacks build failed. Ensure your Node.js application has a valid package.json file." && exit 1)

# Production stage - Use Paketo run image
FROM paketobuildpacks/run-jammy-base:latest

# Set working directory
WORKDIR /workspace

# Copy built application from builder stage
COPY --from=builder --chown=cnb:cnb /workspace /workspace
COPY --from=builder --chown=cnb:cnb /layers /layers
COPY --from=builder --chown=cnb:cnb /platform /platform

# Expose dynamic PORT (Paketo Buildpacks set PORT env var at runtime)
# Default to 3000 if PORT is not set
EXPOSE ${PORT:-3000}

# Use the web process from Paketo Buildpacks
# The PORT environment variable will be set by the platform
CMD ["/cnb/process/web"]
`
}

// generatePythonDockerfile generates a Dockerfile for Python using Paketo Buildpacks
func (g *DockerfileGenerator) generatePythonDockerfile() string {
	return `# syntax=docker/dockerfile:1
# Multi-stage build using Paketo Buildpacks for Python

# Build stage - Use Paketo Python builder
FROM paketobuildpacks/builder-jammy-base:latest AS builder

# Set working directory
WORKDIR /workspace

# Copy application source
COPY --chown=cnb:cnb . .

# Build using Paketo Buildpacks lifecycle
# The builder will automatically detect Python and install dependencies
# Note: This requires the CNB lifecycle tools to be available in the builder
RUN /cnb/lifecycle/creator \
    -app=/workspace \
    -layers=/layers \
    -platform=/platform \
    -cache-dir=/cache \
    -log-level=info \
    || (echo "ERROR: Paketo Buildpacks build failed. Ensure your Python application has a valid requirements.txt, setup.py, or Pipfile." && exit 1)

# Production stage - Use Paketo run image
FROM paketobuildpacks/run-jammy-base:latest

# Set working directory
WORKDIR /workspace

# Copy built application from builder stage
COPY --from=builder --chown=cnb:cnb /workspace /workspace
COPY --from=builder --chown=cnb:cnb /layers /layers
COPY --from=builder --chown=cnb:cnb /platform /platform

# Expose dynamic PORT (Paketo Buildpacks set PORT env var at runtime)
# Default to 8000 if PORT is not set
EXPOSE ${PORT:-8000}

# Use the web process from Paketo Buildpacks
# The PORT environment variable will be set by the platform
CMD ["/cnb/process/web"]
`
}

// generateGoDockerfile generates a Dockerfile for Go using Paketo Buildpacks
func (g *DockerfileGenerator) generateGoDockerfile() string {
	return `# syntax=docker/dockerfile:1
# Multi-stage build using Paketo Buildpacks for Go

# Build stage - Use Paketo Go builder
FROM paketobuildpacks/builder-jammy-base:latest AS builder

# Set working directory
WORKDIR /workspace

# Copy application source
COPY --chown=cnb:cnb . .

# Build using Paketo Buildpacks lifecycle
# The builder will automatically detect Go and build the binary
# Note: This requires the CNB lifecycle tools to be available in the builder
RUN /cnb/lifecycle/creator \
    -app=/workspace \
    -layers=/layers \
    -platform=/platform \
    -cache-dir=/cache \
    -log-level=info \
    || (echo "ERROR: Paketo Buildpacks build failed. Ensure your Go application has a valid go.mod file." && exit 1)

# Production stage - Use Paketo run image
FROM paketobuildpacks/run-jammy-base:latest

# Set working directory
WORKDIR /workspace

# Copy built application from builder stage
COPY --from=builder --chown=cnb:cnb /workspace /workspace
COPY --from=builder --chown=cnb:cnb /layers /layers
COPY --from=builder --chown=cnb:cnb /platform /platform

# Expose dynamic PORT (Paketo Buildpacks set PORT env var at runtime)
# Default to 8080 if PORT is not set
EXPOSE ${PORT:-8080}

# Use the web process from Paketo Buildpacks
# The PORT environment variable will be set by the platform
CMD ["/cnb/process/web"]
`
}

// generateJavaDockerfile generates a Dockerfile for Java using Paketo Buildpacks
func (g *DockerfileGenerator) generateJavaDockerfile() string {
	return `# syntax=docker/dockerfile:1
# Multi-stage build using Paketo Buildpacks for Java

# Build stage - Use Paketo Java builder
FROM paketobuildpacks/builder-jammy-base:latest AS builder

# Set working directory
WORKDIR /workspace

# Copy application source
COPY --chown=cnb:cnb . .

# Build using Paketo Buildpacks lifecycle
# The builder will automatically detect Java (Maven/Gradle) and build
# Note: This requires the CNB lifecycle tools to be available in the builder
RUN /cnb/lifecycle/creator \
    -app=/workspace \
    -layers=/layers \
    -platform=/platform \
    -cache-dir=/cache \
    -log-level=info \
    || (echo "ERROR: Paketo Buildpacks build failed. Ensure your Java application has a valid pom.xml or build.gradle file." && exit 1)

# Production stage - Use Paketo run image
FROM paketobuildpacks/run-jammy-base:latest

# Set working directory
WORKDIR /workspace

# Copy built application from builder stage
COPY --from=builder --chown=cnb:cnb /workspace /workspace
COPY --from=builder --chown=cnb:cnb /layers /layers
COPY --from=builder --chown=cnb:cnb /platform /platform

# Expose dynamic PORT (Paketo Buildpacks set PORT env var at runtime)
# Default to 8080 if PORT is not set
EXPOSE ${PORT:-8080}

# Use the web process from Paketo Buildpacks
# The PORT environment variable will be set by the platform
CMD ["/cnb/process/web"]
`
}
