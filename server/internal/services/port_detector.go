package services

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"go.uber.org/zap"
)

// PortDetectionResult represents the result of port detection
type PortDetectionResult struct {
	DetectedPort *int   // Port found in source code (nil if not detected or using env)
	PortSource   string // "hardcoded", "env", or "none"
	Warning      string // Warning message if hardcoded port detected
}

// PortDetector detects hardcoded ports in application source code
type PortDetector struct {
	logger *zap.Logger
}

// NewPortDetector creates a new port detector
func NewPortDetector(logger *zap.Logger) *PortDetector {
	return &PortDetector{
		logger: logger,
	}
}

// DetectPort scans source code for hardcoded port numbers
// Returns PortDetectionResult with detected port information
// This is NON-BLOCKING - never returns an error, always succeeds
func (d *PortDetector) DetectPort(repoPath string, runtime Runtime) PortDetectionResult {
	// Default result: no port detected (using env or no explicit port)
	result := PortDetectionResult{
		DetectedPort: nil,
		PortSource:   "env",
		Warning:      "",
	}

	// Skip detection for static sites and unknown runtimes
	if runtime == RuntimeStatic || runtime == RuntimeUnknown {
		d.logger.Debug("Skipping port detection for static/unknown runtime",
			zap.String("runtime", string(runtime)),
			zap.String("repo_path", repoPath),
		)
		return result
	}

	// Scan source files based on runtime
	detectedPorts := d.scanForPorts(repoPath, runtime)
	
	if len(detectedPorts) == 0 {
		// No hardcoded ports found - using env vars (recommended)
		return result
	}

	// Use the first detected port (most common case)
	// In practice, apps rarely have multiple different ports
	detectedPort := detectedPorts[0]
	
	result.DetectedPort = &detectedPort
	result.PortSource = "hardcoded"
	
	// Generate warning if port is not 8080
	if detectedPort != 8080 {
		result.Warning = fmt.Sprintf(
			"⚠️ Hardcoded port detected: %d. Stackyn uses PORT=8080 internally. We recommend using process.env.PORT (or equivalent for your runtime).",
			detectedPort,
		)
		d.logger.Info("Hardcoded port detected",
			zap.Int("detected_port", detectedPort),
			zap.String("repo_path", repoPath),
			zap.String("runtime", string(runtime)),
		)
	}

	return result
}

// scanForPorts scans source files for hardcoded port numbers
func (d *PortDetector) scanForPorts(repoPath string, runtime Runtime) []int {
	var ports []int
	portSet := make(map[int]bool) // Use map to avoid duplicates

	// Define patterns based on runtime
	var patterns []*regexp.Regexp
	var filePatterns []string

	switch runtime {
	case RuntimeNodeJS:
		// Node.js patterns: app.listen(3000), server.listen(port, ...), listen(80)
		patterns = []*regexp.Regexp{
			regexp.MustCompile(`\.listen\s*\(\s*(\d+)\s*[,)]`),           // app.listen(3000)
			regexp.MustCompile(`\.listen\s*\(\s*process\.env\.PORT\s*,\s*(\d+)\s*[,)]`), // listen(PORT, 3000) - second arg
		}
		filePatterns = []string{"*.js", "*.jsx", "*.ts", "*.tsx", "*.mjs", "*.cjs"}
		
	case RuntimePython:
		// Python patterns: app.run(port=5000), app.run(host='0.0.0.0', port=8000)
		patterns = []*regexp.Regexp{
			regexp.MustCompile(`\.run\s*\([^)]*port\s*=\s*(\d+)`),           // app.run(port=5000)
			regexp.MustCompile(`port\s*=\s*(\d+)`),                         // port=8000
			regexp.MustCompile(`listen\s*\(\s*(\d+)\s*[,)]`),              // listen(80) (Flask-SocketIO, etc.)
		}
		filePatterns = []string{"*.py"}
		
	case RuntimeGo:
		// Go patterns: http.ListenAndServe(":8080", nil), ln, err := net.Listen("tcp", ":3000")
		patterns = []*regexp.Regexp{
			regexp.MustCompile(`ListenAndServe\s*\(\s*["\']:(\d+)["\']`),  // http.ListenAndServe(":8080", nil)
			regexp.MustCompile(`Listen\s*\(\s*["\']tcp["\']\s*,\s*["\']:(\d+)["\']`), // net.Listen("tcp", ":3000")
		}
		filePatterns = []string{"*.go"}
		
	case RuntimeRuby:
		// Ruby patterns: port 3000, set :port, 3000
		patterns = []*regexp.Regexp{
			regexp.MustCompile(`port\s+(\d+)`),      // port 3000
			regexp.MustCompile(`set\s+:port\s*,\s*(\d+)`), // set :port, 3000
		}
		filePatterns = []string{"*.rb"}
		
	case RuntimeJava:
		// Java patterns: server.port=8080 (application.properties)
		patterns = []*regexp.Regexp{
			regexp.MustCompile(`server\.port\s*=\s*(\d+)`), // server.port=8080
		}
		filePatterns = []string{"*.properties", "*.yml", "*.yaml"}
		
	default:
		// For other runtimes, use generic patterns
		patterns = []*regexp.Regexp{
			regexp.MustCompile(`\.listen\s*\(\s*(\d+)\s*[,)]`),
			regexp.MustCompile(`port\s*[=:]\s*(\d+)`),
		}
		filePatterns = []string{"*.js", "*.jsx", "*.ts", "*.py", "*.go", "*.rb", "*.java"}
	}

	// Walk through repository and scan files
	err := filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Skip directories
		if info.IsDir() {
			// Skip hidden directories and common build/cache directories
			if strings.HasPrefix(info.Name(), ".") ||
				info.Name() == "node_modules" ||
				info.Name() == "__pycache__" ||
				info.Name() == ".venv" ||
				info.Name() == "venv" ||
				info.Name() == "vendor" ||
				info.Name() == "target" ||
				info.Name() == "build" ||
				info.Name() == "dist" ||
				info.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if file matches pattern
		matched := false
		ext := strings.ToLower(filepath.Ext(path))
		for _, pattern := range filePatterns {
			patternExt := strings.TrimPrefix(pattern, "*")
			if ext == patternExt {
				matched = true
				break
			}
		}

		if !matched {
			return nil
		}

		// Read file content
		content, err := os.ReadFile(path)
		if err != nil {
			return nil // Skip files that can't be read
		}

		contentStr := string(content)

		// Check for environment variable usage (good practice)
		// If file uses process.env.PORT or similar, skip hardcoded detection for this file
		envPatterns := []string{
			"process.env.PORT",
			"os.environ.get('PORT'",
			"os.getenv('PORT'",
			"os.Getenv(\"PORT\"",
			"ENV['PORT']",
			"ENV[\"PORT\"]",
		}
		usesEnvPort := false
		for _, envPattern := range envPatterns {
			if strings.Contains(contentStr, envPattern) {
				usesEnvPort = true
				break
			}
		}

		// Apply regex patterns
		for _, pattern := range patterns {
			matches := pattern.FindAllStringSubmatch(contentStr, -1)
			for _, match := range matches {
				if len(match) >= 2 {
					if port, err := strconv.Atoi(match[1]); err == nil {
						// Validate port range (1-65535)
						if port >= 1 && port <= 65535 {
							// Only add if not using env port OR if it's clearly hardcoded (like listen(80))
							if !usesEnvPort || strings.Contains(match[0], "listen(") {
								portSet[port] = true
							}
						}
					}
				}
			}
		}

		return nil
	})

	if err != nil {
		d.logger.Warn("Error scanning for ports", zap.Error(err), zap.String("repo_path", repoPath))
	}

	// Convert set to slice
	for port := range portSet {
		ports = append(ports, port)
	}

	return ports
}

