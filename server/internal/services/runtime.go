package services

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
)

// Runtime represents a detected application runtime
type Runtime string

const (
	RuntimeNodeJS   Runtime = "nodejs"
	RuntimePython   Runtime = "python"
	RuntimeGo       Runtime = "go"
	RuntimeRuby     Runtime = "ruby"
	RuntimeJava     Runtime = "java"
	RuntimePHP      Runtime = "php"
	RuntimeStatic   Runtime = "static"
	RuntimeUnknown  Runtime = "unknown"
)

// RuntimeDetector detects the runtime of an application
type RuntimeDetector struct {
	logger *zap.Logger
}

// NewRuntimeDetector creates a new runtime detector
func NewRuntimeDetector(logger *zap.Logger) *RuntimeDetector {
	return &RuntimeDetector{
		logger: logger,
	}
}

// DetectRuntime detects the runtime by examining files in the repository
func (d *RuntimeDetector) DetectRuntime(repoPath string) (Runtime, error) {
	// Check for package.json (Node.js)
	if d.fileExists(repoPath, "package.json") {
		d.logger.Info("Detected Node.js runtime", zap.String("path", repoPath))
		return RuntimeNodeJS, nil
	}

	// Check for requirements.txt or setup.py (Python)
	if d.fileExists(repoPath, "requirements.txt") || d.fileExists(repoPath, "setup.py") || d.fileExists(repoPath, "Pipfile") {
		d.logger.Info("Detected Python runtime", zap.String("path", repoPath))
		return RuntimePython, nil
	}

	// Check for go.mod (Go)
	if d.fileExists(repoPath, "go.mod") {
		d.logger.Info("Detected Go runtime", zap.String("path", repoPath))
		return RuntimeGo, nil
	}

	// Check for Gemfile (Ruby)
	if d.fileExists(repoPath, "Gemfile") {
		d.logger.Info("Detected Ruby runtime", zap.String("path", repoPath))
		return RuntimeRuby, nil
	}

	// Check for pom.xml or build.gradle (Java)
	if d.fileExists(repoPath, "pom.xml") || d.fileExists(repoPath, "build.gradle") {
		d.logger.Info("Detected Java runtime", zap.String("path", repoPath))
		return RuntimeJava, nil
	}

	// Check for composer.json (PHP)
	if d.fileExists(repoPath, "composer.json") {
		d.logger.Info("Detected PHP runtime", zap.String("path", repoPath))
		return RuntimePHP, nil
	}

	// Check for static files (HTML, CSS, JS)
	if d.hasStaticFiles(repoPath) {
		d.logger.Info("Detected static site", zap.String("path", repoPath))
		return RuntimeStatic, nil
	}

	d.logger.Warn("Could not detect runtime", zap.String("path", repoPath))
	return RuntimeUnknown, nil
}

// fileExists checks if a file exists in the repository
func (d *RuntimeDetector) fileExists(repoPath, filename string) bool {
	path := filepath.Join(repoPath, filename)
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// hasStaticFiles checks if the repository contains static files
func (d *RuntimeDetector) hasStaticFiles(repoPath string) bool {
	staticExtensions := []string{".html", ".htm", ".css", ".js"}
	
	err := filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if info.IsDir() {
			// Skip hidden directories and node_modules
			if strings.HasPrefix(info.Name(), ".") || info.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		
		ext := strings.ToLower(filepath.Ext(path))
		for _, staticExt := range staticExtensions {
			if ext == staticExt {
				return fmt.Errorf("found static file") // Signal that we found a static file
			}
		}
		return nil
	})
	
	return err != nil // If err is not nil, we found a static file
}

