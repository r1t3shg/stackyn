// Package gitrepo provides Git repository cloning functionality.
// It handles cloning repositories from Git URLs with support for:
//   - Specific branch selection
//   - Shallow cloning (depth=1) for faster operations
//   - Dockerfile validation
//
// The cloner creates isolated directories for each deployment
// to avoid conflicts between concurrent deployments.
package gitrepo

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// IsWorkerApp checks if the Dockerfile indicates this is a worker/background process
// Returns true if worker patterns are found, false otherwise
// This function is conservative - it only flags apps that are clearly workers
func IsWorkerApp(repoPath string) bool {
	dockerfilePath := filepath.Join(repoPath, "Dockerfile")
	
	file, err := os.Open(dockerfilePath)
	if err != nil {
		return false
	}
	defer file.Close()

	// Read all lines into memory
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, strings.TrimSpace(strings.ToLower(scanner.Text())))
	}
	
	if err := scanner.Err(); err != nil {
		log.Printf("[GIT] WARNING - Error reading Dockerfile: %v", err)
		return false
	}

	// First, check for positive indicators that this IS a web server
	// If we find these, it's definitely not a worker
	hasExpose := false
	hasWebServer := false
	hasPortEnv := false
	
	// Web server indicators
	webServerIndicators := []string{
		"node server",
		"node app",
		"node index",
		"npm start",
		"npm run start",
		"python app.py",
		"python main.py",
		"python server.py",
		"uvicorn",
		"gunicorn",
		"flask run",
		"python -m flask",
		"python -m uvicorn",
		"serve",
		"http-server",
	}
	
	// Check all lines for web server indicators
	for _, line := range lines {
		// Check for EXPOSE directive (indicates web server)
		if strings.HasPrefix(line, "expose") {
			hasExpose = true
			log.Printf("[GIT] Found EXPOSE directive - this is likely a web server, not a worker")
		}
		
		// Check for ENV PORT (indicates web server)
		if strings.HasPrefix(line, "env") && strings.Contains(line, "port") {
			hasPortEnv = true
			log.Printf("[GIT] Found ENV PORT - this is likely a web server, not a worker")
		}
		
		// Check for common web server commands in CMD/ENTRYPOINT
		if strings.HasPrefix(line, "cmd") || strings.HasPrefix(line, "entrypoint") {
			for _, indicator := range webServerIndicators {
				if strings.Contains(line, indicator) {
					hasWebServer = true
					log.Printf("[GIT] Found web server command '%s' - this is not a worker", indicator)
					break
				}
			}
		}
	}
	
	// If we found clear web server indicators, it's NOT a worker
	if hasExpose || hasWebServer || hasPortEnv {
		return false
	}
	
	// Second pass: check for worker-specific patterns in CMD/ENTRYPOINT only
	// We only check CMD and ENTRYPOINT (not RUN) because RUN is for build-time setup
	// Specific worker patterns that indicate background processing
	workerPatterns := []string{
		"celery worker",
		"celery -a",
		"sidekiq",
		"bull queue",
		"queue:work",
		"queue:listen",
		"worker:start",
		"worker start",
		"background worker",
		"cron",
		"/app/worker", // Common pattern for worker binaries
	}
	
	for _, line := range lines {
		// Only check CMD and ENTRYPOINT lines (not RUN or other directives)
		if strings.HasPrefix(line, "cmd") || strings.HasPrefix(line, "entrypoint") {
			// Extract the command part (everything after CMD/ENTRYPOINT)
			commandPart := ""
			if strings.HasPrefix(line, "cmd") {
				commandPart = strings.TrimSpace(line[3:])
			} else if strings.HasPrefix(line, "entrypoint") {
				commandPart = strings.TrimSpace(line[10:])
			}
			
			// Check for worker patterns in the actual command
			for _, pattern := range workerPatterns {
				if strings.Contains(commandPart, pattern) {
					log.Printf("[GIT] Detected worker app pattern '%s' in Dockerfile command", pattern)
					return true
				}
			}
		}
	}
	
	return false
}

type Cloner struct {
	WorkDir string
}

func NewCloner(workDir string) *Cloner {
	return &Cloner{WorkDir: workDir}
}

func (c *Cloner) Clone(repoURL string, deploymentID int, branch string) (string, error) {
	repoDir := filepath.Join(c.WorkDir, fmt.Sprintf("deployment-%d", deploymentID))
	log.Printf("[GIT] Cloning repository - URL: %s, Branch: %s, Target: %s", repoURL, branch, repoDir)

	// Remove directory if it exists
	if err := os.RemoveAll(repoDir); err != nil {
		log.Printf("[GIT] ERROR - Failed to clean directory %s: %v", repoDir, err)
		return "", fmt.Errorf("failed to clean directory: %w", err)
	}

	// Clone repository with specific branch
	// First clone the repository (shallow clone for the specific branch)
	log.Printf("[GIT] Executing: git clone --branch %s --single-branch --depth 1 %s %s", branch, repoURL, repoDir)
	cmd := exec.Command("git", "clone", "--branch", branch, "--single-branch", "--depth", "1", repoURL, repoDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("[GIT] ERROR - Clone failed: %v, Output: %s", err, string(output))
		return "", fmt.Errorf("git clone failed: %w, output: %s", err, string(output))
	}

	log.Printf("[GIT] Repository cloned successfully to: %s", repoDir)
	return repoDir, nil
}

// CheckDockerfile checks if a Dockerfile exists in the repository directory
func CheckDockerfile(repoPath string) error {
	dockerfilePath := filepath.Join(repoPath, "Dockerfile")
	log.Printf("[GIT] Checking for Dockerfile at: %s", dockerfilePath)

	// Check if Dockerfile exists
	if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
		log.Printf("[GIT] ERROR - Dockerfile not found at: %s", dockerfilePath)
		return fmt.Errorf("dockerfile not found in repository root directory")
	}

	log.Printf("[GIT] Dockerfile found successfully")
	return nil
}

// CheckDockerCompose checks if a docker-compose.yml file exists in the repository root.
// If found, it returns an error indicating that multi-container apps are not supported.
// This function checks for common Docker Compose file names (case-sensitive and case-insensitive).
func CheckDockerCompose(repoPath string) error {
	// Check for exact case matches first (most common)
	dockerComposePaths := []string{
		"docker-compose.yml",
		"docker-compose.yaml",
		"compose.yml",
		"compose.yaml",
	}

	log.Printf("[GIT] Checking for Docker Compose files in: %s", repoPath)
	
	// Check for exact case matches
	for _, composeFile := range dockerComposePaths {
		composePath := filepath.Join(repoPath, composeFile)
		fileInfo, err := os.Stat(composePath)
		if err == nil {
			// File exists and is not a directory
			if !fileInfo.IsDir() {
				log.Printf("[GIT] ERROR - Docker Compose file detected at: %s (size: %d bytes)", composePath, fileInfo.Size())
				return fmt.Errorf("docker compose file found: %s", composeFile)
			}
		} else if !os.IsNotExist(err) {
			// Some other error occurred (permissions, etc.)
			log.Printf("[GIT] WARNING - Error checking for Docker Compose file at %s: %v", composePath, err)
		}
	}

	// Also check directory contents for case-insensitive matches (for case-insensitive file systems)
	// This is a fallback to catch files like "Docker-Compose.yml" or "DOCKER-COMPOSE.YML"
	entries, err := os.ReadDir(repoPath)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			fileName := strings.ToLower(entry.Name())
			for _, composeFile := range dockerComposePaths {
				if fileName == strings.ToLower(composeFile) && fileName != entry.Name() {
					// Case-insensitive match found (different case)
					composePath := filepath.Join(repoPath, entry.Name())
					log.Printf("[GIT] ERROR - Docker Compose file detected (case-insensitive match) at: %s", composePath)
					return fmt.Errorf("docker compose file found: %s", entry.Name())
				}
			}
		}
	} else {
		log.Printf("[GIT] WARNING - Could not read directory %s to check for Docker Compose files: %v", repoPath, err)
	}

	log.Printf("[GIT] No Docker Compose files found - single container app confirmed")
	return nil
}

// EnsurePackageLock handles the case where package.json exists but package-lock.json doesn't.
// This fixes the common issue where Dockerfiles use `npm ci` but the lock file is missing.
// It tries two approaches:
//   1. First, try to generate package-lock.json using npm (if Node.js is available)
//   2. If that fails, modify the Dockerfile to use `npm install` instead of `npm ci`
func EnsurePackageLock(repoPath string) error {
	packageJSONPath := filepath.Join(repoPath, "package.json")
	packageLockPath := filepath.Join(repoPath, "package-lock.json")
	dockerfilePath := filepath.Join(repoPath, "Dockerfile")

	// Check if package.json exists
	if _, err := os.Stat(packageJSONPath); os.IsNotExist(err) {
		log.Printf("[GIT] No package.json found, skipping package-lock.json check")
		return nil
	}

	// Check if package-lock.json already exists
	if _, err := os.Stat(packageLockPath); err == nil {
		log.Printf("[GIT] package-lock.json already exists, no action needed")
		return nil
	}

	log.Printf("[GIT] package.json found but package-lock.json missing")

	// Try to generate package-lock.json using npm (if Node.js is available)
	if err := generatePackageLock(repoPath); err == nil {
		log.Printf("[GIT] package-lock.json generated successfully using npm")
		return nil
	}

	log.Printf("[GIT] Could not generate package-lock.json, modifying Dockerfile to use 'npm install' instead of 'npm ci'")
	
	// Fallback: modify Dockerfile to use npm install instead of npm ci
	return fixDockerfileNpmCi(repoPath, dockerfilePath)
}

// generatePackageLock attempts to generate package-lock.json using npm
func generatePackageLock(repoPath string) error {
	// Check if npm is available
	if _, err := exec.LookPath("npm"); err != nil {
		return fmt.Errorf("npm not found in PATH")
	}

	log.Printf("[GIT] Attempting to generate package-lock.json using npm...")
	cmd := exec.Command("npm", "install", "--package-lock-only")
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("npm install failed: %w, output: %s", err, string(output))
	}
	return nil
}

// fixDockerfileNpmCi modifies the Dockerfile to replace `npm ci` with `npm install`
// when package-lock.json is missing
func fixDockerfileNpmCi(repoPath, dockerfilePath string) error {
	// Read Dockerfile
	file, err := os.Open(dockerfilePath)
	if err != nil {
		return fmt.Errorf("failed to open Dockerfile: %w", err)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	modified := false

	for scanner.Scan() {
		line := scanner.Text()
		// Check if line contains `npm ci` (case-insensitive, handles variations)
		lowerLine := strings.ToLower(line)
		if strings.Contains(lowerLine, "npm ci") || strings.Contains(lowerLine, "npmci") {
			// Replace npm ci with npm install
			// Preserve the original formatting and any flags
			originalLine := line
			line = strings.ReplaceAll(line, "npm ci", "npm install")
			line = strings.ReplaceAll(line, "npmci", "npm install")
			line = strings.ReplaceAll(line, "npm  ci", "npm install")
			// Also handle case variations
			line = strings.ReplaceAll(line, "NPM CI", "npm install")
			line = strings.ReplaceAll(line, "Npm Ci", "npm install")
			
			if line != originalLine {
				log.Printf("[GIT] Modified Dockerfile line: %s -> %s", originalLine, line)
				modified = true
			}
		}
		lines = append(lines, line)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read Dockerfile: %w", err)
	}

	if !modified {
		log.Printf("[GIT] Dockerfile does not contain 'npm ci', no modification needed")
		return nil
	}

	// Write modified Dockerfile
	if err := os.WriteFile(dockerfilePath, []byte(strings.Join(lines, "\n")), 0644); err != nil {
		return fmt.Errorf("failed to write modified Dockerfile: %w", err)
	}

	log.Printf("[GIT] Dockerfile modified successfully to use 'npm install' instead of 'npm ci'")
	return nil
}

// DetectPortFromDockerfile attempts to detect the port from the Dockerfile's EXPOSE directive,
// ENV PORT variable, or by checking package.json and source files for Node.js apps.
// Returns the first port found, or attempts to detect from common patterns, or 8080 as default.
func DetectPortFromDockerfile(repoPath string) int {
	dockerfilePath := filepath.Join(repoPath, "Dockerfile")
	
	file, err := os.Open(dockerfilePath)
	if err != nil {
		log.Printf("[GIT] WARNING - Failed to open Dockerfile for port detection: %v, trying alternative methods", err)
		return detectPortFromPackageJSON(repoPath)
	}
	defer file.Close()

	// Regex patterns for port detection
	exposeRegex := regexp.MustCompile(`(?i)^\s*EXPOSE\s+(\d+)`)
	envPortRegex := regexp.MustCompile(`(?i)^\s*ENV\s+PORT\s*=\s*(\d+)`)
	// Python patterns: uvicorn --port 8000, gunicorn -b :8000
	uvicornRegex := regexp.MustCompile(`(?i)uvicorn.*--port\s+(\d+)`)
	gunicornRegex := regexp.MustCompile(`(?i)gunicorn.*-b\s+[:\d.]*:(\d+)`)
	
	scanner := bufio.NewScanner(file)
	var detectedPort int
	foundExpose := false
	
	for scanner.Scan() {
		line := scanner.Text()
		
		// First, check for EXPOSE directive (highest priority)
		matches := exposeRegex.FindStringSubmatch(line)
		if len(matches) > 1 {
			port, err := strconv.Atoi(matches[1])
			if err == nil && port > 0 && port < 65536 {
				log.Printf("[GIT] Detected port %d from Dockerfile EXPOSE directive", port)
				return port
			}
		}
		
		// Check for ENV PORT=3000 (common in Node.js apps) or ENV PORT=8000 (Python)
		if !foundExpose {
			envMatches := envPortRegex.FindStringSubmatch(line)
			if len(envMatches) > 1 {
				port, err := strconv.Atoi(envMatches[1])
				if err == nil && port > 0 && port < 65536 {
					detectedPort = port
					log.Printf("[GIT] Detected port %d from Dockerfile ENV PORT directive", port)
				}
			}
			
			// Check for uvicorn command with --port
			if uvicornMatches := uvicornRegex.FindStringSubmatch(line); len(uvicornMatches) > 1 {
				port, err := strconv.Atoi(uvicornMatches[1])
				if err == nil && port > 0 && port < 65536 {
					detectedPort = port
					log.Printf("[GIT] Detected port %d from Dockerfile uvicorn command", port)
				}
			}
			
			// Check for gunicorn command with -b :port
			if gunicornMatches := gunicornRegex.FindStringSubmatch(line); len(gunicornMatches) > 1 {
				port, err := strconv.Atoi(gunicornMatches[1])
				if err == nil && port > 0 && port < 65536 {
					detectedPort = port
					log.Printf("[GIT] Detected port %d from Dockerfile gunicorn command", port)
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("[GIT] WARNING - Error reading Dockerfile: %v, trying alternative methods", err)
		return detectPortFromPackageJSON(repoPath)
	}

	// If we detected a port from ENV PORT, use it
	if detectedPort > 0 {
		return detectedPort
	}

	// No EXPOSE or ENV PORT found, try detecting from package.json, Python files, or source files
	log.Printf("[GIT] No EXPOSE or ENV PORT found in Dockerfile, checking source files...")
	
	// First check for Python apps (FastAPI, Flask, etc.)
	port := detectPortFromPythonFiles(repoPath)
	if port > 0 {
		return port
	}
	
	// Then check for Node.js apps
	return detectPortFromPackageJSON(repoPath)
}

// detectPortFromPackageJSON attempts to detect port from package.json scripts or source files
func detectPortFromPackageJSON(repoPath string) int {
	packageJSONPath := filepath.Join(repoPath, "package.json")
	
	// Check if package.json exists
	if _, err := os.Stat(packageJSONPath); os.IsNotExist(err) {
		log.Printf("[GIT] No package.json found, using default port 8080")
		return 8080
	}

	// Read package.json to check for Node.js app
	file, err := os.Open(packageJSONPath)
	if err != nil {
		log.Printf("[GIT] Failed to read package.json: %v, using default port 8080", err)
		return 8080
	}
	defer file.Close()

	// Check for common Node.js entry points
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.ToLower(scanner.Text())
		// Check for "main" field pointing to server.js, app.js, index.js, etc.
		if strings.Contains(line, `"main"`) || strings.Contains(line, `"start"`) {
			// This is likely a Node.js app, default to 3000 (common for Express)
			log.Printf("[GIT] Detected Node.js app from package.json, using default port 3000")
			return 3000
		}
	}

	// Check source files for port patterns (server.js, app.js, index.js)
	commonEntryPoints := []string{"server.js", "app.js", "index.js", "main.js"}
	for _, entryPoint := range commonEntryPoints {
		sourcePath := filepath.Join(repoPath, entryPoint)
		if _, err := os.Stat(sourcePath); err == nil {
			// File exists, check for port patterns
			port := detectPortFromSourceFile(sourcePath)
			if port > 0 {
				return port
			}
			// If it's a Node.js file but no port found, default to 3000
			log.Printf("[GIT] Found Node.js entry point %s, using default port 3000", entryPoint)
			return 3000
		}
	}

	log.Printf("[GIT] No port detected from package.json or source files, using default port 8080")
	return 8080
}

// detectPortFromPythonFiles attempts to detect port from Python source files (FastAPI, Flask, etc.)
func detectPortFromPythonFiles(repoPath string) int {
	// Common Python entry points
	pythonFiles := []string{"main.py", "app.py", "server.py", "application.py", "wsgi.py"}
	
	for _, pyFile := range pythonFiles {
		filePath := filepath.Join(repoPath, pyFile)
		if _, err := os.Stat(filePath); err == nil {
			// File exists, check for port patterns
			port := detectPortFromPythonFile(filePath)
			if port > 0 {
				return port
			}
			// If it's a Python file but no port found, check if it's FastAPI/Flask and use defaults
			if isFastAPIOrFlask(filePath) {
				log.Printf("[GIT] Found FastAPI/Flask app in %s, using default port 8000", pyFile)
				return 8000
			}
		}
	}
	
	return 0
}

// detectPortFromPythonFile attempts to detect port from Python source code patterns
func detectPortFromPythonFile(filePath string) int {
	file, err := os.Open(filePath)
	if err != nil {
		return 0
	}
	defer file.Close()

	// Common port patterns in Python: PORT = int(os.getenv("PORT", "8000")), port=8000, etc.
	portPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)PORT\s*=\s*int\(os\.getenv\([^,)]+,\s*["'](\d+)["']\)`), // PORT = int(os.getenv("PORT", "8000"))
		regexp.MustCompile(`(?i)PORT\s*=\s*os\.getenv\([^,)]+,\s*["'](\d+)["']`),        // PORT = os.getenv("PORT", "8000")
		regexp.MustCompile(`(?i)port\s*=\s*(\d+)`),                                        // port = 8000
		regexp.MustCompile(`(?i)uvicorn\.run\([^,)]+,\s*port\s*=\s*(\d+)`),              // uvicorn.run(..., port=8000
		regexp.MustCompile(`(?i)app\.run\([^,)]*port\s*=\s*(\d+)`),                      // app.run(port=8000
		regexp.MustCompile(`(?i)--port\s+(\d+)`),                                        // --port 8000
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		for _, pattern := range portPatterns {
			matches := pattern.FindStringSubmatch(line)
			if len(matches) > 1 {
				port, err := strconv.Atoi(matches[1])
				if err == nil && port > 0 && port < 65536 {
					log.Printf("[GIT] Detected port %d from Python file %s", port, filepath.Base(filePath))
					return port
				}
			}
		}
	}

	return 0
}

// isFastAPIOrFlask checks if a Python file contains FastAPI or Flask imports
func isFastAPIOrFlask(filePath string) bool {
	file, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.ToLower(scanner.Text())
		if strings.Contains(line, "from fastapi") || strings.Contains(line, "import fastapi") {
			return true
		}
		if strings.Contains(line, "from flask") || strings.Contains(line, "import flask") {
			return true
		}
	}

	return false
}

// detectPortFromSourceFile attempts to detect port from source code patterns (Node.js)
func detectPortFromSourceFile(filePath string) int {
	file, err := os.Open(filePath)
	if err != nil {
		return 0
	}
	defer file.Close()

	// Common port patterns in Node.js: PORT || 3000, listen(3000), port: 3000
	portPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)PORT\s*\|\|\s*(\d+)`),           // PORT || 3000
		regexp.MustCompile(`(?i)\.listen\((\d+)`),                // app.listen(3000
		regexp.MustCompile(`(?i)port\s*[:=]\s*(\d+)`),            // port: 3000 or port = 3000
		regexp.MustCompile(`(?i)process\.env\.PORT\s*\|\|\s*(\d+)`), // process.env.PORT || 3000
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		for _, pattern := range portPatterns {
			matches := pattern.FindStringSubmatch(line)
			if len(matches) > 1 {
				port, err := strconv.Atoi(matches[1])
				if err == nil && port > 0 && port < 65536 {
					log.Printf("[GIT] Detected port %d from source file %s", port, filepath.Base(filePath))
					return port
				}
			}
		}
	}

	return 0
}
