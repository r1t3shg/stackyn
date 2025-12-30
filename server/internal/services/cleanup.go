package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"go.uber.org/zap"
)

// CleanupService handles cleanup operations
type CleanupService struct {
	client     *client.Client
	logger     *zap.Logger
	tempDirs   []string // Directories to prune
	maxDiskUsagePercent float64 // Maximum disk usage percentage
}

// NewCleanupService creates a new cleanup service
func NewCleanupService(dockerHost string, logger *zap.Logger, tempDirs []string, maxDiskUsagePercent float64) (*CleanupService, error) {
	cli, err := client.NewClientWithOpts(
		client.WithHost(dockerHost),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	return &CleanupService{
		client:              cli,
		logger:              logger,
		tempDirs:            tempDirs,
		maxDiskUsagePercent: maxDiskUsagePercent,
	}, nil
}

// Close closes the Docker client
func (s *CleanupService) Close() error {
	return s.client.Close()
}

// CleanupResult represents the result of a cleanup operation
type CleanupResult struct {
	ContainersRemoved int
	ImagesRemoved     int
	SpaceFreedMB      int64
	TempDirsPruned    int
	Errors            []string
}

// RunCleanup performs all cleanup operations
func (s *CleanupService) RunCleanup(ctx context.Context) (*CleanupResult, error) {
	result := &CleanupResult{
		Errors: []string{},
	}

	s.logger.Info("Starting cleanup operation")

	// Step 1: Remove dead containers
	containersRemoved, err := s.removeDeadContainers(ctx)
	if err != nil {
		s.logger.Warn("Failed to remove dead containers", zap.Error(err))
		result.Errors = append(result.Errors, fmt.Sprintf("dead containers: %v", err))
	} else {
		result.ContainersRemoved = containersRemoved
		s.logger.Info("Removed dead containers", zap.Int("count", containersRemoved))
	}

	// Step 2: Remove dangling images
	imagesRemoved, spaceFreed, err := s.removeDanglingImages(ctx)
	if err != nil {
		s.logger.Warn("Failed to remove dangling images", zap.Error(err))
		result.Errors = append(result.Errors, fmt.Sprintf("dangling images: %v", err))
	} else {
		result.ImagesRemoved = imagesRemoved
		result.SpaceFreedMB = spaceFreed
		s.logger.Info("Removed dangling images",
			zap.Int("count", imagesRemoved),
			zap.Int64("space_freed_mb", spaceFreed),
		)
	}

	// Step 3: Prune temp directories
	tempDirsPruned, err := s.pruneTempDirs(ctx)
	if err != nil {
		s.logger.Warn("Failed to prune temp directories", zap.Error(err))
		result.Errors = append(result.Errors, fmt.Sprintf("temp dirs: %v", err))
	} else {
		result.TempDirsPruned = tempDirsPruned
		s.logger.Info("Pruned temp directories", zap.Int("count", tempDirsPruned))
	}

	// Step 4: Enforce disk quotas
	quotaEnforced, err := s.enforceDiskQuotas(ctx)
	if err != nil {
		s.logger.Warn("Failed to enforce disk quotas", zap.Error(err))
		result.Errors = append(result.Errors, fmt.Sprintf("disk quotas: %v", err))
	} else if quotaEnforced {
		s.logger.Info("Disk quotas enforced")
	}

	s.logger.Info("Cleanup operation completed",
		zap.Int("containers_removed", result.ContainersRemoved),
		zap.Int("images_removed", result.ImagesRemoved),
		zap.Int64("space_freed_mb", result.SpaceFreedMB),
		zap.Int("temp_dirs_pruned", result.TempDirsPruned),
		zap.Int("errors", len(result.Errors)),
	)

	return result, nil
}

// removeDeadContainers removes containers that are stopped or exited
func (s *CleanupService) removeDeadContainers(ctx context.Context) (int, error) {
	// List all containers (including stopped ones)
	containers, err := s.client.ContainerList(ctx, container.ListOptions{
		All: true,
		Filters: filters.NewArgs(
			filters.Arg("status", "exited"),
			filters.Arg("status", "dead"),
			filters.Arg("status", "created"), // Created but never started
		),
	})
	if err != nil {
		return 0, fmt.Errorf("failed to list containers: %w", err)
	}

	removed := 0
	for _, c := range containers {
		// Skip containers with app.id label (managed containers)
		if _, hasAppLabel := c.Labels["app.id"]; hasAppLabel {
			continue
		}

		// Remove container
		if err := s.client.ContainerRemove(ctx, c.ID, container.RemoveOptions{
			Force: true,
		}); err != nil {
			s.logger.Warn("Failed to remove container",
				zap.String("container_id", c.ID),
				zap.Error(err),
			)
			continue
		}

		removed++
		s.logger.Debug("Removed dead container", zap.String("container_id", c.ID))
	}

	return removed, nil
}

// removeDanglingImages removes dangling (untagged) images
func (s *CleanupService) removeDanglingImages(ctx context.Context) (int, int64, error) {
	// List dangling images
	filter := filters.NewArgs()
	filter.Add("dangling", "true")

	images, err := s.client.ImageList(ctx, image.ListOptions{
		Filters: filter,
	})
	if err != nil {
		return 0, 0, fmt.Errorf("failed to list images: %w", err)
	}

	removed := 0
	var totalSize int64

	for _, img := range images {
		// Skip images that are in use
		if img.Containers > 0 {
			continue
		}

		// Calculate image size
		imageSize := img.Size

		// Remove image
		_, err := s.client.ImageRemove(ctx, img.ID, image.RemoveOptions{
			Force:         false,
			PruneChildren: true,
		})
		if err != nil {
			s.logger.Warn("Failed to remove image",
				zap.String("image_id", img.ID),
				zap.Error(err),
			)
			continue
		}

		removed++
		totalSize += imageSize
		s.logger.Debug("Removed dangling image",
			zap.String("image_id", img.ID),
			zap.Int64("size_bytes", imageSize),
		)
	}

	// Convert bytes to MB
	spaceFreedMB := totalSize / (1024 * 1024)

	return removed, spaceFreedMB, nil
}

// pruneTempDirs removes old files from temp directories
func (s *CleanupService) pruneTempDirs(ctx context.Context) (int, error) {
	pruned := 0
	maxAge := 24 * time.Hour // Remove files older than 24 hours

	for _, dir := range s.tempDirs {
		prunedInDir, err := s.pruneDirectory(ctx, dir, maxAge)
		if err != nil {
			s.logger.Warn("Failed to prune directory",
				zap.String("directory", dir),
				zap.Error(err),
			)
			continue
		}
		pruned += prunedInDir
	}

	return pruned, nil
}

// pruneDirectory removes old files from a directory
func (s *CleanupService) pruneDirectory(ctx context.Context, dirPath string, maxAge time.Duration) (int, error) {
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		return 0, nil // Directory doesn't exist
	}

	pruned := 0
	cutoffTime := time.Now().Add(-maxAge)

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Remove old files
		if info.ModTime().Before(cutoffTime) {
			if err := os.Remove(path); err != nil {
				s.logger.Warn("Failed to remove old file",
					zap.String("path", path),
					zap.Error(err),
				)
				return nil // Continue with other files
			}
			pruned++
			s.logger.Debug("Removed old file", zap.String("path", path))
		}

		return nil
	})

	return pruned, err
}

// enforceDiskQuotas enforces disk usage quotas by cleaning up old data
func (s *CleanupService) enforceDiskQuotas(ctx context.Context) (bool, error) {
	// Get current disk usage
	usage, err := s.getDiskUsage(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get disk usage: %w", err)
	}

	if usage < s.maxDiskUsagePercent {
		return false, nil // No action needed
	}

	s.logger.Warn("Disk usage exceeds quota, performing aggressive cleanup",
		zap.Float64("current_usage_percent", usage),
		zap.Float64("max_usage_percent", s.maxDiskUsagePercent),
	)

	// Aggressive cleanup: remove old images and containers
	// Step 1: Remove old stopped containers (older than 7 days)
	oldContainers, err := s.removeOldContainers(ctx, 7*24*time.Hour)
	if err != nil {
		s.logger.Warn("Failed to remove old containers", zap.Error(err))
	} else {
		s.logger.Info("Removed old containers", zap.Int("count", oldContainers))
	}

	// Step 2: Remove old unused images (older than 30 days)
	oldImages, spaceFreed, err := s.removeOldImages(ctx, 30*24*time.Hour)
	if err != nil {
		s.logger.Warn("Failed to remove old images", zap.Error(err))
	} else {
		s.logger.Info("Removed old images",
			zap.Int("count", oldImages),
			zap.Int64("space_freed_mb", spaceFreed),
		)
	}

	// Step 3: Prune build cache
	if err := s.pruneBuildCache(ctx); err != nil {
		s.logger.Warn("Failed to prune build cache", zap.Error(err))
	}

	// Check if we're still over quota
	newUsage, err := s.getDiskUsage(ctx)
	if err != nil {
		return true, fmt.Errorf("failed to verify disk usage after cleanup: %w", err)
	}

	if newUsage >= s.maxDiskUsagePercent {
		s.logger.Error("Disk usage still exceeds quota after cleanup",
			zap.Float64("usage_percent", newUsage),
		)
	}

	return true, nil
}

// getDiskUsage gets the current disk usage percentage
// Note: This is a simplified implementation. For production, consider using
// a library like github.com/shirou/gopsutil/v3/disk for cross-platform support
func (s *CleanupService) getDiskUsage(ctx context.Context) (float64, error) {
	// Get current working directory to check its filesystem
	wd, err := os.Getwd()
	if err != nil {
		return 0, fmt.Errorf("failed to get working directory: %w", err)
	}

	// Calculate disk usage by walking temp directories and Docker data
	// This is a simplified approach - in production, use proper filesystem stats
	var totalSize int64
	var fileCount int64

	// Walk temp directories to estimate usage
	for _, dir := range s.tempDirs {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Skip errors
			}
			if !info.IsDir() {
				totalSize += info.Size()
				fileCount++
			}
			return nil
		})
		if err != nil && !os.IsNotExist(err) {
			s.logger.Debug("Failed to walk directory", zap.String("dir", dir), zap.Error(err))
		}
	}

	// For now, we'll use a heuristic based on temp directory sizes
	// In production, you'd want to use actual filesystem statistics
	// This is a placeholder that estimates usage based on temp directories
	// A real implementation would use syscall.Statfs (Unix) or GetDiskFreeSpaceEx (Windows)
	
	// Return a conservative estimate - if temp dirs are large, assume high usage
	// This will trigger cleanup more aggressively
	estimatedUsageMB := float64(totalSize) / (1024 * 1024)
	
	// Assume 100GB total disk, calculate percentage
	// TODO: Get actual disk size from filesystem
	assumedTotalDiskGB := 100.0
	estimatedUsagePercent := (estimatedUsageMB / 1024.0 / assumedTotalDiskGB) * 100.0

	// Cap at reasonable values
	if estimatedUsagePercent > 100 {
		estimatedUsagePercent = 100
	}
	if estimatedUsagePercent < 0 {
		estimatedUsagePercent = 0
	}

	s.logger.Debug("Estimated disk usage",
		zap.Float64("usage_percent", estimatedUsagePercent),
		zap.Int64("temp_size_mb", totalSize/(1024*1024)),
		zap.String("working_dir", wd),
	)

	return estimatedUsagePercent, nil
}

// removeOldContainers removes containers older than the specified age
func (s *CleanupService) removeOldContainers(ctx context.Context, maxAge time.Duration) (int, error) {
	containers, err := s.client.ContainerList(ctx, container.ListOptions{
		All: true,
		Filters: filters.NewArgs(
			filters.Arg("status", "exited"),
		),
	})
	if err != nil {
		return 0, fmt.Errorf("failed to list containers: %w", err)
	}

	cutoffTime := time.Now().Add(-maxAge)
	removed := 0

	for _, c := range containers {
		// Skip managed containers
		if _, hasAppLabel := c.Labels["app.id"]; hasAppLabel {
			continue
		}

		// Check container creation time
		containerJSON, err := s.client.ContainerInspect(ctx, c.ID)
		if err != nil {
			continue
		}

		// Parse creation time (it's a string in the API response)
		// ContainerInspect returns Created as a string in RFC3339 format
		createdTime, err := time.Parse(time.RFC3339Nano, containerJSON.Created)
		if err != nil {
			// Try RFC3339 format if RFC3339Nano fails
			createdTime, err = time.Parse(time.RFC3339, containerJSON.Created)
			if err != nil {
				s.logger.Warn("Failed to parse container creation time",
					zap.String("container_id", c.ID),
					zap.String("created", containerJSON.Created),
					zap.Error(err),
				)
				continue
			}
		}
		
		if createdTime.Before(cutoffTime) {
			if err := s.client.ContainerRemove(ctx, c.ID, container.RemoveOptions{
				Force: true,
			}); err != nil {
				s.logger.Warn("Failed to remove old container",
					zap.String("container_id", c.ID),
					zap.Error(err),
				)
				continue
			}
			removed++
		}
	}

	return removed, nil
}

// removeOldImages removes images older than the specified age
func (s *CleanupService) removeOldImages(ctx context.Context, maxAge time.Duration) (int, int64, error) {
	images, err := s.client.ImageList(ctx, image.ListOptions{
		All: true,
	})
	if err != nil {
		return 0, 0, fmt.Errorf("failed to list images: %w", err)
	}

	cutoffTime := time.Now().Add(-maxAge)
	removed := 0
	var totalSize int64

	for _, img := range images {
		// Skip images in use
		if img.Containers > 0 {
			continue
		}

		// Check image creation time
		if time.Unix(img.Created, 0).Before(cutoffTime) {
			imageSize := img.Size

			_, err := s.client.ImageRemove(ctx, img.ID, image.RemoveOptions{
				Force:         false,
				PruneChildren: true,
			})
			if err != nil {
				s.logger.Warn("Failed to remove old image",
					zap.String("image_id", img.ID),
					zap.Error(err),
				)
				continue
			}

			removed++
			totalSize += imageSize
		}
	}

	spaceFreedMB := totalSize / (1024 * 1024)
	return removed, spaceFreedMB, nil
}

// pruneBuildCache prunes Docker build cache
func (s *CleanupService) pruneBuildCache(ctx context.Context) error {
	// Prune build cache
	pruneReport, err := s.client.BuildCachePrune(ctx, types.BuildCachePruneOptions{
		All: true,
	})
	if err != nil {
		return fmt.Errorf("failed to prune build cache: %w", err)
	}

	s.logger.Info("Pruned build cache",
		zap.Uint64("space_freed_bytes", pruneReport.SpaceReclaimed),
	)

	return nil
}

