package services

import (
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/zap"
)

// DockerfileGenerator generates Dockerfiles for different runtimes
type DockerfileGenerator struct {
	logger *zap.Logger
}

// NewDockerfileGenerator creates a new Dockerfile generator
func NewDockerfileGenerator(logger *zap.Logger) *DockerfileGenerator {
	return &DockerfileGenerator{
		logger: logger,
	}
}

// GenerateDockerfile generates a Dockerfile for the given runtime
func (g *DockerfileGenerator) GenerateDockerfile(repoPath string, runtime Runtime) error {
	// Check if Dockerfile already exists
	dockerfilePath := filepath.Join(repoPath, "Dockerfile")
	if _, err := os.Stat(dockerfilePath); err == nil {
		g.logger.Info("Dockerfile already exists, skipping generation", zap.String("path", dockerfilePath))
		return nil
	}

	var content string
	switch runtime {
	case RuntimeNodeJS:
		content = g.generateNodeJSDockerfile(repoPath)
	case RuntimePython:
		content = g.generatePythonDockerfile(repoPath)
	case RuntimeGo:
		content = g.generateGoDockerfile(repoPath)
	case RuntimeRuby:
		content = g.generateRubyDockerfile(repoPath)
	case RuntimeJava:
		content = g.generateJavaDockerfile(repoPath)
	case RuntimePHP:
		content = g.generatePHPDockerfile(repoPath)
	case RuntimeStatic:
		content = g.generateStaticDockerfile(repoPath)
	default:
		return fmt.Errorf("unsupported runtime: %s", runtime)
	}

	// Write Dockerfile
	if err := os.WriteFile(dockerfilePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write Dockerfile: %w", err)
	}

	g.logger.Info("Generated Dockerfile", zap.String("path", dockerfilePath), zap.String("runtime", string(runtime)))
	return nil
}

func (g *DockerfileGenerator) generateNodeJSDockerfile(repoPath string) string {
	// Check for package.json to determine if it's a monorepo or single app
	// Default to Node 18 LTS
	return `FROM node:18-alpine AS builder

WORKDIR /app

# Copy package files
COPY package*.json ./

# Install dependencies
RUN npm ci --only=production

# Copy application code
COPY . .

# Build if needed (for Next.js, React, etc.)
RUN if [ -f "package.json" ] && grep -q "\"build\"" package.json; then npm run build; fi

# Production stage
FROM node:18-alpine

WORKDIR /app

# Copy dependencies and built files
COPY --from=builder /app/node_modules ./node_modules
COPY --from=builder /app .

# Expose port (default 3000, can be overridden)
EXPOSE 3000

# Start command
CMD ["node", "index.js"]
`
}

func (g *DockerfileGenerator) generatePythonDockerfile(repoPath string) string {
	return `FROM python:3.11-slim

WORKDIR /app

# Install dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy application code
COPY . .

# Expose port (default 8000, can be overridden)
EXPOSE 8000

# Start command (adjust based on framework)
CMD ["python", "app.py"]
`
}

func (g *DockerfileGenerator) generateGoDockerfile(repoPath string) string {
	return `FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o app .

# Production stage
FROM alpine:latest

WORKDIR /app

# Copy binary
COPY --from=builder /app/app .

# Expose port (default 8080, can be overridden)
EXPOSE 8080

CMD ["./app"]
`
}

func (g *DockerfileGenerator) generateRubyDockerfile(repoPath string) string {
	return `FROM ruby:3.2-alpine

WORKDIR /app

# Install dependencies
COPY Gemfile Gemfile.lock ./
RUN bundle install

# Copy application code
COPY . .

# Expose port (default 3000, can be overridden)
EXPOSE 3000

CMD ["bundle", "exec", "rackup", "-o", "0.0.0.0"]
`
}

func (g *DockerfileGenerator) generateJavaDockerfile(repoPath string) string {
	return `FROM maven:3.9-eclipse-temurin-17 AS builder

WORKDIR /app

# Copy pom.xml and build
COPY pom.xml .
RUN mvn dependency:go-offline

# Copy source and build
COPY src ./src
RUN mvn package -DskipTests

# Production stage
FROM eclipse-temurin:17-jre-alpine

WORKDIR /app

# Copy JAR
COPY --from=builder /app/target/*.jar app.jar

# Expose port (default 8080, can be overridden)
EXPOSE 8080

CMD ["java", "-jar", "app.jar"]
`
}

func (g *DockerfileGenerator) generatePHPDockerfile(repoPath string) string {
	return `FROM php:8.2-apache

WORKDIR /var/www/html

# Install dependencies
COPY composer.json composer.lock ./
RUN composer install --no-dev --optimize-autoloader

# Copy application code
COPY . .

# Expose port
EXPOSE 80

CMD ["apache2-foreground"]
`
}

func (g *DockerfileGenerator) generateStaticDockerfile(repoPath string) string {
	return `FROM nginx:alpine

WORKDIR /usr/share/nginx/html

# Copy static files
COPY . .

# Expose port
EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
`
}

