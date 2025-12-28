// Package buildpacks provides Cloud Native Buildpacks (CNB) integration for Stackyn.
//
// Cloud Native Buildpacks (CNB) is an industry-standard build system used by platforms
// like Heroku, Railway, Google Cloud Run, and others. Instead of manually writing Dockerfiles,
// CNB automatically:
//   - Detects the application type (Node.js, Python, Java, Go, etc.)
//   - Installs dependencies and build tools
//   - Compiles and builds the application
//   - Creates an optimized production-ready container image
//
// Key Benefits:
//   - No Dockerfile required - CNB handles everything automatically
//   - Supports 20+ languages and frameworks out of the box
//   - Optimized images with security updates
//   - Consistent builds across different projects
//   - Handles complex build scenarios (TypeScript, build tools, etc.)
//
// Architecture:
//   - Buildpacks: Small, reusable units that handle specific tasks (e.g., Node.js detection, npm install)
//   - Builder: A collection of buildpacks configured for specific languages
//   - Lifecycle: Orchestrates the build process (detect, analyze, restore, build, export)
//
// Stackyn uses the Paketo Buildpacks builder which supports:
//   - Node.js (including TypeScript, npm, yarn, pnpm)
//   - Python (pip, poetry, pipenv)
//   - Java (Maven, Gradle)
//   - Go (standard Go modules)
//   - And many more...
//
// The builder uses the pack CLI tool to build images. Pack is the official CLI for CNB
// and provides a simple interface to build container images using buildpacks.
//
// Usage in Stackyn:
//   The BuildpacksBuilder implements the same interface as dockerbuild.Builder, allowing
//   the engine to use either Dockerfile-based builds or CNB builds. When using CNB:
//   1. Skip Dockerfile generation (CNB doesn't need it)
//   2. Use BuildpacksBuilder.Build() to build images directly
//   3. CNB automatically sets PORT environment variable (usually 8080)
//   4. Build process is simpler and more reliable
package buildpacks

import (
	"context"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strings"

	"github.com/docker/docker/client"
)

// BuildpacksBuilder handles building container images using Cloud Native Buildpacks.
// Unlike Dockerfile-based builds, CNB automatically detects and handles:
//   - Language/framework detection
//   - Dependency installation
//   - Build compilation
//   - Production image optimization
//
// The builder uses the "pack" CLI tool, which is the official Cloud Native Buildpacks CLI.
// Pack must be installed on the system for this builder to work.
type BuildpacksBuilder struct {
	// builder is the CNB builder image to use (e.g., "paketobuildpacks/builder:base")
	// This builder contains all the buildpacks needed to build various application types
	builder string

	// packCLIPath is the path to the pack CLI executable
	// If empty, "pack" is assumed to be in PATH
	packCLIPath string

	// dockerHost is the Docker daemon address (used to set DOCKER_HOST env var for pack)
	dockerHost string

	// client is the Docker API client used to verify image existence
	// This allows us to check if images exist without requiring docker CLI in PATH
	client *client.Client
}

// NewBuildpacksBuilder creates a new BuildpacksBuilder instance.
//
// Parameters:
//   - builder: The CNB builder image to use. Recommended builders:
//     * "paketobuildpacks/builder:base" - Supports Node.js, Python, Go, Java, .NET Core, PHP, Ruby
//     * "paketobuildpacks/builder:full" - Includes additional language support
//     * "heroku/buildpacks:20" - Heroku's official builder (legacy)
//   - dockerHost: The Docker daemon address (e.g., "unix:///var/run/docker.sock")
//   - packCLIPath: Optional path to pack CLI executable (empty = use "pack" from PATH)
//
// Returns:
//   - *BuildpacksBuilder: A new builder instance
//   - error: Error if pack CLI cannot be found or builder validation fails
//
// Example:
//   builder, err := buildpacks.NewBuildpacksBuilder(
//       "paketobuildpacks/builder:base",
//       "unix:///var/run/docker.sock",
//       "",
//   )
func NewBuildpacksBuilder(builder, dockerHost, packCLIPath string) (*BuildpacksBuilder, error) {
	log.Printf("[BUILDPACKS] Initializing Cloud Native Buildpacks builder - Builder: %s", builder)

	// Determine pack CLI path
	packPath := packCLIPath
	if packPath == "" {
		packPath = "pack"
	}

	// Verify that pack CLI is available
	// CNB requires the pack CLI tool to be installed on the system
	log.Printf("[BUILDPACKS] Checking for pack CLI at: %s", packPath)
	checkCmd := exec.Command(packPath, "version")
	if err := checkCmd.Run(); err != nil {
		return nil, fmt.Errorf("pack CLI not found at '%s'. Please install pack CLI: https://buildpacks.io/docs/tools/pack/", packPath)
	}

	// Get pack version for logging
	versionCmd := exec.Command(packPath, "version")
	versionOutput, err := versionCmd.Output()
	if err == nil {
		log.Printf("[BUILDPACKS] Pack CLI version: %s", strings.TrimSpace(string(versionOutput)))
	}

	// Create Docker API client for image existence checks
	// This allows us to verify images without requiring docker CLI in PATH
	dockerClient, err := client.NewClientWithOpts(
		client.WithHost(dockerHost),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	log.Printf("[BUILDPACKS] Buildpacks builder initialized successfully with builder: %s", builder)
	return &BuildpacksBuilder{
		builder:     builder,
		packCLIPath: packPath,
		dockerHost:  dockerHost,
		client:      dockerClient,
	}, nil
}

// Build builds a container image from a source directory using Cloud Native Buildpacks.
//
// The build process follows these CNB lifecycle phases:
//   1. Detect: Identify which buildpacks to use based on source code
//   2. Analyze: Restore cached layers if available
//   3. Restore: Restore dependency cache from previous builds
//   4. Build: Install dependencies and build the application
//   5. Export: Create the final container image
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - repoPath: The local filesystem path to the source code directory
//   - imageName: The name to tag the built image (e.g., "mvp-myapp:123")
//
// Returns:
//   - string: The image name that was built (same as input imageName)
//   - io.ReadCloser: A stream containing the build output/logs (must be closed by caller)
//   - error: Error if pack CLI execution fails or build fails
//
// Example:
//   imageName, logs, err := builder.Build(ctx, "/path/to/app", "myapp:123")
//   if err != nil {
//       log.Fatal(err)
//   }
//   defer logs.Close()
//   // Read logs to show build progress...
func (b *BuildpacksBuilder) Build(ctx context.Context, repoPath string, imageName string) (string, io.ReadCloser, error) {
	log.Printf("[BUILDPACKS] Starting CNB build - Image: %s, Source: %s, Builder: %s", imageName, repoPath, b.builder)

	// Build the pack CLI command
	// pack build <image-name> --path <source-path> --builder <builder-image>
	//
	// Additional flags:
	//   --pull-policy if-not-present: Only pull builder image if not locally available
	//   --trust-builder: Trust the builder (required for CNB security model)
	//   --verbose: Show detailed build logs
	//
	// Note: Pack builds images directly - it doesn't generate Dockerfiles.
	// The entire build process is handled by the buildpacks.
	cmd := exec.CommandContext(ctx, b.packCLIPath,
		"build", imageName,
		"--path", repoPath,
		"--builder", b.builder,
		"--pull-policy", "if-not-present", // Only pull builder if not cached locally
		"--trust-builder",                  // Trust the builder (required for CNB security)
	)

	// Set environment variables
	// DOCKER_HOST tells pack where to find the Docker daemon
	cmd.Env = append(cmd.Env, fmt.Sprintf("DOCKER_HOST=%s", b.dockerHost))

	// Get stdout and stderr pipes to capture build logs
	// Pack outputs build logs to stdout, so we capture both streams
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		stdout.Close()
		return "", nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Combine stdout and stderr into a single stream
	// This allows callers to read all build output from a single source
	logReader := io.MultiReader(stdout, stderr)

	// Start the pack build process
	// The command runs asynchronously - we return immediately with the log stream
	log.Printf("[BUILDPACKS] Starting pack build process...")
	if err := cmd.Start(); err != nil {
		stdout.Close()
		stderr.Close()
		return "", nil, fmt.Errorf("failed to start pack build: %w", err)
	}

	// Create a reader that will close the pipes when done
	// We also need to wait for the command to finish, so we use a custom closer
	result := &buildLogReader{
		reader: logReader,
		stdout: stdout,
		stderr: stderr,
		cmd:    cmd,
	}

	log.Printf("[BUILDPACKS] Pack build started successfully for image: %s", imageName)
	return imageName, result, nil
}

// buildLogReader wraps the log reader and ensures proper cleanup when closed.
// It waits for the pack command to finish and closes all pipes.
type buildLogReader struct {
	reader io.Reader
	stdout io.ReadCloser
	stderr io.ReadCloser
	cmd    *exec.Cmd
}

// Read implements io.Reader interface.
func (r *buildLogReader) Read(p []byte) (n int, err error) {
	return r.reader.Read(p)
}

// Close implements io.Closer interface.
// It waits for the pack command to finish and closes all pipes.
// IMPORTANT: This method returns an error if the pack build failed.
// The caller (ParseBuildLog) should check the error returned from Close()
// to detect build failures. Pack CLI outputs plain text (not JSON), so we
// detect failures by checking the exit code.
func (r *buildLogReader) Close() error {
	// Wait for the command to finish
	// This ensures the build completes before we return
	waitErr := r.cmd.Wait()
	
	// Close pipes first (always close, even on error)
	closeErr := r.closePipes()
	
	// If command failed, return that error (pack build failed)
	if waitErr != nil {
		log.Printf("[BUILDPACKS] Pack build command exited with error: %v", waitErr)
		if closeErr != nil {
			return fmt.Errorf("build failed: %w (also failed to close pipes: %v)", waitErr, closeErr)
		}
		// Return error so caller knows build failed
		// The error will be wrapped, but ParseBuildLog will detect it
		return waitErr
	}

	log.Printf("[BUILDPACKS] Pack build completed successfully")
	return closeErr
}

// closePipes closes all pipes and returns the first error encountered.
func (r *buildLogReader) closePipes() error {
	var firstErr error

	if err := r.stdout.Close(); err != nil && firstErr == nil {
		firstErr = err
	}

	if err := r.stderr.Close(); err != nil && firstErr == nil {
		firstErr = err
	}

	return firstErr
}

// ImageExists checks if a Docker image exists by inspecting it using the Docker API client.
// This uses the same approach as dockerbuild.Builder to avoid requiring docker CLI in PATH.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - imageName: The image name to check (e.g., "mvp-myapp:123")
//
// Returns:
//   - bool: True if image exists, false otherwise
//   - error: Error if Docker inspection fails (not if image doesn't exist)
func (b *BuildpacksBuilder) ImageExists(ctx context.Context, imageName string) (bool, error) {
	_, _, err := b.client.ImageInspectWithRaw(ctx, imageName)
	if err != nil {
		// Check if error is "image not found" type
		// Use client.IsErrNotFound to properly detect when image doesn't exist
		if client.IsErrNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to inspect image %s: %w", imageName, err)
	}
	return true, nil
}

// GetDefaultBuilder returns the default CNB builder recommended for Stackyn.
// This is the Paketo Buildpacks base builder which supports the most common languages.
//
// Supported languages by paketobuildpacks/builder:base:
//   - Node.js (npm, yarn, pnpm) - with TypeScript support
//   - Python (pip, poetry, pipenv)
//   - Java (Maven, Gradle)
//   - Go (standard Go modules)
//   - .NET Core
//   - PHP
//   - Ruby
//   - And more...
//
// Returns:
//   - string: The default builder image name
func GetDefaultBuilder() string {
	// Paketo Buildpacks is the Cloud Foundry Foundation's buildpacks implementation
	// The "base" builder is recommended for most use cases as it balances
	// language support with image size
	return "paketobuildpacks/builder:base"
}

// ValidatePackCLI checks if the pack CLI is installed and available.
// This is useful for startup validation to fail fast if pack is missing.
//
// Parameters:
//   - packCLIPath: Path to pack CLI (empty = check "pack" in PATH)
//
// Returns:
//   - error: Error if pack CLI is not found or not executable
func ValidatePackCLI(packCLIPath string) error {
	path := packCLIPath
	if path == "" {
		path = "pack"
	}

	cmd := exec.Command(path, "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pack CLI not found at '%s'. Please install from: https://buildpacks.io/docs/tools/pack/", path)
	}

	return nil
}

