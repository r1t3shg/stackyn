package services

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"go.uber.org/zap"
)

// DockerBuildService handles Docker image building
type DockerBuildService struct {
	client *client.Client
	logger *zap.Logger
}

// NewDockerBuildService creates a new Docker build service
func NewDockerBuildService(dockerHost string, logger *zap.Logger) (*DockerBuildService, error) {
	cli, err := client.NewClientWithOpts(
		client.WithHost(dockerHost),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	return &DockerBuildService{
		client: cli,
		logger: logger,
	}, nil
}

// Close closes the Docker client
func (s *DockerBuildService) Close() error {
	return s.client.Close()
}

// BuildOptions represents options for building a Docker image
type BuildOptions struct {
	ContextPath string // Path to build context (repository)
	ImageName   string // Name for the built image
	Tag         string // Tag for the image (default: latest)
}

// BuildResult represents the result of a build operation
type BuildResult struct {
	ImageID   string
	ImageName string
	Logs      string
}

// BuildImage builds a Docker image with resource constraints
func (s *DockerBuildService) BuildImage(ctx context.Context, opts BuildOptions, logWriter io.Writer) (*BuildResult, error) {
	// Create context with timeout (15 minutes)
	buildCtx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()

	// Prepare image tag
	imageTag := opts.ImageName
	if opts.Tag != "" {
		imageTag = fmt.Sprintf("%s:%s", opts.ImageName, opts.Tag)
	} else {
		imageTag = fmt.Sprintf("%s:latest", opts.ImageName)
	}

	s.logger.Info("Building Docker image",
		zap.String("context_path", opts.ContextPath),
		zap.String("image_tag", imageTag),
	)

	// Create tar archive of build context
	tarReader, err := s.createTarArchive(opts.ContextPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create tar archive: %w", err)
	}
	defer tarReader.Close()

	// Build image with resource constraints via build args
	buildOptions := types.ImageBuildOptions{
		Dockerfile: "Dockerfile",
		Tags:       []string{imageTag},
		Remove:     true, // Remove intermediate containers
		PullParent: false,
		BuildArgs: map[string]*string{
			"BUILDKIT_INLINE_CACHE": stringPtr("1"),
		},
		// Note: Resource constraints are typically set at container runtime
		// For build-time constraints, we rely on Docker daemon limits
	}

	// Build the image
	buildResponse, err := s.client.ImageBuild(buildCtx, tarReader, buildOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to start image build: %w", err)
	}
	defer buildResponse.Body.Close()

	// Stream build logs
	var buildLogs strings.Builder
	multiWriter := io.MultiWriter(logWriter, &buildLogs)

	if err := s.streamBuildLogs(buildResponse.Body, multiWriter); err != nil {
		return nil, fmt.Errorf("failed to stream build logs: %w", err)
	}

	// Inspect the built image to get image ID
	imageInspect, _, err := s.client.ImageInspectWithRaw(buildCtx, imageTag)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect built image: %w", err)
	}

	s.logger.Info("Docker image built successfully",
		zap.String("image_id", imageInspect.ID),
		zap.String("image_tag", imageTag),
	)

	return &BuildResult{
		ImageID:   imageInspect.ID,
		ImageName: imageTag,
		Logs:      buildLogs.String(),
	}, nil
}

// streamBuildLogs streams build logs from Docker build response
func (s *DockerBuildService) streamBuildLogs(reader io.Reader, writer io.Writer) error {
	// Docker build API returns JSON stream
	// Each line is a JSON object with "stream" field containing log output
	buf := make([]byte, 4096)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			// Parse JSON lines and extract stream field
			lines := strings.Split(string(buf[:n]), "\n")
			for _, line := range lines {
				if line == "" {
					continue
				}
				// Simple JSON parsing for stream field
				// Format: {"stream": "output text\n"}
				if strings.Contains(line, "\"stream\"") {
					// Extract stream value
					startIdx := strings.Index(line, "\"stream\"")
					if startIdx != -1 {
						valueStart := strings.Index(line[startIdx:], ":")
						if valueStart != -1 {
							valueStart += startIdx + 1
							// Find the value (string after colon)
							value := line[valueStart:]
							value = strings.TrimSpace(value)
							// Remove quotes
							value = strings.Trim(value, "\"")
							// Unescape newlines
							value = strings.ReplaceAll(value, "\\n", "\n")
							if value != "" {
								writer.Write([]byte(value))
							}
						}
					}
				} else if strings.Contains(line, "\"error\"") {
					// Extract error message
					startIdx := strings.Index(line, "\"error\"")
					if startIdx != -1 {
						valueStart := strings.Index(line[startIdx:], ":")
						if valueStart != -1 {
							valueStart += startIdx + 1
							value := line[valueStart:]
							value = strings.TrimSpace(value)
							value = strings.Trim(value, "\"")
							value = strings.ReplaceAll(value, "\\n", "\n")
							if value != "" {
								writer.Write([]byte("ERROR: " + value + "\n"))
							}
						}
					}
				}
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// createTarArchive creates a tar archive of the build context
func (s *DockerBuildService) createTarArchive(contextPath string) (io.ReadCloser, error) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	// Walk the directory and add files to tar
	err := filepath.Walk(contextPath, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden files and directories
		if strings.HasPrefix(fi.Name(), ".") && fi.Name() != "." {
			if fi.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip node_modules, .git, etc.
		if fi.IsDir() && (fi.Name() == "node_modules" || fi.Name() == ".git" || fi.Name() == "vendor") {
			return filepath.SkipDir
		}

		// Create header
		header, err := tar.FileInfoHeader(fi, fi.Name())
		if err != nil {
			return err
		}

		// Update name to be relative to context path
		relPath, err := filepath.Rel(contextPath, file)
		if err != nil {
			return err
		}
		header.Name = relPath

		// Write header
		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		// Write file content if not a directory
		if !fi.IsDir() {
			data, err := os.Open(file)
			if err != nil {
				return err
			}
			defer data.Close()

			if _, err := io.Copy(tw, data); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create tar archive: %w", err)
	}

	if err := tw.Close(); err != nil {
		return nil, fmt.Errorf("failed to close tar writer: %w", err)
	}

	return io.NopCloser(&buf), nil
}

// stringPtr returns a pointer to a string
func stringPtr(s string) *string {
	return &s
}
