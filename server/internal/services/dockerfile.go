package services

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
)

// DockerfileGenerator generates Dockerfiles using Paketo Buildpacks
type DockerfileGenerator struct {
	logger *zap.Logger
}

// NewDockerfileGenerator creates a new Dockerfile generator
func NewDockerfileGenerator(logger *zap.Logger) *DockerfileGenerator {
	return &DockerfileGenerator{
		logger: logger,
	}
}

// GenerateDockerfile generates a Dockerfile using Paketo Buildpacks for the given runtime
func (g *DockerfileGenerator) GenerateDockerfile(repoPath string, runtime Runtime) error {
	// Check if Dockerfile already exists
	dockerfilePath := filepath.Join(repoPath, "Dockerfile")
	if _, err := os.Stat(dockerfilePath); err == nil {
		g.logger.Info("Dockerfile already exists, enhancing it for Stackyn compatibility", zap.String("path", dockerfilePath))
		// Enhance the existing Dockerfile instead of skipping
		return g.enhanceExistingDockerfile(dockerfilePath, runtime)
	}

	var content string

	switch runtime {
	case RuntimeNodeJS:
		content = g.generateNodeJSDockerfile()
	case RuntimePython:
		content = g.generatePythonDockerfile()
	case RuntimeGo:
		content = g.generateGoDockerfile()
	case RuntimeJava:
		content = g.generateJavaDockerfile()
	case RuntimeRuby, RuntimePHP, RuntimeStatic:
		// These runtimes are not supported by Paketo Buildpacks
		return fmt.Errorf("runtime '%s' is not supported by Paketo Buildpacks. Supported runtimes: Node.js, Python, Go, Java", runtime)
	case RuntimeUnknown:
		return fmt.Errorf("could not detect runtime. Supported runtimes: Node.js, Python, Go, Java")
	default:
		return fmt.Errorf("unsupported runtime: %s. Supported runtimes: Node.js, Python, Go, Java", runtime)
	}

	// Write Dockerfile
	if err := os.WriteFile(dockerfilePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write Dockerfile: %w", err)
	}

	g.logger.Info("Generated Dockerfile using Paketo Buildpacks",
		zap.String("path", dockerfilePath),
		zap.String("runtime", string(runtime)),
	)

	return nil
}

// generateNodeJSDockerfile generates a Dockerfile for Node.js using Paketo Buildpacks
func (g *DockerfileGenerator) generateNodeJSDockerfile() string {
	return `# syntax=docker/dockerfile:1
# Multi-stage build using Paketo Buildpacks for Node.js

# Build stage - Use Paketo Node.js builder
FROM paketobuildpacks/builder-jammy-base:latest AS builder

# Set CNB Platform API version (required for Paketo Buildpacks)
ENV CNB_PLATFORM_API=0.12

# Set working directory
WORKDIR /workspace

# Copy application source
COPY --chown=cnb:cnb . .

# The Paketo builder image already provides /platform, /layers, and /cache directories.
# However, /platform/env and /cache may need to exist with proper permissions.
# Switch to root to create directories if needed, then back to cnb user
USER root
RUN mkdir -p /platform/env /cache && chown -R cnb:cnb /platform/env /cache || true
USER cnb

# Set up environment variables for lifecycle
ENV CNB_PLATFORM_DIR=/platform

# Run lifecycle phases individually (since creator requires image references)
# 1. Detect - Detect which buildpacks to use
# 2. Analyze - Skip (not needed for first build, would require Docker registry auth)
# 3. Restore - Restore layer metadata from cache (optional, fails gracefully if no cache)
# 4. Build - Execute buildpacks (this is the critical phase)
# Note: We skip analyzer since it requires registry authentication and we're doing a fresh build
RUN /cnb/lifecycle/detector \
    -app=/workspace \
    -platform=/platform \
    -log-level=info && \
    /cnb/lifecycle/restorer \
    -cache-dir=/cache \
    -layers=/layers \
    -log-level=info || true && \
    /cnb/lifecycle/builder \
    -app=/workspace \
    -layers=/layers \
    -platform=/platform \
    -log-level=info || \
    (echo "ERROR: Paketo Buildpacks build failed. Ensure your Node.js application has a valid package.json file." && exit 1)

# Create the /cnb/process/web script manually (exporter phase would normally create this)
# This script runs the application based on what Paketo Buildpacks detected
# Switch to root to create /cnb/process directory
USER root
RUN mkdir -p /cnb/process && \
    echo '#!/bin/sh' > /cnb/process/web && \
    echo 'set -e' >> /cnb/process/web && \
    echo '# Load environment variables from Paketo layers' >> /cnb/process/web && \
    echo 'for env_file in /layers/*/*/env.launch/*/*; do' >> /cnb/process/web && \
    echo '  [ -f "$env_file" ] || continue' >> /cnb/process/web && \
    echo '  var_name=$(basename "$(dirname "$env_file")")' >> /cnb/process/web && \
    echo '  var_value=$(cat "$env_file")' >> /cnb/process/web && \
    echo '  export "$var_name=$var_value"' >> /cnb/process/web && \
    echo 'done' >> /cnb/process/web && \
    echo '# Ensure Node.js is in PATH (backup if not set by layers)' >> /cnb/process/web && \
    echo 'if [ -d /layers/paketo-buildpacks_node-engine/node ] && ! echo "$PATH" | grep -q "paketo-buildpacks_node-engine"; then' >> /cnb/process/web && \
    echo '  export PATH="/layers/paketo-buildpacks_node-engine/node/bin:$PATH"' >> /cnb/process/web && \
    echo '  export NODE_HOME="${NODE_HOME:-/layers/paketo-buildpacks_node-engine/node}"' >> /cnb/process/web && \
    echo 'fi' >> /cnb/process/web && \
    echo '# Set NODE_PATH to include Paketo-installed node_modules (check both npm and yarn)' >> /cnb/process/web && \
    echo 'if [ -d /layers/paketo-buildpacks_npm-install/launch-modules/node_modules ]; then' >> /cnb/process/web && \
    echo '  export NODE_PATH="/layers/paketo-buildpacks_npm-install/launch-modules/node_modules${NODE_PATH:+:}$NODE_PATH"' >> /cnb/process/web && \
    echo 'fi' >> /cnb/process/web && \
    echo 'if [ -d /layers/paketo-buildpacks_yarn-install/launch-modules/node_modules ]; then' >> /cnb/process/web && \
    echo '  export NODE_PATH="/layers/paketo-buildpacks_yarn-install/launch-modules/node_modules${NODE_PATH:+:}$NODE_PATH"' >> /cnb/process/web && \
    echo 'fi' >> /cnb/process/web && \
    echo '# Try server.js first (most common), then start.sh, then other files' >> /cnb/process/web && \
    echo '# Check if package.json has "type": "module" OR if server.js uses ES module syntax' >> /cnb/process/web && \
    echo 'IS_ESM=false' >> /cnb/process/web && \
    echo 'if [ -f /workspace/package.json ]; then' >> /cnb/process/web && \
    echo '  # Check for "type": "module" in package.json' >> /cnb/process/web && \
    echo '  if node -e "try { const pkg = require(\\47/workspace/package.json\\47); process.exit(pkg.type === \\47module\\47 ? 0 : 1); } catch(e) { process.exit(1); }" 2>/dev/null; then' >> /cnb/process/web && \
    echo '    IS_ESM=true' >> /cnb/process/web && \
    echo '  fi' >> /cnb/process/web && \
    echo 'fi' >> /cnb/process/web && \
    echo '# Also check if server.js uses ES module syntax (import/export)' >> /cnb/process/web && \
    echo 'if [ "$IS_ESM" = "false" ] && [ -f /workspace/server.js ]; then' >> /cnb/process/web && \
    echo '  if grep -qE "^import |^export |from ['\''\"]" /workspace/server.js 2>/dev/null; then' >> /cnb/process/web && \
    echo '    IS_ESM=true' >> /cnb/process/web && \
    echo '  fi' >> /cnb/process/web && \
    echo 'fi' >> /cnb/process/web && \
    echo 'if [ -f /workspace/server.js ]; then' >> /cnb/process/web && \
    echo '  if [ "$IS_ESM" = "true" ]; then' >> /cnb/process/web && \
    echo '    # Use ESM loader if available (created during build)' >> /cnb/process/web && \
    echo '    if [ -f /tmp/esm-loader.mjs ]; then' >> /cnb/process/web && \
    echo '      exec node --loader /tmp/esm-loader.mjs server.js' >> /cnb/process/web && \
    echo '    else' >> /cnb/process/web && \
    echo '      exec node server.js' >> /cnb/process/web && \
    echo '    fi' >> /cnb/process/web && \
    echo '  else' >> /cnb/process/web && \
    echo '    exec node server.js' >> /cnb/process/web && \
    echo '  fi' >> /cnb/process/web && \
    echo 'elif [ -f /workspace/start.sh ]; then' >> /cnb/process/web && \
    echo '  exec sh /workspace/start.sh' >> /cnb/process/web && \
    echo 'elif [ -f /workspace/index.js ]; then' >> /cnb/process/web && \
    echo '  if [ "$IS_ESM" = "true" ] && [ -f /tmp/esm-loader.mjs ]; then' >> /cnb/process/web && \
    echo '    exec node --loader /tmp/esm-loader.mjs index.js' >> /cnb/process/web && \
    echo '  else' >> /cnb/process/web && \
    echo '    exec node index.js' >> /cnb/process/web && \
    echo '  fi' >> /cnb/process/web && \
    echo 'elif [ -f /workspace/app.js ]; then' >> /cnb/process/web && \
    echo '  if [ "$IS_ESM" = "true" ] && [ -f /tmp/esm-loader.mjs ]; then' >> /cnb/process/web && \
    echo '    exec node --loader /tmp/esm-loader.mjs app.js' >> /cnb/process/web && \
    echo '  else' >> /cnb/process/web && \
    echo '    exec node app.js' >> /cnb/process/web && \
    echo '  fi' >> /cnb/process/web && \
    echo 'elif [ -f /workspace/package.json ]; then' >> /cnb/process/web && \
    echo '  MAIN=$(node -p "try { const pkg = require(\\47./package.json\\47); pkg.main || (pkg.scripts && pkg.scripts.start ? pkg.scripts.start.split(\\47 \\47).pop() : null) || \\47index.js\\47; } catch(e) { \\47index.js\\47; }")' >> /cnb/process/web && \
    echo '  if [ -f "/workspace/$MAIN" ]; then' >> /cnb/process/web && \
    echo '    if [ "$IS_ESM" = "true" ] && [ -f /tmp/esm-loader.mjs ]; then' >> /cnb/process/web && \
    echo '      exec node --loader /tmp/esm-loader.mjs "$MAIN"' >> /cnb/process/web && \
    echo '    else' >> /cnb/process/web && \
    echo '      exec node "$MAIN"' >> /cnb/process/web && \
    echo '    fi' >> /cnb/process/web && \
    echo '  else' >> /cnb/process/web && \
    echo '    echo "ERROR: Entry point $MAIN not found"' >> /cnb/process/web && \
    echo '    exit 1' >> /cnb/process/web && \
    echo '  fi' >> /cnb/process/web && \
    echo 'else' >> /cnb/process/web && \
    echo '  echo "ERROR: Could not determine application entry point"' >> /cnb/process/web && \
    echo '  exit 1' >> /cnb/process/web && \
    echo 'fi' >> /cnb/process/web && \
    chmod +x /cnb/process/web && \
    chown -R cnb:cnb /cnb/process
USER cnb

# Production stage - Use Paketo run image
FROM paketobuildpacks/run-jammy-base:latest

# Set working directory
WORKDIR /workspace

# Copy built application from builder stage
COPY --from=builder --chown=cnb:cnb /workspace /workspace
COPY --from=builder --chown=cnb:cnb /layers /layers

# Create /cnb/process directory and script in run stage
# The run image might not have /cnb/process, so we recreate it here
USER root
RUN mkdir -p /cnb/process /tmp && \
    # Create symlink to node_modules so Node.js can find packages for ES modules
    # NODE_PATH doesn't work for ES module imports, so we need node_modules in workspace
    # Check both npm-install and yarn-install buildpacks
    if [ -d /layers/paketo-buildpacks_npm-install/launch-modules/node_modules ]; then \
        ln -sf /layers/paketo-buildpacks_npm-install/launch-modules/node_modules /workspace/node_modules; \
    elif [ -d /layers/paketo-buildpacks_yarn-install/launch-modules/node_modules ]; then \
        ln -sf /layers/paketo-buildpacks_yarn-install/launch-modules/node_modules /workspace/node_modules; \
    fi

# Copy the script from builder and ensure it's executable
COPY --from=builder /cnb/process/web /cnb/process/web

# Install socat for port forwarding (allows apps to listen on any port, we forward to 8080)
# Install wget for Docker health checks
RUN apt-get update && apt-get install -y socat wget && rm -rf /var/lib/apt/lists/*

# Create port forwarding wrapper script
# This forwards port 8080 to the app's actual port so apps work regardless of hardcoded port
RUN echo '#!/bin/sh' > /cnb/process/web-wrapper && \
    echo 'set +e' >> /cnb/process/web-wrapper && \
    echo 'echo "Stackyn: Starting app with port forwarding wrapper..."' >> /cnb/process/web-wrapper && \
    echo '# Start the app in background' >> /cnb/process/web-wrapper && \
    echo '/cnb/process/web &' >> /cnb/process/web-wrapper && \
    echo 'APP_PID=$!' >> /cnb/process/web-wrapper && \
    echo 'echo "Stackyn: App started with PID $APP_PID, waiting for it to bind to a port..."' >> /cnb/process/web-wrapper && \
    echo '# Wait for app to start and bind to a port' >> /cnb/process/web-wrapper && \
    echo 'sleep 5' >> /cnb/process/web-wrapper && \
    echo '# Detect which port the app is listening on (check common ports: 80, 3000, 5000, 8000)' >> /cnb/process/web-wrapper && \
    echo 'APP_PORT=""' >> /cnb/process/web-wrapper && \
    echo 'for port in 80 3000 5000 8000; do' >> /cnb/process/web-wrapper && \
    echo '  if ss -tln 2>/dev/null | grep -qE ":$port\\b" || netstat -tln 2>/dev/null | grep -qE ":$port\\b"; then' >> /cnb/process/web-wrapper && \
    echo '    APP_PORT=$port' >> /cnb/process/web-wrapper && \
    echo '    echo "Stackyn: Detected app listening on port $port"' >> /cnb/process/web-wrapper && \
    echo '    break' >> /cnb/process/web-wrapper && \
    echo '  fi' >> /cnb/process/web-wrapper && \
    echo 'done' >> /cnb/process/web-wrapper && \
    echo '# Default to port 80 if not detected (most common case)' >> /cnb/process/web-wrapper && \
    echo 'APP_PORT=${APP_PORT:-80}' >> /cnb/process/web-wrapper && \
    echo 'echo "Stackyn: Using port $APP_PORT for app"' >> /cnb/process/web-wrapper && \
    echo '# Forward port 8080 to the app port if different' >> /cnb/process/web-wrapper && \
    echo 'if [ "$APP_PORT" != "8080" ]; then' >> /cnb/process/web-wrapper && \
    echo '  echo "Stackyn: Forwarding port 8080 to app port $APP_PORT"' >> /cnb/process/web-wrapper && \
    echo '  socat TCP-LISTEN:8080,fork,reuseaddr TCP:localhost:$APP_PORT &' >> /cnb/process/web-wrapper && \
    echo '  SOCAT_PID=$!' >> /cnb/process/web-wrapper && \
    echo '  echo "Stackyn: Port forwarding active (socat PID $SOCAT_PID)"' >> /cnb/process/web-wrapper && \
    echo 'else' >> /cnb/process/web-wrapper && \
    echo '  echo "Stackyn: App is already on port 8080, no forwarding needed"' >> /cnb/process/web-wrapper && \
    echo 'fi' >> /cnb/process/web-wrapper && \
    echo '# Wait for app process (socat will exit if app dies)' >> /cnb/process/web-wrapper && \
    echo 'wait $APP_PID' >> /cnb/process/web-wrapper && \
    chmod +x /cnb/process/web-wrapper

# Create ESM loader file to handle directory and file imports without extensions in ES modules
RUN echo 'import { dirname } from "path";' > /tmp/esm-loader.mjs && \
    echo 'import { existsSync, statSync } from "fs";' >> /tmp/esm-loader.mjs && \
    echo 'export async function resolve(specifier, context, nextResolve) {' >> /tmp/esm-loader.mjs && \
    echo '  try {' >> /tmp/esm-loader.mjs && \
    echo '    return await nextResolve(specifier, context);' >> /tmp/esm-loader.mjs && \
    echo '  } catch (err) {' >> /tmp/esm-loader.mjs && \
    echo '    if (err.code === "ERR_UNSUPPORTED_DIR_IMPORT" || err.code === "ERR_MODULE_NOT_FOUND") {' >> /tmp/esm-loader.mjs && \
    echo '      if ((specifier.startsWith("./") || specifier.startsWith("../")) && !specifier.endsWith(".js") && !specifier.endsWith(".mjs") && !specifier.endsWith(".json")) {' >> /tmp/esm-loader.mjs && \
    echo '        try {' >> /tmp/esm-loader.mjs && \
    echo '          const { resolve, join } = await import("path");' >> /tmp/esm-loader.mjs && \
    echo '          const parentURL = context.parentURL || "file:///workspace/";' >> /tmp/esm-loader.mjs && \
    echo '          const parentDir = dirname(new URL(parentURL).pathname);' >> /tmp/esm-loader.mjs && \
    echo '          const resolvedPath = resolve(parentDir, specifier);' >> /tmp/esm-loader.mjs && \
    echo '          // First, try as directory with index.js' >> /tmp/esm-loader.mjs && \
    echo '          if (existsSync(resolvedPath) && statSync(resolvedPath).isDirectory()) {' >> /tmp/esm-loader.mjs && \
    echo '            const indexPath = join(resolvedPath, "index.js");' >> /tmp/esm-loader.mjs && \
    echo '            if (existsSync(indexPath)) {' >> /tmp/esm-loader.mjs && \
    echo '              const relativePath = "./" + specifier.replace(/\\\\/g, "/") + "/index.js";' >> /tmp/esm-loader.mjs && \
    echo '              return await nextResolve(relativePath, context);' >> /tmp/esm-loader.mjs && \
    echo '            }' >> /tmp/esm-loader.mjs && \
    echo '          }' >> /tmp/esm-loader.mjs && \
    echo '          // If not a directory, try as file with .js extension' >> /tmp/esm-loader.mjs && \
    echo '          const jsPath = resolvedPath + ".js";' >> /tmp/esm-loader.mjs && \
    echo '          if (existsSync(jsPath) && statSync(jsPath).isFile()) {' >> /tmp/esm-loader.mjs && \
    echo '            const relativePath = specifier + ".js";' >> /tmp/esm-loader.mjs && \
    echo '            return await nextResolve(relativePath, context);' >> /tmp/esm-loader.mjs && \
    echo '          }' >> /tmp/esm-loader.mjs && \
    echo '          // Try .mjs extension' >> /tmp/esm-loader.mjs && \
    echo '          const mjsPath = resolvedPath + ".mjs";' >> /tmp/esm-loader.mjs && \
    echo '          if (existsSync(mjsPath) && statSync(mjsPath).isFile()) {' >> /tmp/esm-loader.mjs && \
    echo '            const relativePath = specifier + ".mjs";' >> /tmp/esm-loader.mjs && \
    echo '            return await nextResolve(relativePath, context);' >> /tmp/esm-loader.mjs && \
    echo '          }' >> /tmp/esm-loader.mjs && \
    echo '        } catch (e) {}' >> /tmp/esm-loader.mjs && \
    echo '      }' >> /tmp/esm-loader.mjs && \
    echo '    }' >> /tmp/esm-loader.mjs && \
    echo '    throw err;' >> /tmp/esm-loader.mjs && \
    echo '  }' >> /tmp/esm-loader.mjs && \
    echo '}' >> /tmp/esm-loader.mjs && \
    chmod +x /cnb/process/web && chmod 644 /tmp/esm-loader.mjs && \
    chown -R cnb:cnb /cnb/process
USER cnb

# Expose dynamic PORT (Paketo Buildpacks set PORT env var at runtime)
# Default to 3000 if PORT is not set
EXPOSE ${PORT:-3000}

# Use the wrapper script that handles port forwarding
# Use shell form to ensure proper execution
CMD ["/bin/sh", "/cnb/process/web-wrapper"]
`
}

// generatePythonDockerfile generates a Dockerfile for Python using Paketo Buildpacks
func (g *DockerfileGenerator) generatePythonDockerfile() string {
	return `# syntax=docker/dockerfile:1
# Multi-stage build using Paketo Buildpacks for Python

# Build stage - Use Paketo Python builder
FROM paketobuildpacks/builder-jammy-base:latest AS builder

# Set CNB Platform API version (required for Paketo Buildpacks)
ENV CNB_PLATFORM_API=0.12

# Set working directory
WORKDIR /workspace

# Copy application source
COPY --chown=cnb:cnb . .

# Set up environment variables for lifecycle
ENV CNB_PLATFORM_DIR=/platform

# Switch to root to create directories if needed, then back to cnb user
USER root
RUN mkdir -p /platform/env /cache && chown -R cnb:cnb /platform/env /cache || true
USER cnb

# Run lifecycle phases individually (since creator requires process type)
# 1. Detect - Detect which buildpacks to use
# 2. Restore - Restore layer metadata from cache (optional, fails gracefully if no cache)
# 3. Build - Execute buildpacks (this is the critical phase)
RUN /cnb/lifecycle/detector \
    -app=/workspace \
    -platform=/platform \
    -log-level=info && \
    /cnb/lifecycle/restorer \
    -cache-dir=/cache \
    -layers=/layers \
    -log-level=info || true && \
    /cnb/lifecycle/builder \
    -app=/workspace \
    -layers=/layers \
    -platform=/platform \
    -log-level=info || \
    (echo "ERROR: Paketo Buildpacks build failed. Ensure your Python application has a valid requirements.txt, setup.py, Pipfile, or pyproject.toml." && exit 1)

# Create the /cnb/process/web script manually (exporter phase would normally create this)
# This script runs the Python application based on what Paketo Buildpacks detected
USER root
RUN mkdir -p /cnb/process && \
    echo '#!/bin/sh' > /cnb/process/web && \
    echo 'set -e' >> /cnb/process/web && \
    echo '# Load environment variables from Paketo layers' >> /cnb/process/web && \
    echo 'for env_file in /layers/*/*/env.launch/*/*; do' >> /cnb/process/web && \
    echo '  [ -f "$env_file" ] || continue' >> /cnb/process/web && \
    echo '  var_name=$(basename "$(dirname "$env_file")")' >> /cnb/process/web && \
    echo '  var_value=$(cat "$env_file")' >> /cnb/process/web && \
    echo '  export "$var_name=$var_value"' >> /cnb/process/web && \
    echo 'done' >> /cnb/process/web && \
    echo '# Find Python executable' >> /cnb/process/web && \
    echo 'PYTHON=""' >> /cnb/process/web && \
    echo '# Check Paketo Python layer first (multiple possible locations)' >> /cnb/process/web && \
    echo 'if [ -f /layers/paketo-buildpacks_cpython/python/bin/python3 ]; then' >> /cnb/process/web && \
    echo '  PYTHON="/layers/paketo-buildpacks_cpython/python/bin/python3"' >> /cnb/process/web && \
    echo 'elif [ -f /layers/paketo-buildpacks_cpython/python/bin/python ]; then' >> /cnb/process/web && \
    echo '  PYTHON="/layers/paketo-buildpacks_cpython/python/bin/python"' >> /cnb/process/web && \
    echo 'elif [ -d /layers/paketo-buildpacks_cpython ]; then' >> /cnb/process/web && \
    echo '  # Search for python in any subdirectory' >> /cnb/process/web && \
    echo '  PYTHON=$(find /layers/paketo-buildpacks_cpython -name python3 -type f 2>/dev/null | head -1)' >> /cnb/process/web && \
    echo '  [ -z "$PYTHON" ] && PYTHON=$(find /layers/paketo-buildpacks_cpython -name python -type f 2>/dev/null | head -1)' >> /cnb/process/web && \
    echo '  [ -n "$PYTHON" ] && [ -x "$PYTHON" ] || PYTHON=""' >> /cnb/process/web && \
    echo 'fi' >> /cnb/process/web && \
    echo '# Check system Python if Paketo Python not found' >> /cnb/process/web && \
    echo 'if [ -z "$PYTHON" ]; then' >> /cnb/process/web && \
    echo '  if command -v python3 >/dev/null 2>&1; then' >> /cnb/process/web && \
    echo '    PYTHON="python3"' >> /cnb/process/web && \
    echo '  elif command -v python >/dev/null 2>&1; then' >> /cnb/process/web && \
    echo '    PYTHON="python"' >> /cnb/process/web && \
    echo '  fi' >> /cnb/process/web && \
    echo 'fi' >> /cnb/process/web && \
    echo '# Final check - if still no Python, show error with debug info' >> /cnb/process/web && \
    echo 'if [ -z "$PYTHON" ]; then' >> /cnb/process/web && \
    echo '  echo "ERROR: Python executable not found"' >> /cnb/process/web && \
    echo '  echo "Debug: Checking Python locations..."' >> /cnb/process/web && \
    echo '  [ -d /layers/paketo-buildpacks_cpython ] && echo "  - /layers/paketo-buildpacks_cpython exists" || echo "  - /layers/paketo-buildpacks_cpython not found"' >> /cnb/process/web && \
    echo '  [ -d /layers ] && find /layers -name "*python*" -type d 2>/dev/null | head -5 || echo "  - No Python layers found"' >> /cnb/process/web && \
    echo '  exit 1' >> /cnb/process/web && \
    echo 'fi' >> /cnb/process/web && \
    echo '# Add Python bin to PATH' >> /cnb/process/web && \
    echo 'if [ -n "$PYTHON" ] && [ "$PYTHON" != "python3" ] && [ "$PYTHON" != "python" ]; then' >> /cnb/process/web && \
    echo '  # Extract directory from Python path and add to PATH' >> /cnb/process/web && \
    echo '  PYTHON_DIR=$(dirname "$PYTHON")' >> /cnb/process/web && \
    echo '  export PATH="$PYTHON_DIR:$PATH"' >> /cnb/process/web && \
    echo 'elif [ -d /layers/paketo-buildpacks_cpython/python/bin ]; then' >> /cnb/process/web && \
    echo '  export PATH="/layers/paketo-buildpacks_cpython/python/bin:$PATH"' >> /cnb/process/web && \
    echo 'elif [ -d /layers/paketo-buildpacks_cpython ]; then' >> /cnb/process/web && \
    echo '  # Find any bin directory in cpython layer' >> /cnb/process/web && \
    echo '  PYTHON_BIN=$(find /layers/paketo-buildpacks_cpython -type d -name bin 2>/dev/null | head -1)' >> /cnb/process/web && \
    echo '  [ -n "$PYTHON_BIN" ] && export PATH="$PYTHON_BIN:$PATH"' >> /cnb/process/web && \
    echo 'fi' >> /cnb/process/web && \
    echo '# Add pip-installed packages to PATH' >> /cnb/process/web && \
    echo 'if [ -d /layers/paketo-buildpacks_pip-install/packages ]; then' >> /cnb/process/web && \
    echo '  export PYTHONPATH="/layers/paketo-buildpacks_pip-install/packages/lib/python*/site-packages:$PYTHONPATH"' >> /cnb/process/web && \
    echo 'fi' >> /cnb/process/web && \
    echo '# Try to detect and run the application' >> /cnb/process/web && \
    echo '# Check for common FastAPI/uvicorn patterns' >> /cnb/process/web && \
    echo 'if [ -f /workspace/main.py ] && grep -q "FastAPI\|from fastapi" /workspace/main.py 2>/dev/null; then' >> /cnb/process/web && \
    echo '  exec "$PYTHON" -m uvicorn main:app --host 0.0.0.0 --port "${PORT:-8000}"' >> /cnb/process/web && \
    echo '# Check for app.py with FastAPI' >> /cnb/process/web && \
    echo 'elif [ -f /workspace/app.py ] && grep -q "FastAPI\|from fastapi" /workspace/app.py 2>/dev/null; then' >> /cnb/process/web && \
    echo '  exec "$PYTHON" -m uvicorn app:app --host 0.0.0.0 --port "${PORT:-8000}"' >> /cnb/process/web && \
    echo '# Check for Flask' >> /cnb/process/web && \
    echo 'elif [ -f /workspace/app.py ] && grep -q "Flask\|from flask" /workspace/app.py 2>/dev/null; then' >> /cnb/process/web && \
    echo '  exec "$PYTHON" -m flask run --host 0.0.0.0 --port "${PORT:-8000}"' >> /cnb/process/web && \
    echo '# Check for Django' >> /cnb/process/web && \
    echo 'elif [ -f /workspace/manage.py ]; then' >> /cnb/process/web && \
    echo '  exec "$PYTHON" manage.py runserver 0.0.0.0:"${PORT:-8000}"' >> /cnb/process/web && \
    echo '# Check for gunicorn with wsgi' >> /cnb/process/web && \
    echo 'elif [ -f /workspace/wsgi.py ]; then' >> /cnb/process/web && \
    echo '  exec "$PYTHON" -m gunicorn wsgi:app --bind 0.0.0.0:"${PORT:-8000}"' >> /cnb/process/web && \
    echo '# Check for Procfile' >> /cnb/process/web && \
    echo 'elif [ -f /workspace/Procfile ]; then' >> /cnb/process/web && \
    echo '  WEB_CMD=$(grep "^web:" /workspace/Procfile | cut -d: -f2- | sed "s/^[[:space:]]*//")' >> /cnb/process/web && \
    echo '  if [ -n "$WEB_CMD" ]; then' >> /cnb/process/web && \
    echo '    exec sh -c "$WEB_CMD"' >> /cnb/process/web && \
    echo '  fi' >> /cnb/process/web && \
    echo '# Check for main.py or app.py as fallback' >> /cnb/process/web && \
    echo 'elif [ -f /workspace/main.py ]; then' >> /cnb/process/web && \
    echo '  exec "$PYTHON" main.py' >> /cnb/process/web && \
    echo 'elif [ -f /workspace/app.py ]; then' >> /cnb/process/web && \
    echo '  exec "$PYTHON" app.py' >> /cnb/process/web && \
    echo '# Check for pyproject.toml with [project.scripts] or [tool.uvicorn]' >> /cnb/process/web && \
    echo 'elif [ -f /workspace/pyproject.toml ]; then' >> /cnb/process/web && \
    echo '  # Try to extract uvicorn command from pyproject.toml' >> /cnb/process/web && \
    echo '  if grep -q "uvicorn" /workspace/pyproject.toml 2>/dev/null; then' >> /cnb/process/web && \
    echo '    MODULE=$(grep -A 5 "\\[tool.uvicorn\\]" /workspace/pyproject.toml 2>/dev/null | grep "app" | cut -d= -f2 | tr -d " \\"\\047" || echo "main:app")' >> /cnb/process/web && \
    echo '    exec "$PYTHON" -m uvicorn "$MODULE" --host 0.0.0.0 --port "${PORT:-8000}"' >> /cnb/process/web && \
    echo '  else' >> /cnb/process/web && \
    echo '    exec "$PYTHON" -m pip list >/dev/null 2>&1 && echo "Python environment ready. Please specify how to run your application." && exit 1' >> /cnb/process/web && \
    echo '  fi' >> /cnb/process/web && \
    echo 'else' >> /cnb/process/web && \
    echo '  echo "ERROR: Could not determine how to run Python application"' >> /cnb/process/web && \
    echo '  echo "Please ensure your application has one of:"' >> /cnb/process/web && \
    echo '  echo "  - main.py or app.py with FastAPI/Flask/Django"' >> /cnb/process/web && \
    echo '  echo "  - wsgi.py for WSGI applications"' >> /cnb/process/web && \
    echo '  echo "  - Procfile with web command"' >> /cnb/process/web && \
    echo '  echo "  - pyproject.toml with uvicorn configuration"' >> /cnb/process/web && \
    echo '  exit 1' >> /cnb/process/web && \
    echo 'fi' >> /cnb/process/web && \
    chmod +x /cnb/process/web && \
    chown -R cnb:cnb /cnb/process
USER cnb

# Production stage - Use Paketo run image
FROM paketobuildpacks/run-jammy-base:latest

# Set working directory
WORKDIR /workspace

# Copy built application from builder stage
COPY --from=builder --chown=cnb:cnb /workspace /workspace
COPY --from=builder --chown=cnb:cnb /layers /layers

# Copy the script from builder and ensure it's executable
COPY --from=builder /cnb/process/web /cnb/process/web

# Install socat for port forwarding (allows apps to listen on any port, we forward to 8080)
# Install wget for Docker health checks
USER root
RUN apt-get update && apt-get install -y socat wget && rm -rf /var/lib/apt/lists/*

# Create port forwarding wrapper script
# This forwards port 8080 to the app's actual port so apps work regardless of hardcoded port
RUN echo '#!/bin/sh' > /cnb/process/web-wrapper && \
    echo 'set +e' >> /cnb/process/web-wrapper && \
    echo 'echo "Stackyn: Starting app with port forwarding wrapper..."' >> /cnb/process/web-wrapper && \
    echo '# Start the app in background' >> /cnb/process/web-wrapper && \
    echo '/cnb/process/web &' >> /cnb/process/web-wrapper && \
    echo 'APP_PID=$!' >> /cnb/process/web-wrapper && \
    echo 'echo "Stackyn: App started with PID $APP_PID, waiting for it to bind to a port..."' >> /cnb/process/web-wrapper && \
    echo '# Wait for app to start and bind to a port' >> /cnb/process/web-wrapper && \
    echo 'sleep 5' >> /cnb/process/web-wrapper && \
    echo '# Detect which port the app is listening on (check common ports: 80, 3000, 5000, 8000)' >> /cnb/process/web-wrapper && \
    echo 'APP_PORT=""' >> /cnb/process/web-wrapper && \
    echo 'for port in 80 3000 5000 8000; do' >> /cnb/process/web-wrapper && \
    echo '  if ss -tln 2>/dev/null | grep -qE ":$port\\b" || netstat -tln 2>/dev/null | grep -qE ":$port\\b"; then' >> /cnb/process/web-wrapper && \
    echo '    APP_PORT=$port' >> /cnb/process/web-wrapper && \
    echo '    echo "Stackyn: Detected app listening on port $port"' >> /cnb/process/web-wrapper && \
    echo '    break' >> /cnb/process/web-wrapper && \
    echo '  fi' >> /cnb/process/web-wrapper && \
    echo 'done' >> /cnb/process/web-wrapper && \
    echo '# Default to port 80 if not detected (most common case)' >> /cnb/process/web-wrapper && \
    echo 'APP_PORT=${APP_PORT:-80}' >> /cnb/process/web-wrapper && \
    echo 'echo "Stackyn: Using port $APP_PORT for app"' >> /cnb/process/web-wrapper && \
    echo '# Forward port 8080 to the app port if different' >> /cnb/process/web-wrapper && \
    echo 'if [ "$APP_PORT" != "8080" ]; then' >> /cnb/process/web-wrapper && \
    echo '  echo "Stackyn: Forwarding port 8080 to app port $APP_PORT"' >> /cnb/process/web-wrapper && \
    echo '  socat TCP-LISTEN:8080,fork,reuseaddr TCP:localhost:$APP_PORT &' >> /cnb/process/web-wrapper && \
    echo '  SOCAT_PID=$!' >> /cnb/process/web-wrapper && \
    echo '  echo "Stackyn: Port forwarding active (socat PID $SOCAT_PID)"' >> /cnb/process/web-wrapper && \
    echo 'else' >> /cnb/process/web-wrapper && \
    echo '  echo "Stackyn: App is already on port 8080, no forwarding needed"' >> /cnb/process/web-wrapper && \
    echo 'fi' >> /cnb/process/web-wrapper && \
    echo '# Wait for app process (socat will exit if app dies)' >> /cnb/process/web-wrapper && \
    echo 'wait $APP_PID' >> /cnb/process/web-wrapper && \
    chmod +x /cnb/process/web-wrapper && \
    chmod +x /cnb/process/web && \
    chown -R cnb:cnb /cnb/process
USER cnb

# Expose dynamic PORT (Paketo Buildpacks set PORT env var at runtime)
# Default to 8000 if PORT is not set
EXPOSE ${PORT:-8000}

# Use the wrapper script that handles port forwarding
# Use shell form to ensure proper execution
CMD ["/bin/sh", "/cnb/process/web-wrapper"]
`
}

// generateGoDockerfile generates a Dockerfile for Go using Paketo Buildpacks
func (g *DockerfileGenerator) generateGoDockerfile() string {
	return `# syntax=docker/dockerfile:1
# Multi-stage build using Paketo Buildpacks for Go

# Build stage - Use Paketo Go builder
FROM paketobuildpacks/builder-jammy-base:latest AS builder

# Set CNB Platform API version (required for Paketo Buildpacks)
ENV CNB_PLATFORM_API=0.12

# Set working directory
WORKDIR /workspace

# Copy application source
COPY --chown=cnb:cnb . .

# Set up environment variables for lifecycle
ENV CNB_PLATFORM_DIR=/platform

# Switch to root to create directories if needed, then back to cnb user
USER root
RUN mkdir -p /platform/env /cache && chown -R cnb:cnb /platform/env /cache || true
USER cnb

# Run lifecycle phases individually (since creator requires process type)
# 1. Detect - Detect which buildpacks to use
# 2. Restore - Restore layer metadata from cache (optional, fails gracefully if no cache)
# 3. Build - Execute buildpacks (this is the critical phase)
RUN /cnb/lifecycle/detector \
    -app=/workspace \
    -platform=/platform \
    -log-level=info && \
    /cnb/lifecycle/restorer \
    -cache-dir=/cache \
    -layers=/layers \
    -log-level=info || true && \
    /cnb/lifecycle/builder \
    -app=/workspace \
    -layers=/layers \
    -platform=/platform \
    -log-level=info || \
    (echo "ERROR: Paketo Buildpacks build failed. Ensure your Go application has a valid go.mod file." && exit 1)

# Create the /cnb/process/web script manually (exporter phase would normally create this)
# This script runs the Go binary based on what Paketo Buildpacks detected
USER root
RUN mkdir -p /cnb/process && \
    echo '#!/bin/sh' > /cnb/process/web && \
    echo 'set -e' >> /cnb/process/web && \
    echo '# Load environment variables from Paketo layers' >> /cnb/process/web && \
    echo 'for env_file in /layers/*/*/env.launch/*/*; do' >> /cnb/process/web && \
    echo '  [ -f "$env_file" ] || continue' >> /cnb/process/web && \
    echo '  var_name=$(basename "$(dirname "$env_file")")' >> /cnb/process/web && \
    echo '  var_value=$(cat "$env_file")' >> /cnb/process/web && \
    echo '  export "$var_name=$var_value"' >> /cnb/process/web && \
    echo 'done' >> /cnb/process/web && \
    echo '# Find and run the Go binary' >> /cnb/process/web && \
    echo '# Paketo Go buildpack creates the binary based on go.mod module name or main package location' >> /cnb/process/web && \
    echo '# Check common locations and patterns' >> /cnb/process/web && \
    echo 'BINARY=""' >> /cnb/process/web && \
    echo '# First, check if there'\''s a binary in /layers/paketo-buildpacks_go-build/targets/bin/' >> /cnb/process/web && \
    echo 'if [ -d /layers/paketo-buildpacks_go-build/targets/bin ]; then' >> /cnb/process/web && \
    echo '  BINARY=$(find /layers/paketo-buildpacks_go-build/targets/bin -type f -executable 2>/dev/null | head -1)' >> /cnb/process/web && \
    echo 'fi' >> /cnb/process/web && \
    echo '# If not found, check workspace root for common binary names' >> /cnb/process/web && \
    echo 'if [ -z "$BINARY" ]; then' >> /cnb/process/web && \
    echo '  for name in server main app; do' >> /cnb/process/web && \
    echo '    if [ -f "/workspace/$name" ] && [ -x "/workspace/$name" ]; then' >> /cnb/process/web && \
    echo '      BINARY="/workspace/$name"' >> /cnb/process/web && \
    echo '      break' >> /cnb/process/web && \
    echo '    fi' >> /cnb/process/web && \
    echo '  done' >> /cnb/process/web && \
    echo 'fi' >> /cnb/process/web && \
    echo '# If still not found, find any executable in workspace root' >> /cnb/process/web && \
    echo 'if [ -z "$BINARY" ]; then' >> /cnb/process/web && \
    echo '  BINARY=$(find /workspace -maxdepth 1 -type f -executable 2>/dev/null | head -1)' >> /cnb/process/web && \
    echo 'fi' >> /cnb/process/web && \
    echo '# If binary found, execute it' >> /cnb/process/web && \
    echo 'if [ -n "$BINARY" ] && [ -f "$BINARY" ]; then' >> /cnb/process/web && \
    echo '  exec "$BINARY"' >> /cnb/process/web && \
    echo 'else' >> /cnb/process/web && \
    echo '  echo "ERROR: Could not find Go binary. Expected executable in /workspace or /layers/paketo-buildpacks_go-build/targets/bin/"' >> /cnb/process/web && \
    echo '  echo "Available files in /workspace:"' >> /cnb/process/web && \
    echo '  ls -la /workspace 2>/dev/null || true' >> /cnb/process/web && \
    echo '  exit 1' >> /cnb/process/web && \
    echo 'fi' >> /cnb/process/web && \
    chmod +x /cnb/process/web && \
    chown -R cnb:cnb /cnb/process
USER cnb

# Production stage - Use Paketo run image
FROM paketobuildpacks/run-jammy-base:latest

# Set working directory
WORKDIR /workspace

# Copy built application from builder stage
COPY --from=builder --chown=cnb:cnb /workspace /workspace
COPY --from=builder --chown=cnb:cnb /layers /layers

# Copy the script from builder and ensure it's executable
COPY --from=builder /cnb/process/web /cnb/process/web

# Install socat for port forwarding (allows apps to listen on any port, we forward to 8080)
# Install wget for Docker health checks
USER root
RUN apt-get update && apt-get install -y socat wget && rm -rf /var/lib/apt/lists/*

# Create port forwarding wrapper script
# This forwards port 8080 to the app's actual port so apps work regardless of hardcoded port
RUN echo '#!/bin/sh' > /cnb/process/web-wrapper && \
    echo 'set +e' >> /cnb/process/web-wrapper && \
    echo 'echo "Stackyn: Starting app with port forwarding wrapper..."' >> /cnb/process/web-wrapper && \
    echo '# Start the app in background' >> /cnb/process/web-wrapper && \
    echo '/cnb/process/web &' >> /cnb/process/web-wrapper && \
    echo 'APP_PID=$!' >> /cnb/process/web-wrapper && \
    echo 'echo "Stackyn: App started with PID $APP_PID, waiting for it to bind to a port..."' >> /cnb/process/web-wrapper && \
    echo '# Wait for app to start and bind to a port' >> /cnb/process/web-wrapper && \
    echo 'sleep 5' >> /cnb/process/web-wrapper && \
    echo '# Detect which port the app is listening on (check common ports: 80, 3000, 5000, 8000)' >> /cnb/process/web-wrapper && \
    echo 'APP_PORT=""' >> /cnb/process/web-wrapper && \
    echo 'for port in 80 3000 5000 8000; do' >> /cnb/process/web-wrapper && \
    echo '  if ss -tln 2>/dev/null | grep -qE ":$port\\b" || netstat -tln 2>/dev/null | grep -qE ":$port\\b"; then' >> /cnb/process/web-wrapper && \
    echo '    APP_PORT=$port' >> /cnb/process/web-wrapper && \
    echo '    echo "Stackyn: Detected app listening on port $port"' >> /cnb/process/web-wrapper && \
    echo '    break' >> /cnb/process/web-wrapper && \
    echo '  fi' >> /cnb/process/web-wrapper && \
    echo 'done' >> /cnb/process/web-wrapper && \
    echo '# Default to port 80 if not detected (most common case)' >> /cnb/process/web-wrapper && \
    echo 'APP_PORT=${APP_PORT:-80}' >> /cnb/process/web-wrapper && \
    echo 'echo "Stackyn: Using port $APP_PORT for app"' >> /cnb/process/web-wrapper && \
    echo '# Forward port 8080 to the app port if different' >> /cnb/process/web-wrapper && \
    echo 'if [ "$APP_PORT" != "8080" ]; then' >> /cnb/process/web-wrapper && \
    echo '  echo "Stackyn: Forwarding port 8080 to app port $APP_PORT"' >> /cnb/process/web-wrapper && \
    echo '  socat TCP-LISTEN:8080,fork,reuseaddr TCP:localhost:$APP_PORT &' >> /cnb/process/web-wrapper && \
    echo '  SOCAT_PID=$!' >> /cnb/process/web-wrapper && \
    echo '  echo "Stackyn: Port forwarding active (socat PID $SOCAT_PID)"' >> /cnb/process/web-wrapper && \
    echo 'else' >> /cnb/process/web-wrapper && \
    echo '  echo "Stackyn: App is already on port 8080, no forwarding needed"' >> /cnb/process/web-wrapper && \
    echo 'fi' >> /cnb/process/web-wrapper && \
    echo '# Wait for app process (socat will exit if app dies)' >> /cnb/process/web-wrapper && \
    echo 'wait $APP_PID' >> /cnb/process/web-wrapper && \
    chmod +x /cnb/process/web-wrapper && \
    chmod +x /cnb/process/web && \
    chown -R cnb:cnb /cnb/process
USER cnb

# Expose dynamic PORT (Paketo Buildpacks set PORT env var at runtime)
# Default to 8080 if PORT is not set
EXPOSE ${PORT:-8080}

# Use the wrapper script that handles port forwarding
# Use shell form to ensure proper execution
CMD ["/bin/sh", "/cnb/process/web-wrapper"]
`
}

// generateJavaDockerfile generates a Dockerfile for Java using Paketo Buildpacks
func (g *DockerfileGenerator) generateJavaDockerfile() string {
	return `# syntax=docker/dockerfile:1
# Multi-stage build using Paketo Buildpacks for Java

# Build stage - Use Paketo Java builder
FROM paketobuildpacks/builder-jammy-base:latest AS builder

# Set CNB Platform API version (required for Paketo Buildpacks)
ENV CNB_PLATFORM_API=0.12

# Set working directory
WORKDIR /workspace

# Copy application source
COPY --chown=cnb:cnb . .

# Build using Paketo Buildpacks lifecycle
# The builder will automatically detect Java (Maven/Gradle) and build
# Note: This requires the CNB lifecycle tools to be available in the builder
RUN /cnb/lifecycle/creator \
    -app=/workspace \
    -layers=/layers \
    -platform=/platform \
    -cache-dir=/cache \
    -log-level=info \
    || (echo "ERROR: Paketo Buildpacks build failed. Ensure your Java application has a valid pom.xml or build.gradle file." && exit 1)

# Production stage - Use Paketo run image
FROM paketobuildpacks/run-jammy-base:latest

# Set working directory
WORKDIR /workspace

# Copy built application from builder stage
COPY --from=builder --chown=cnb:cnb /workspace /workspace
COPY --from=builder --chown=cnb:cnb /layers /layers
COPY --from=builder --chown=cnb:cnb /platform /platform

# Install socat for port forwarding (allows apps to listen on any port, we forward to 8080)
# Note: /cnb/process/web is created by Paketo Buildpacks lifecycle, so we don't need to copy it
USER root
RUN apt-get update && apt-get install -y socat && rm -rf /var/lib/apt/lists/*

# Create port forwarding wrapper script
# This forwards port 8080 to the app's actual port so apps work regardless of hardcoded port
RUN echo '#!/bin/sh' > /cnb/process/web-wrapper && \
    echo 'set +e' >> /cnb/process/web-wrapper && \
    echo 'echo "Stackyn: Starting app with port forwarding wrapper..."' >> /cnb/process/web-wrapper && \
    echo '# Start the app in background' >> /cnb/process/web-wrapper && \
    echo '/cnb/process/web &' >> /cnb/process/web-wrapper && \
    echo 'APP_PID=$!' >> /cnb/process/web-wrapper && \
    echo 'echo "Stackyn: App started with PID $APP_PID, waiting for it to bind to a port..."' >> /cnb/process/web-wrapper && \
    echo '# Wait for app to start and bind to a port' >> /cnb/process/web-wrapper && \
    echo 'sleep 5' >> /cnb/process/web-wrapper && \
    echo '# Detect which port the app is listening on (check common ports: 80, 3000, 5000, 8000)' >> /cnb/process/web-wrapper && \
    echo 'APP_PORT=""' >> /cnb/process/web-wrapper && \
    echo 'for port in 80 3000 5000 8000; do' >> /cnb/process/web-wrapper && \
    echo '  if ss -tln 2>/dev/null | grep -qE ":$port\\b" || netstat -tln 2>/dev/null | grep -qE ":$port\\b"; then' >> /cnb/process/web-wrapper && \
    echo '    APP_PORT=$port' >> /cnb/process/web-wrapper && \
    echo '    echo "Stackyn: Detected app listening on port $port"' >> /cnb/process/web-wrapper && \
    echo '    break' >> /cnb/process/web-wrapper && \
    echo '  fi' >> /cnb/process/web-wrapper && \
    echo 'done' >> /cnb/process/web-wrapper && \
    echo '# Default to port 80 if not detected (most common case)' >> /cnb/process/web-wrapper && \
    echo 'APP_PORT=${APP_PORT:-80}' >> /cnb/process/web-wrapper && \
    echo 'echo "Stackyn: Using port $APP_PORT for app"' >> /cnb/process/web-wrapper && \
    echo '# Forward port 8080 to the app port if different' >> /cnb/process/web-wrapper && \
    echo 'if [ "$APP_PORT" != "8080" ]; then' >> /cnb/process/web-wrapper && \
    echo '  echo "Stackyn: Forwarding port 8080 to app port $APP_PORT"' >> /cnb/process/web-wrapper && \
    echo '  socat TCP-LISTEN:8080,fork,reuseaddr TCP:localhost:$APP_PORT &' >> /cnb/process/web-wrapper && \
    echo '  SOCAT_PID=$!' >> /cnb/process/web-wrapper && \
    echo '  echo "Stackyn: Port forwarding active (socat PID $SOCAT_PID)"' >> /cnb/process/web-wrapper && \
    echo 'else' >> /cnb/process/web-wrapper && \
    echo '  echo "Stackyn: App is already on port 8080, no forwarding needed"' >> /cnb/process/web-wrapper && \
    echo 'fi' >> /cnb/process/web-wrapper && \
    echo '# Wait for app process (socat will exit if app dies)' >> /cnb/process/web-wrapper && \
    echo 'wait $APP_PID' >> /cnb/process/web-wrapper && \
    chmod +x /cnb/process/web-wrapper && \
    mkdir -p /cnb/process && \
    chown -R cnb:cnb /cnb/process
USER cnb

# Expose dynamic PORT (Paketo Buildpacks set PORT env var at runtime)
# Default to 8080 if PORT is not set
EXPOSE ${PORT:-8080}

# Use the wrapper script that handles port forwarding
# The PORT environment variable will be set by the platform
CMD ["/bin/sh", "/cnb/process/web-wrapper"]
`
}

// enhanceExistingDockerfile enhances a user-provided Dockerfile for Stackyn compatibility
func (g *DockerfileGenerator) enhanceExistingDockerfile(dockerfilePath string, runtime Runtime) error {
	// Read the existing Dockerfile
	content, err := os.ReadFile(dockerfilePath)
	if err != nil {
		return fmt.Errorf("failed to read Dockerfile: %w", err)
	}

	originalContent := string(content)
	enhanced := originalContent
	modified := false

	// 1. Add wget and socat to apt-get install commands (for health checks and port forwarding)
	needsWget := strings.Contains(enhanced, "apt-get install") && !strings.Contains(enhanced, "wget")
	needsSocat := strings.Contains(enhanced, "apt-get install") && !strings.Contains(enhanced, "socat")
	if needsWget || needsSocat {
		enhanced = g.addWgetToAptInstall(enhanced)
		modified = true
		if needsWget {
			g.logger.Info("Added wget to Dockerfile for health checks")
		}
		if needsSocat {
			g.logger.Info("Added socat to Dockerfile for port forwarding")
		}
	}

	// 2. Fix Poetry version issues (upgrade old versions)
	if runtime == RuntimePython && strings.Contains(enhanced, "poetry") {
		enhanced = g.fixPoetryVersion(enhanced)
		modified = true
		g.logger.Info("Fixed Poetry version in Dockerfile")
	}

	// 3. Add PORT environment variable if not present
	if !strings.Contains(enhanced, "ENV PORT") && !strings.Contains(enhanced, "export PORT") {
		enhanced = g.addPortEnvVar(enhanced)
		modified = true
		g.logger.Info("Added PORT environment variable to Dockerfile")
	}

	// 4. Add port forwarding wrapper for Python apps using Poetry
	if runtime == RuntimePython && strings.Contains(enhanced, "poetry") && strings.Contains(enhanced, "uvicorn") {
		if !strings.Contains(enhanced, "socat") && !strings.Contains(enhanced, "web-wrapper") {
			enhanced = g.addPortForwardingWrapper(enhanced)
			modified = true
			g.logger.Info("Added port forwarding wrapper for Poetry-based Python app")
		}
	}

	// 5. Add socat for port forwarding if using hardcoded ports
	if (strings.Contains(enhanced, "EXPOSE 8000") || strings.Contains(enhanced, "EXPOSE 3000") || 
		strings.Contains(enhanced, "EXPOSE 5000") || strings.Contains(enhanced, "EXPOSE 80")) &&
		!strings.Contains(enhanced, "socat") {
		enhanced = g.addSocatForPortForwarding(enhanced)
		modified = true
		g.logger.Info("Added socat for port forwarding")
	}

	if modified {
		// Write the enhanced Dockerfile
		if err := os.WriteFile(dockerfilePath, []byte(enhanced), 0644); err != nil {
			return fmt.Errorf("failed to write enhanced Dockerfile: %w", err)
		}
		g.logger.Info("Enhanced Dockerfile for Stackyn compatibility", zap.String("path", dockerfilePath))
	} else {
		g.logger.Info("Dockerfile already compatible with Stackyn, no changes needed")
	}

	return nil
}

// addWgetToAptInstall adds wget and socat to apt-get install commands
func (g *DockerfileGenerator) addWgetToAptInstall(content string) string {
	hasWget := strings.Contains(content, "wget")
	hasSocat := strings.Contains(content, "socat")
	
	// If both already present, return as-is
	if hasWget && hasSocat {
		return content
	}
	
	lines := strings.Split(content, "\n")
	result := make([]string, 0, len(lines))
	modified := false
	
	// Find the apt-get install line and add packages to it
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// Look for apt-get install line (could be in RUN or continuation)
		if strings.Contains(trimmed, "apt-get install") {
			// Check if packages need to be added
			if (!hasWget || !hasSocat) && 
				!strings.Contains(line, "socat") && 
				!strings.Contains(line, "wget") {
				
				// Add packages after --no-install-recommends or after -y flag
				packagesToAdd := []string{}
				if !hasSocat {
					packagesToAdd = append(packagesToAdd, "socat")
				}
				if !hasWget {
					packagesToAdd = append(packagesToAdd, "wget")
				}
				
				if len(packagesToAdd) > 0 {
					// Pattern: apt-get install -y --no-install-recommends netcat && ...
					if strings.Contains(trimmed, "--no-install-recommends") && strings.Contains(trimmed, "netcat") {
						// Replace "netcat" with "netcat socat wget"
						line = strings.Replace(line, "netcat", "netcat "+strings.Join(packagesToAdd, " "), 1)
						modified = true
					} else if strings.Contains(trimmed, "--no-install-recommends") {
						// Insert packages after --no-install-recommends, before && or \
						if strings.Contains(trimmed, "&&") {
							line = strings.Replace(line, "--no-install-recommends &&", "--no-install-recommends "+strings.Join(packagesToAdd, " ")+" &&", 1)
							modified = true
						} else if strings.HasSuffix(line, "\\") {
							line = strings.TrimSuffix(line, "\\") + " " + strings.Join(packagesToAdd, " ") + " \\"
							modified = true
						} else {
							line = strings.Replace(line, "--no-install-recommends", "--no-install-recommends "+strings.Join(packagesToAdd, " "), 1)
							modified = true
						}
					} else if strings.Contains(trimmed, "apt-get install -y") {
						// Simple case: add after -y flag
						if strings.Contains(trimmed, "&&") {
							line = strings.Replace(line, "apt-get install -y &&", "apt-get install -y "+strings.Join(packagesToAdd, " ")+" &&", 1)
							modified = true
						} else {
							line = strings.Replace(line, "apt-get install -y", "apt-get install -y "+strings.Join(packagesToAdd, " "), 1)
							modified = true
						}
					}
				}
			}
		}
		
		result = append(result, line)
	}
	
	// If no modification made in multi-line parsing, use simple replacement
	if !modified {
		enhanced := strings.Join(result, "\n")
		// Simple pattern replacement
		if !hasWget || !hasSocat {
			packagesToAdd := []string{}
			if !hasSocat {
				packagesToAdd = append(packagesToAdd, "socat")
			}
			if !hasWget {
				packagesToAdd = append(packagesToAdd, "wget")
			}
			
			if len(packagesToAdd) > 0 {
				pkgStr := strings.Join(packagesToAdd, " ")
				// Replace common patterns
				enhanced = strings.ReplaceAll(enhanced,
					"--no-install-recommends netcat &&",
					"--no-install-recommends netcat "+pkgStr+" &&")
				enhanced = strings.ReplaceAll(enhanced,
					"--no-install-recommends netcat \\",
					"--no-install-recommends netcat "+pkgStr+" \\")
				return enhanced
			}
		}
		return enhanced
	}
	
	return strings.Join(result, "\n")
}

// insertAt inserts a string at the given index in a slice
func insertAt(slice []string, index int, value string) []string {
	result := make([]string, 0, len(slice)+1)
	result = append(result, slice[:index]...)
	result = append(result, value)
	result = append(result, slice[index:]...)
	return result
}

// fixPoetryVersion fixes old Poetry versions and deprecated flags
func (g *DockerfileGenerator) fixPoetryVersion(content string) string {
	// Replace poetry==1.1 or poetry==1.0 with just poetry (latest)
	lines := strings.Split(content, "\n")
	result := make([]string, 0, len(lines))
	
	for _, line := range lines {
		// Fix old Poetry versions
		if strings.Contains(line, "poetry==1.0") || strings.Contains(line, "poetry==1.1") || 
			strings.Contains(line, "poetry==1.2") || strings.Contains(line, "poetry==1.3") {
			line = strings.ReplaceAll(line, "poetry==1.0", "poetry")
			line = strings.ReplaceAll(line, "poetry==1.1", "poetry")
			line = strings.ReplaceAll(line, "poetry==1.2", "poetry")
			line = strings.ReplaceAll(line, "poetry==1.3", "poetry")
		}
		
		// Fix deprecated --no-dev flag (replaced with --without dev in Poetry 2.0+)
		if strings.Contains(line, "poetry install") && strings.Contains(line, "--no-dev") {
			line = strings.ReplaceAll(line, "--no-dev", "--without dev")
		}
		
		// Ensure poetry config virtualenvs.create false is set
		if strings.Contains(line, "poetry install") && !strings.Contains(content, "poetry config virtualenvs.create false") {
			// Add config before install if not present
			if strings.Contains(line, "RUN") && strings.Contains(line, "poetry install") {
				// Split the RUN command and add config
				parts := strings.Split(line, "poetry install")
				if len(parts) == 2 {
					line = parts[0] + "poetry config virtualenvs.create false && poetry install" + parts[1]
				}
			}
		}
		result = append(result, line)
	}
	
	return strings.Join(result, "\n")
}

// addPortEnvVar adds PORT environment variable
func (g *DockerfileGenerator) addPortEnvVar(content string) string {
	lines := strings.Split(content, "\n")
	result := make([]string, 0, len(lines)+2)
	
	// Find a good place to add ENV PORT (after other ENV vars or after FROM)
	insertIndex := 0
	for i, line := range lines {
		result = append(result, line)
		// Insert after FROM or after first ENV
		if strings.HasPrefix(strings.TrimSpace(line), "FROM") {
			insertIndex = i + 1
		} else if insertIndex == 0 && strings.HasPrefix(strings.TrimSpace(line), "ENV") {
			insertIndex = i + 1
		}
	}
	
	// Insert ENV PORT after FROM or first ENV
	if insertIndex > 0 && insertIndex < len(result) {
		newResult := make([]string, 0, len(result)+2)
		newResult = append(newResult, result[:insertIndex]...)
		newResult = append(newResult, "", "ENV PORT=8080")
		newResult = append(newResult, result[insertIndex:]...)
		return strings.Join(newResult, "\n")
	}
	
	// Fallback: add at the beginning after FROM
	if len(result) > 0 {
		newResult := make([]string, 0, len(result)+2)
		newResult = append(newResult, result[0])
		newResult = append(newResult, "", "ENV PORT=8080")
		newResult = append(newResult, result[1:]...)
		return strings.Join(newResult, "\n")
	}
	
	return content
}

// addSocatForPortForwarding adds socat installation
func (g *DockerfileGenerator) addSocatForPortForwarding(content string) string {
	// Add socat to apt-get install if present (addWgetToAptInstall also adds socat)
	if strings.Contains(content, "apt-get install") {
		enhanced := g.addWgetToAptInstall(content)
		// Ensure socat is added
		if !strings.Contains(enhanced, "socat") {
			enhanced = strings.ReplaceAll(enhanced, 
				"apt-get install -y", 
				"apt-get install -y socat")
		}
		return enhanced
	}
	
	// If no apt-get, add a RUN command to install socat
	lines := strings.Split(content, "\n")
	result := make([]string, 0, len(lines)+5)
	
	// Find where to insert (after WORKDIR or before CMD)
	insertIndex := len(lines)
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "CMD") || strings.HasPrefix(strings.TrimSpace(line), "ENTRYPOINT") {
			insertIndex = i
			break
		}
	}
	
	// Build result
	for i, line := range lines {
		result = append(result, line)
		if i == insertIndex-1 && i < len(lines)-1 {
			// Insert socat installation
			result = append(result, "")
			result = append(result, "# Install socat for port forwarding (added by Stackyn)")
			result = append(result, "RUN apt-get update && apt-get install -y socat && rm -rf /var/lib/apt/lists/*")
		}
	}
	
	return strings.Join(result, "\n")
}

// addPortForwardingWrapper adds a port forwarding wrapper for Python apps
func (g *DockerfileGenerator) addPortForwardingWrapper(content string) string {
	lines := strings.Split(content, "\n")
	result := make([]string, 0, len(lines)+30)
	
	// Find CMD line
	cmdIndex := -1
	var originalCMD string
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "CMD") {
			cmdIndex = i
			originalCMD = line
			break
		}
	}
	
	if cmdIndex == -1 {
		// No CMD found, just return original
		return content
	}
	
	// Build result with wrapper
	for i, line := range lines {
		if i == cmdIndex {
			// Replace CMD with wrapper script creation and new CMD
			result = append(result, "")
			result = append(result, "# Port forwarding wrapper (added by Stackyn)")
			result = append(result, "RUN echo '#!/bin/sh' > /entrypoint.sh && \\")
			result = append(result, "    echo 'set +e' >> /entrypoint.sh && \\")
			result = append(result, "    echo 'APP_PORT=${APP_PORT:-8000}' >> /entrypoint.sh && \\")
			result = append(result, "    echo 'echo \"Stackyn: Starting app on port $APP_PORT...\"' >> /entrypoint.sh && \\")
			
			// Extract the actual command from CMD
			cmdContent := strings.TrimSpace(strings.TrimPrefix(originalCMD, "CMD"))
			cmdContent = strings.Trim(cmdContent, "[]\"'")
			
			// Add the original command to wrapper
			result = append(result, fmt.Sprintf("    echo '%s &' >> /entrypoint.sh && \\", cmdContent))
			result = append(result, "    echo 'APP_PID=$!' >> /entrypoint.sh && \\")
			result = append(result, "    echo 'sleep 3' >> /entrypoint.sh && \\")
			result = append(result, "    echo 'if [ \"$PORT\" != \"$APP_PORT\" ]; then' >> /entrypoint.sh && \\")
			result = append(result, "    echo '  socat TCP-LISTEN:8080,fork,reuseaddr TCP:localhost:$APP_PORT &' >> /entrypoint.sh && \\")
			result = append(result, "    echo 'fi' >> /entrypoint.sh && \\")
			result = append(result, "    echo 'wait $APP_PID' >> /entrypoint.sh && \\")
			result = append(result, "    chmod +x /entrypoint.sh")
			result = append(result, "")
			result = append(result, "CMD [\"/entrypoint.sh\"]")
		} else {
			result = append(result, line)
		}
	}
	
	return strings.Join(result, "\n")
}
