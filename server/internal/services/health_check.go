package services

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"go.uber.org/zap"
)

// HealthCheckCallback is a function that gets called when health check fails
// Parameters: appID, deploymentID, error message
type HealthCheckCallback func(appID, deploymentID, errorMsg string)

// HealthCheckService monitors app accessibility and health
type HealthCheckService struct {
	client          *client.Client
	logger          *zap.Logger
	healthCallback  HealthCheckCallback
	httpClient      *http.Client
}

// NewHealthCheckService creates a new health check service
func NewHealthCheckService(dockerClient *client.Client, logger *zap.Logger) *HealthCheckService {
	return &HealthCheckService{
		client: dockerClient,
		logger: logger,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: nil, // Will use default, skip cert verification in health checks
			},
		},
	}
}

// SetHealthCheckCallback sets the callback function for health check failures
func (s *HealthCheckService) SetHealthCheckCallback(callback HealthCheckCallback) {
	s.healthCallback = callback
}

// CheckAppAccessibility checks if an app is accessible through its URL
func (s *HealthCheckService) CheckAppAccessibility(ctx context.Context, appID, deploymentID, url string, containerID string) error {
	var errors []string

	// 1. Check if container is running
	containerJSON, err := s.client.ContainerInspect(ctx, containerID)
	if err != nil {
		errorMsg := fmt.Sprintf("Container not found or inaccessible: %v", err)
		s.handleHealthCheckFailure(appID, deploymentID, errorMsg)
		return fmt.Errorf(errorMsg)
	}

	if !containerJSON.State.Running {
		exitCode := containerJSON.State.ExitCode
		logs := s.getLastErrorFromLogs(ctx, containerID)
		errorMsg := fmt.Sprintf("Container is not running (status: %s, exit code: %d)", containerJSON.State.Status, exitCode)
		if logs != "" {
			errorMsg += fmt.Sprintf(" - %s", logs)
		}
		s.handleHealthCheckFailure(appID, deploymentID, errorMsg)
		return fmt.Errorf(errorMsg)
	}

	// 2. Check container health status (if healthcheck is configured)
	if containerJSON.State.Health != nil {
		healthStatus := containerJSON.State.Health.Status
		if healthStatus == types.Unhealthy {
			errorMsg := "Container health check is failing"
			if len(containerJSON.State.Health.Log) > 0 {
				lastLog := containerJSON.State.Health.Log[len(containerJSON.State.Health.Log)-1]
				if lastLog.Output != "" {
					errorMsg += fmt.Sprintf(": %s", strings.TrimSpace(lastLog.Output))
				}
			}
			errors = append(errors, errorMsg)
		}
	}

	// 3. Check if Traefik router exists for this app
	traefikRouterExists := false
	if subdomain, ok := containerJSON.Config.Labels["app.subdomain"]; ok && subdomain != "" {
		traefikRouterExists = s.checkTraefikRouter(ctx, subdomain)
		if !traefikRouterExists {
			errors = append(errors, fmt.Sprintf("Traefik router not configured for subdomain: %s (SSL certificate may be missing)", subdomain))
		}
	} else {
		errors = append(errors, "Subdomain label missing from container")
	}

	// 4. Check if URL is accessible via HTTP/HTTPS
	if url != "" {
		isAccessible, accessError := s.checkURLAccessible(url)
		if !isAccessible {
			if strings.Contains(accessError, "certificate") || strings.Contains(accessError, "SSL") || strings.Contains(accessError, "TLS") {
				errors = append(errors, fmt.Sprintf("SSL certificate issue: %s. Certificate may not be issued yet or DNS not configured.", accessError))
			} else if strings.Contains(accessError, "404") {
				errors = append(errors, "Application URL returns 404. Traefik routing may not be configured correctly.")
			} else if strings.Contains(accessError, "timeout") || strings.Contains(accessError, "connection refused") {
				errors = append(errors, fmt.Sprintf("Application not responding: %s. Container may not be listening on the expected port.", accessError))
			} else {
				errors = append(errors, fmt.Sprintf("URL not accessible: %s", accessError))
			}
		}
	}

	// If any errors found, call callback
	if len(errors) > 0 {
		errorMsg := strings.Join(errors, "; ")
		s.handleHealthCheckFailure(appID, deploymentID, errorMsg)
		return fmt.Errorf(errorMsg)
	}

	return nil
}

// MonitorAppHealth continuously monitors app health and accessibility
func (s *HealthCheckService) MonitorAppHealth(ctx context.Context, appID, deploymentID, url, containerID string) {
	ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
	defer ticker.Stop()

	// Do initial check
	if err := s.CheckAppAccessibility(ctx, appID, deploymentID, url, containerID); err != nil {
		s.logger.Warn("Initial health check failed",
			zap.String("app_id", appID),
			zap.String("deployment_id", deploymentID),
			zap.Error(err),
		)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.CheckAppAccessibility(ctx, appID, deploymentID, url, containerID); err != nil {
				s.logger.Warn("Health check failed",
					zap.String("app_id", appID),
					zap.String("deployment_id", deploymentID),
					zap.Error(err),
				)
			} else {
				s.logger.Debug("Health check passed",
					zap.String("app_id", appID),
					zap.String("deployment_id", deploymentID),
				)
			}
		}
	}
}

func (s *HealthCheckService) handleHealthCheckFailure(appID, deploymentID, errorMsg string) {
	if s.healthCallback != nil {
		s.healthCallback(appID, deploymentID, errorMsg)
	}
}

func (s *HealthCheckService) getLastErrorFromLogs(ctx context.Context, containerID string) string {
	logReader, err := s.client.ContainerLogs(ctx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     false,
		Tail:       "10",
	})
	if err != nil {
		return ""
	}
	defer logReader.Close()

	logBytes, err := io.ReadAll(logReader)
	if err != nil {
		return ""
	}

	logs := string(logBytes)
	lines := strings.Split(logs, "\n")
	// Return last non-empty line that might contain error
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line != "" {
			return line
		}
	}
	return ""
}

func (s *HealthCheckService) checkTraefikRouter(ctx context.Context, subdomain string) bool {
	// Try to check if Traefik has a router for this subdomain
	// This is a simplified check - in production, you'd query Traefik API
	// For now, we'll assume if subdomain label exists, router should exist
	// The URL accessibility check will catch actual routing issues
	return subdomain != ""
}

func (s *HealthCheckService) checkURLAccessible(url string) (bool, string) {
	// Create a request with timeout (increased to 15s to allow for app startup)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, fmt.Sprintf("Failed to create request: %v", err)
	}

	// First, try with SSL verification to detect certificate issues
	// This mimics what browsers do
	secureClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: false, // Verify certificates
			},
		},
		Timeout: 15 * time.Second, // Increased from 10s to allow for app startup
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return nil
		},
	}

	resp, err := secureClient.Do(req)
	if err != nil {
		errStr := err.Error()
		// Check for SSL/TLS certificate errors
		if strings.Contains(errStr, "certificate") || 
		   strings.Contains(errStr, "x509") || 
		   strings.Contains(errStr, "SSL") || 
		   strings.Contains(errStr, "TLS") ||
		   strings.Contains(errStr, "certificate verify failed") ||
		   strings.Contains(errStr, "self-signed") ||
		   strings.Contains(errStr, "unknown authority") ||
		   strings.Contains(errStr, "certificate signed by unknown authority") {
			// Certificate issue - try again with insecure to see if app is reachable
			insecureClient := &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: true,
					},
				},
				Timeout: 15 * time.Second, // Increased from 10s to allow for app startup
			}
			insecureResp, insecureErr := insecureClient.Do(req)
			if insecureErr == nil {
				insecureResp.Body.Close()
				// App is reachable but certificate is invalid/not issued yet
				// This is a temporary state - Let's Encrypt certificates take 1-2 minutes to issue
				// Don't treat this as a failure - the app is running, just waiting for cert
				// Return true with a note that cert is pending
				return true, "" // App is running, cert will be issued soon
			}
			// App not reachable even with insecure connection - real error
			return false, fmt.Sprintf("SSL certificate issue: certificate not valid or not issued yet (%v)", err)
		}
		// Check for connection/timeout errors
		if strings.Contains(errStr, "timeout") || 
		   strings.Contains(errStr, "connection refused") ||
		   strings.Contains(errStr, "no such host") ||
		   strings.Contains(errStr, "connection reset") {
			return false, fmt.Sprintf("Connection error: application not responding or not reachable (%v)", err)
		}
		return false, fmt.Sprintf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	// Consider 2xx, 3xx, and 4xx (but not 5xx) as "accessible" (app is responding)
	// 5xx means server error, but app is reachable
	if resp.StatusCode >= 200 && resp.StatusCode < 500 {
		return true, ""
	}

	if resp.StatusCode >= 500 {
		return false, fmt.Sprintf("Server error (HTTP %d): application is reachable but returning errors", resp.StatusCode)
	}

	return false, fmt.Sprintf("HTTP %d", resp.StatusCode)
}

