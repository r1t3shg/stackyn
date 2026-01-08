package services

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
)

// LogType represents the type of log
type LogType string

const (
	LogTypeBuild   LogType = "build"
	LogTypeRuntime LogType = "runtime"
)

// LogPersistenceService handles log persistence
type LogPersistenceService struct {
	logger     *zap.Logger
	storageDir string
	usePostgres bool
	// TODO: Add Postgres client when DB is ready
	maxStoragePerAppMB int64 // Maximum storage per app in MB
}

// NewLogPersistenceService creates a new log persistence service
func NewLogPersistenceService(logger *zap.Logger, storageDir string, usePostgres bool, maxStoragePerAppMB int64) *LogPersistenceService {
	logger.Info("Initializing LogPersistenceService",
		zap.String("storage_dir", storageDir),
		zap.Bool("use_postgres", usePostgres),
		zap.Int64("max_storage_per_app_mb", maxStoragePerAppMB),
	)
	return &LogPersistenceService{
		logger:            logger,
		storageDir:        storageDir,
		usePostgres:       usePostgres,
		maxStoragePerAppMB: maxStoragePerAppMB,
	}
}

// LogEntry represents a log entry
type LogEntry struct {
	AppID        string
	BuildJobID   string // For build logs
	DeploymentID string // For runtime logs
	LogType      string // String representation of LogType
	Timestamp    time.Time
	Content      string
	Size         int64 // Size in bytes
}

// PersistLog persists logs to storage (filesystem or Postgres)
func (s *LogPersistenceService) PersistLog(ctx context.Context, entry LogEntry) error {
	// Check storage limit
	if err := s.checkStorageLimit(ctx, entry.AppID, entry.Size); err != nil {
		return fmt.Errorf("storage limit exceeded: %w", err)
	}

	if s.usePostgres {
		return s.persistToPostgres(ctx, entry)
	}
	return s.persistToFilesystem(ctx, entry)
}

// PersistLogStream persists logs from a stream
// Accepts interface{} to allow different entry types from different packages
func (s *LogPersistenceService) PersistLogStream(ctx context.Context, entry interface{}, reader io.Reader) error {
	// Convert entry to LogEntry
	var logEntry LogEntry
	switch e := entry.(type) {
	case LogEntry:
		logEntry = e
	case map[string]interface{}:
		// Convert from map (used by deployment service)
		if appID, ok := e["app_id"].(string); ok {
			logEntry.AppID = appID
		}
		if deploymentID, ok := e["deployment_id"].(string); ok {
			logEntry.DeploymentID = deploymentID
		}
		if logType, ok := e["log_type"].(string); ok {
			logEntry.LogType = logType
		}
		if timestamp, ok := e["timestamp"].(time.Time); ok {
			logEntry.Timestamp = timestamp
		}
		if size, ok := e["size"].(int64); ok {
			logEntry.Size = size
		}
	default:
		// Try to extract fields via reflection or use defaults
		logEntry = LogEntry{
			LogType:   "runtime",
			Timestamp: time.Now(),
		}
	}
	// For stream persistence, we'll check limit after first chunk
	// For now, proceed with persistence
	
	if s.usePostgres {
		return s.persistStreamToPostgres(ctx, logEntry, reader)
	}
	return s.persistStreamToFilesystem(ctx, logEntry, reader)
}

// persistToFilesystem persists logs to filesystem
func (s *LogPersistenceService) persistToFilesystem(ctx context.Context, entry LogEntry) error {
	// Create directory structure: storage/{app_id}/{log_type}/{timestamp}.log
	logDir := filepath.Join(s.storageDir, entry.AppID, string(entry.LogType))
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Generate filename based on type
	var filename string
	switch LogType(entry.LogType) {
	case LogTypeBuild:
		filename = fmt.Sprintf("%s.log", entry.BuildJobID)
	case LogTypeRuntime:
		filename = fmt.Sprintf("%s.log", entry.DeploymentID)
	default:
		filename = fmt.Sprintf("%s.log", time.Now().Format("20060102-150405"))
	}

	logPath := filepath.Join(logDir, filename)

	// Append to file
	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	if _, err := file.WriteString(entry.Content); err != nil {
		return fmt.Errorf("failed to write log: %w", err)
	}

	s.logger.Debug("Persisted log to filesystem",
		zap.String("app_id", entry.AppID),
		zap.String("log_type", entry.LogType),
		zap.String("path", logPath),
	)

	return nil
}

// persistStreamToFilesystem persists logs from a stream to filesystem
func (s *LogPersistenceService) persistStreamToFilesystem(ctx context.Context, entry LogEntry, reader io.Reader) error {
	// Create directory structure
	logDir := filepath.Join(s.storageDir, entry.AppID, string(entry.LogType))
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Generate filename
	var filename string
	switch LogType(entry.LogType) {
	case LogTypeBuild:
		filename = fmt.Sprintf("%s.log", entry.BuildJobID)
	case LogTypeRuntime:
		filename = fmt.Sprintf("%s.log", entry.DeploymentID)
	default:
		filename = fmt.Sprintf("%s.log", time.Now().Format("20060102-150405"))
	}

	logPath := filepath.Join(logDir, filename)
	
	s.logger.Info("Persisting log stream to filesystem",
		zap.String("app_id", entry.AppID),
		zap.String("log_type", string(entry.LogType)),
		zap.String("deployment_id", entry.DeploymentID),
		zap.String("log_path", logPath),
		zap.String("log_dir", logDir),
		zap.String("filename", filename),
	)

	// Create or append to file
	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	// Copy from stream to file
	written, err := io.Copy(file, reader)
	if err != nil {
		s.logger.Error("Failed to write log stream",
			zap.Error(err),
			zap.String("app_id", entry.AppID),
			zap.String("log_type", string(entry.LogType)),
			zap.String("log_path", logPath),
		)
		return fmt.Errorf("failed to write log stream: %w", err)
	}

	s.logger.Info("Successfully persisted log stream to filesystem",
		zap.String("app_id", entry.AppID),
		zap.String("log_type", string(entry.LogType)),
		zap.String("log_path", logPath),
		zap.Int64("bytes_written", written),
		zap.String("deployment_id", entry.DeploymentID),
	)

	return nil
}

// persistToPostgres persists logs to Postgres (chunked)
func (s *LogPersistenceService) persistToPostgres(ctx context.Context, entry LogEntry) error {
	// TODO: Implement Postgres persistence with chunking
	// For now, log that it should use Postgres
	s.logger.Info("Postgres log persistence not yet implemented",
		zap.String("app_id", entry.AppID),
		zap.String("log_type", string(entry.LogType)),
	)
	return fmt.Errorf("Postgres log persistence not yet implemented")
}

// persistStreamToPostgres persists logs from a stream to Postgres (chunked)
func (s *LogPersistenceService) persistStreamToPostgres(ctx context.Context, entry LogEntry, reader io.Reader) error {
	// TODO: Implement Postgres stream persistence with chunking
	// For now, log that it should use Postgres
	s.logger.Info("Postgres log stream persistence not yet implemented",
		zap.String("app_id", entry.AppID),
		zap.String("log_type", string(entry.LogType)),
	)
	return fmt.Errorf("Postgres log stream persistence not yet implemented")
}

// checkStorageLimit checks if adding new logs would exceed storage limit
func (s *LogPersistenceService) checkStorageLimit(ctx context.Context, appID string, newSize int64) error {
	currentSize, err := s.getCurrentStorageSize(ctx, appID)
	if err != nil {
		return fmt.Errorf("failed to get current storage size: %w", err)
	}

	maxSizeBytes := s.maxStoragePerAppMB * 1024 * 1024
	if currentSize+newSize > maxSizeBytes {
		return fmt.Errorf("storage limit exceeded: current=%d bytes, new=%d bytes, max=%d bytes",
			currentSize, newSize, maxSizeBytes)
	}

	return nil
}

// getCurrentStorageSize gets the current storage size for an app
func (s *LogPersistenceService) getCurrentStorageSize(ctx context.Context, appID string) (int64, error) {
	if s.usePostgres {
		// TODO: Query Postgres for total size
		return 0, fmt.Errorf("Postgres storage size calculation not yet implemented")
	}

	// Calculate filesystem size
	appDir := filepath.Join(s.storageDir, appID)
	var totalSize int64

	err := filepath.Walk(appDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

	if os.IsNotExist(err) {
		return 0, nil // Directory doesn't exist yet
	}

	return totalSize, err
}

// GetLogs retrieves logs for an app
func (s *LogPersistenceService) GetLogs(ctx context.Context, appID string, logType string, limit int, offset int) ([]LogEntry, error) {
	if s.usePostgres {
		return s.getLogsFromPostgres(ctx, appID, logType, limit, offset)
	}
	return s.getLogsFromFilesystem(ctx, appID, logType, limit, offset)
}

// GetLogsByDeploymentID retrieves runtime logs for a specific deployment
func (s *LogPersistenceService) GetLogsByDeploymentID(ctx context.Context, appID string, deploymentID string) (string, error) {
	if s.usePostgres {
		return s.getLogsByDeploymentIDFromPostgres(ctx, appID, deploymentID)
	}
	return s.getLogsByDeploymentIDFromFilesystem(ctx, appID, deploymentID)
}

// getLogsByDeploymentIDFromFilesystem retrieves runtime logs for a specific deployment from filesystem
func (s *LogPersistenceService) getLogsByDeploymentIDFromFilesystem(ctx context.Context, appID string, deploymentID string) (string, error) {
	logDir := filepath.Join(s.storageDir, appID, string(LogTypeRuntime))
	logPath := filepath.Join(logDir, fmt.Sprintf("%s.log", deploymentID))
	
	s.logger.Info("Attempting to read runtime logs",
		zap.String("app_id", appID),
		zap.String("deployment_id", deploymentID),
		zap.String("log_path", logPath),
		zap.String("log_dir", logDir),
		zap.String("storage_dir", s.storageDir),
	)
	
	// Check if directory exists
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		s.logger.Warn("Log directory does not exist",
			zap.String("log_dir", logDir),
			zap.String("app_id", appID),
			zap.String("storage_dir", s.storageDir),
		)
		return "", nil
	}
	
	// List files in directory for debugging
	files, err := os.ReadDir(logDir)
	if err != nil {
		s.logger.Warn("Failed to read log directory", zap.Error(err), zap.String("log_dir", logDir))
	} else {
		s.logger.Info("Files in log directory",
			zap.String("log_dir", logDir),
			zap.Int("file_count", len(files)),
		)
		var fileNames []string
		for _, file := range files {
			fileNames = append(fileNames, file.Name())
		}
		s.logger.Info("Found log files", zap.Strings("files", fileNames))
	}
	
	// Check if target file exists before reading
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		s.logger.Warn("Log file does not exist",
			zap.String("log_path", logPath),
			zap.String("app_id", appID),
			zap.String("deployment_id", deploymentID),
		)
		return "", nil // Return empty string if file doesn't exist (no logs yet)
	}
	
	content, err := os.ReadFile(logPath)
	if err != nil {
		s.logger.Error("Failed to read log file", zap.Error(err), zap.String("log_path", logPath))
		return "", fmt.Errorf("failed to read log file: %w", err)
	}
	
	s.logger.Info("Successfully read runtime logs",
		zap.String("log_path", logPath),
		zap.Int("content_length", len(content)),
		zap.String("app_id", appID),
		zap.String("deployment_id", deploymentID),
	)
	
	return string(content), nil
}

// getLogsByDeploymentIDFromPostgres retrieves runtime logs for a specific deployment from Postgres
func (s *LogPersistenceService) getLogsByDeploymentIDFromPostgres(ctx context.Context, appID string, deploymentID string) (string, error) {
	// TODO: Implement Postgres log retrieval by deployment ID
	return "", fmt.Errorf("Postgres log retrieval by deployment ID not yet implemented")
}

// getLogsFromFilesystem retrieves logs from filesystem
func (s *LogPersistenceService) getLogsFromFilesystem(ctx context.Context, appID string, logType string, limit int, offset int) ([]LogEntry, error) {
	logDir := filepath.Join(s.storageDir, appID, logType)
	
	// Read log files
	files, err := os.ReadDir(logDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []LogEntry{}, nil
		}
		return nil, fmt.Errorf("failed to read log directory: %w", err)
	}

	// Sort by modification time (newest first)
	// TODO: Implement proper sorting

	var entries []LogEntry
	count := 0
	skipped := 0

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		// Skip to offset
		if skipped < offset {
			skipped++
			continue
		}

		if count >= limit {
			break
		}

		filePath := filepath.Join(logDir, file.Name())
		content, err := os.ReadFile(filePath)
		if err != nil {
			s.logger.Warn("Failed to read log file", zap.Error(err), zap.String("path", filePath))
			continue
		}

		info, err := file.Info()
		if err != nil {
			continue
		}

		entry := LogEntry{
			AppID:     appID,
			LogType:   logType,
			Timestamp: info.ModTime(),
			Content:   string(content),
			Size:      info.Size(),
		}

		// Extract build job ID or deployment ID from filename
		// Format: {id}.log
		if len(file.Name()) > 4 && file.Name()[len(file.Name())-4:] == ".log" {
			id := file.Name()[:len(file.Name())-4]
			if logType == string(LogTypeBuild) {
				entry.BuildJobID = id
			} else {
				entry.DeploymentID = id
			}
		}

		entries = append(entries, entry)
		count++
	}

	return entries, nil
}

// getLogsFromPostgres retrieves logs from Postgres
func (s *LogPersistenceService) getLogsFromPostgres(ctx context.Context, appID string, logType string, limit int, offset int) ([]LogEntry, error) {
	// TODO: Implement Postgres log retrieval
	return nil, fmt.Errorf("Postgres log retrieval not yet implemented")
}

// DeleteOldLogs deletes old logs to free up space
func (s *LogPersistenceService) DeleteOldLogs(ctx context.Context, appID string, before time.Time) error {
	if s.usePostgres {
		return s.deleteOldLogsFromPostgres(ctx, appID, before)
	}
	return s.deleteOldLogsFromFilesystem(ctx, appID, before)
}

// deleteOldLogsFromFilesystem deletes old log files
func (s *LogPersistenceService) deleteOldLogsFromFilesystem(ctx context.Context, appID string, before time.Time) error {
	appDir := filepath.Join(s.storageDir, appID)
	
	return filepath.Walk(appDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && info.ModTime().Before(before) {
			if err := os.Remove(path); err != nil {
				s.logger.Warn("Failed to delete old log file", zap.Error(err), zap.String("path", path))
			} else {
				s.logger.Info("Deleted old log file", zap.String("path", path))
			}
		}
		return nil
	})
}

// deleteOldLogsFromPostgres deletes old logs from Postgres
func (s *LogPersistenceService) deleteOldLogsFromPostgres(ctx context.Context, appID string, before time.Time) error {
	// TODO: Implement Postgres log deletion
	return fmt.Errorf("Postgres log deletion not yet implemented")
}

