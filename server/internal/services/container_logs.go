package services

import (
	"context"
	"fmt"
	"io"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"go.uber.org/zap"
)

// ContainerLogService handles container log streaming
type ContainerLogService struct {
	client *client.Client
	logger *zap.Logger
}

// NewContainerLogService creates a new container log service
func NewContainerLogService(dockerHost string, logger *zap.Logger) (*ContainerLogService, error) {
	cli, err := client.NewClientWithOpts(
		client.WithHost(dockerHost),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	return &ContainerLogService{
		client: cli,
		logger: logger,
	}, nil
}

// Close closes the Docker client
func (s *ContainerLogService) Close() error {
	return s.client.Close()
}

// StreamContainerLogs streams logs from a container
func (s *ContainerLogService) StreamContainerLogs(ctx context.Context, containerID string, since string, tail string, follow bool) (io.ReadCloser, error) {
	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     follow,
		Timestamps: true,
	}

	if since != "" {
		options.Since = since
	}

	if tail != "" {
		options.Tail = tail
	}

	reader, err := s.client.ContainerLogs(ctx, containerID, options)
	if err != nil {
		return nil, fmt.Errorf("failed to stream container logs: %w", err)
	}

	return reader, nil
}

// GetContainerLogs gets logs from a container (non-streaming)
func (s *ContainerLogService) GetContainerLogs(ctx context.Context, containerID string, since string, tail string) (string, error) {
	reader, err := s.StreamContainerLogs(ctx, containerID, since, tail, false)
	if err != nil {
		return "", err
	}
	defer reader.Close()

	// Read all logs
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("failed to read container logs: %w", err)
	}

	// Docker logs are prefixed with 8-byte header, strip it
	// Format: [STREAM_TYPE, 0, 0, 0, SIZE1, SIZE2, SIZE3, SIZE4] followed by log data
	if len(data) > 8 {
		return string(data[8:]), nil
	}

	return string(data), nil
}

