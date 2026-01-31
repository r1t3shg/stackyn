package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
)

// ConstraintsService enforces MVP constraints
type ConstraintsService struct {
	logger       *zap.Logger
	maxBuildTime int // Maximum build time in minutes
}

// NewConstraintsService creates a new constraints service
func NewConstraintsService(logger *zap.Logger, maxBuildTimeMinutes int) *ConstraintsService {
	return &ConstraintsService{
		logger:       logger,
		maxBuildTime: maxBuildTimeMinutes,
	}
}

// ConstraintError represents a constraint violation
type ConstraintError struct {
	Constraint string
	Message    string
	Details    string
}

func (e *ConstraintError) Error() string {
	return e.Message
}

// IsConstraintError checks if an error is a ConstraintError
func IsConstraintError(err error) bool {
	_, ok := err.(*ConstraintError)
	return ok
}

// GetConstraintError extracts ConstraintError from error
func GetConstraintError(err error) (*ConstraintError, bool) {
	constraintErr, ok := err.(*ConstraintError)
	return constraintErr, ok
}

// ValidateRepoURL validates that the repository URL is a public GitHub repository
func (s *ConstraintsService) ValidateRepoURL(ctx context.Context, repoURL string) error {
	// Must be a GitHub repository
	if !strings.Contains(repoURL, "github.com") {
		return &ConstraintError{
			Constraint: "github_only",
			Message:    "Only public GitHub repositories are supported. Please provide a GitHub repository URL.",
			Details:    fmt.Sprintf("Repository URL must be from github.com. Provided: %s", repoURL),
		}
	}

	// Must be HTTPS (not SSH)
	if strings.HasPrefix(repoURL, "git@") || strings.HasPrefix(repoURL, "ssh://") {
		return &ConstraintError{
			Constraint: "github_only",
			Message:    "Only HTTPS GitHub URLs are supported. SSH URLs are not allowed.",
			Details:    fmt.Sprintf("Please use HTTPS URL format: https://github.com/owner/repo. Provided: %s", repoURL),
		}
	}

	// Must be public (this is validated in GitService, but we check format here)
	if !strings.HasPrefix(repoURL, "https://github.com/") {
		return &ConstraintError{
			Constraint: "github_only",
			Message:    "Invalid GitHub repository URL format.",
			Details:    fmt.Sprintf("URL must be in format: https://github.com/owner/repo. Provided: %s", repoURL),
		}
	}

	return nil
}

// ValidateNoDockerCompose checks that the repository does not contain docker-compose files
func (s *ConstraintsService) ValidateNoDockerCompose(ctx context.Context, repoPath string) error {
	dockerComposeFiles := []string{
		"docker-compose.yml",
		"docker-compose.yaml",
		"docker-compose.yaml",
		"compose.yml",
		"compose.yaml",
		".docker-compose.yml",
		".docker-compose.yaml",
	}

	for _, filename := range dockerComposeFiles {
		filePath := filepath.Join(repoPath, filename)
		if _, err := os.Stat(filePath); err == nil {
			return &ConstraintError{
				Constraint: "no_docker_compose",
				Message:    "Docker Compose files are not supported in MVP. Please remove docker-compose.yml or use a single-container setup.",
				Details:    fmt.Sprintf("Found docker-compose file: %s. MVP supports single-container applications only.", filename),
			}
		}
	}

	return nil
}

// ValidateSingleContainer ensures the application is configured for single container only
// This is validated by checking for multi-container configurations
func (s *ConstraintsService) ValidateSingleContainer(ctx context.Context, repoPath string) error {
	// Check for Kubernetes manifests
	k8sFiles := []string{
		"k8s",
		"kubernetes",
		".k8s",
		".kubernetes",
	}

	for _, dir := range k8sFiles {
		dirPath := filepath.Join(repoPath, dir)
		if info, err := os.Stat(dirPath); err == nil && info.IsDir() {
			return &ConstraintError{
				Constraint: "single_container_only",
				Message:    "Kubernetes configurations are not supported. MVP supports single-container applications only.",
				Details:    fmt.Sprintf("Found Kubernetes directory: %s. Please use a single Dockerfile instead.", dir),
			}
		}
	}

	// Check for multiple Dockerfiles (indicating multi-stage or multi-service)
	dockerfiles := []string{
		"Dockerfile",
		"dockerfile",
		".dockerfile",
	}

	dockerfileCount := 0
	for _, filename := range dockerfiles {
		filePath := filepath.Join(repoPath, filename)
		if _, err := os.Stat(filePath); err == nil {
			dockerfileCount++
		}
	}

	// Check for Dockerfile.* variants (e.g., Dockerfile.prod, Dockerfile.dev)
	matches, _ := filepath.Glob(filepath.Join(repoPath, "Dockerfile.*"))
	if len(matches) > 0 {
		return &ConstraintError{
			Constraint: "single_container_only",
			Message:    "Multiple Dockerfiles are not supported. MVP supports single-container applications only.",
			Details:    fmt.Sprintf("Found multiple Dockerfile variants. Please use a single Dockerfile for your application."),
		}
	}

	return nil
}

// ValidateNoBackgroundWorkers checks that the application does not require background workers
func (s *ConstraintsService) ValidateNoBackgroundWorkers(ctx context.Context, repoPath string) error {
	// Check for common background worker indicators
	workerIndicators := []string{
		"worker",
		"workers",
		"background",
		"queue",
		"celery",
		"sidekiq",
		"bull",
		"agenda",
		"cron",
		"schedule",
	}

	// Check package.json for worker scripts
	packageJsonPath := filepath.Join(repoPath, "package.json")
	if _, err := os.Stat(packageJsonPath); err == nil {
		content, err := os.ReadFile(packageJsonPath)
		if err == nil {
			contentStr := strings.ToLower(string(content))
			for _, indicator := range workerIndicators {
				if strings.Contains(contentStr, indicator) {
					// Check if it's in scripts section (more likely to be a worker)
					if strings.Contains(contentStr, `"scripts"`) && strings.Contains(contentStr, indicator) {
						return &ConstraintError{
							Constraint: "no_background_workers",
							Message:    "Background workers are not supported in MVP. Please use a single HTTP application.",
							Details:    fmt.Sprintf("Found worker-related configuration in package.json. MVP supports HTTP-only applications."),
						}
					}
				}
			}
		}
	}

	// Check for Procfile (common in Heroku-style deployments with workers)
	procfilePath := filepath.Join(repoPath, "Procfile")
	if _, err := os.Stat(procfilePath); err == nil {
		content, err := os.ReadFile(procfilePath)
		if err == nil {
			contentStr := strings.ToLower(string(content))
			// Check for worker processes
			if strings.Contains(contentStr, "worker:") || strings.Contains(contentStr, "worker ") {
				return &ConstraintError{
					Constraint: "no_background_workers",
					Message:    "Background workers defined in Procfile are not supported. MVP supports single HTTP applications only.",
					Details:    "Found worker process in Procfile. Please remove worker processes and use HTTP-only application.",
				}
			}
		}
	}

	// Check for common worker configuration files
	workerConfigFiles := []string{
		"celery.py",
		"celeryconfig.py",
		"sidekiq.rb",
		"worker.js",
		"worker.ts",
		"workers.js",
		"workers.ts",
	}

	for _, filename := range workerConfigFiles {
		filePath := filepath.Join(repoPath, filename)
		if _, err := os.Stat(filePath); err == nil {
			return &ConstraintError{
				Constraint: "no_background_workers",
				Message:    "Background worker configurations are not supported. MVP supports HTTP-only applications.",
				Details:    fmt.Sprintf("Found worker configuration file: %s. Please remove background workers.", filename),
			}
		}
	}

	return nil
}

// ValidateHTTPOnly ensures the application is HTTP-only (no SSL/HTTPS requirements)
func (s *ConstraintsService) ValidateHTTPOnly(ctx context.Context, repoPath string) error {
	// Check for SSL/HTTPS configuration files
	sslFiles := []string{
		"ssl",
		".ssl",
		"certificates",
		"certs",
		".certs",
		"tls",
		".tls",
	}

	for _, dir := range sslFiles {
		dirPath := filepath.Join(repoPath, dir)
		if info, err := os.Stat(dirPath); err == nil && info.IsDir() {
			return &ConstraintError{
				Constraint: "http_only",
				Message:    "SSL/HTTPS certificates are not supported in MVP. Applications must use HTTP only.",
				Details:    fmt.Sprintf("Found SSL/certificate directory: %s. MVP supports HTTP-only applications.", dir),
			}
		}
	}

	// Check for SSL-related configuration in common config files
	configFiles := []string{
		"config.json",
		"config.yml",
		"config.yaml",
		".env.example",
		".env",
	}

	for _, filename := range configFiles {
		filePath := filepath.Join(repoPath, filename)
		if _, err := os.Stat(filePath); err == nil {
			content, err := os.ReadFile(filePath)
			if err == nil {
				contentStr := strings.ToLower(string(content))
				// Check for SSL/HTTPS configuration
				if strings.Contains(contentStr, "ssl") ||
					strings.Contains(contentStr, "https") ||
					strings.Contains(contentStr, "tls") ||
					strings.Contains(contentStr, "certificate") {
					// Only error if it's a required configuration
					if strings.Contains(contentStr, "ssl=true") ||
						strings.Contains(contentStr, "https=true") ||
						strings.Contains(contentStr, "require_ssl") {
						return &ConstraintError{
							Constraint: "http_only",
							Message:    "SSL/HTTPS requirements are not supported. MVP supports HTTP-only applications.",
							Details:    fmt.Sprintf("Found SSL/HTTPS requirement in %s. Please remove SSL requirements.", filename),
						}
					}
				}
			}
		}
	}

	return nil
}

// ValidateBuildTime checks if build time is within limits
func (s *ConstraintsService) ValidateBuildTime(ctx context.Context, buildTimeMinutes int) error {
	if buildTimeMinutes > s.maxBuildTime {
		return &ConstraintError{
			Constraint: "max_build_time",
			Message:    fmt.Sprintf("Build time exceeded maximum allowed time of %d minutes. Please optimize your build process.", s.maxBuildTime),
			Details:    fmt.Sprintf("Build took %d minutes, but maximum allowed is %d minutes.", buildTimeMinutes, s.maxBuildTime),
		}
	}

	return nil
}

// ValidateAllConstraints validates all MVP constraints for a repository
func (s *ConstraintsService) ValidateAllConstraints(ctx context.Context, repoURL, repoPath string) error {
	// Validate repository URL
	if err := s.ValidateRepoURL(ctx, repoURL); err != nil {
		return err
	}

	// Validate no docker-compose
	if err := s.ValidateNoDockerCompose(ctx, repoPath); err != nil {
		return err
	}

	// Validate single container
	if err := s.ValidateSingleContainer(ctx, repoPath); err != nil {
		return err
	}

	// Validate no background workers
	if err := s.ValidateNoBackgroundWorkers(ctx, repoPath); err != nil {
		return err
	}

	// Validate HTTP only
	if err := s.ValidateHTTPOnly(ctx, repoPath); err != nil {
		return err
	}

	return nil
}
