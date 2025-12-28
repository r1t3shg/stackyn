# Backend Dockerfile - Go API Server and Worker
# Multi-stage build for optimized production image
#
# This Dockerfile builds both the API server and worker binaries.
# The same image can run either service depending on the CMD override.
#
# Build Instructions:
#   docker build -t stackyn-backend .
#
# Run API Server:
#   docker run -p 8080:8080 stackyn-backend /app/api
#
# Run Worker:
#   docker run stackyn-backend /app/worker
#
# Environment Variables:
#   - DATABASE_URL: PostgreSQL connection string
#   - DOCKER_HOST: Docker daemon address (unix:///var/run/docker.sock)
#   - BASE_DOMAIN: Base domain for subdomain routing
#   - PORT: HTTP server port (API only, default: 8080)
#
# Security Notes:
#   - Runs as non-root user (appuser) by default
#   - Worker may need root access for Docker socket (override USER if needed)
#   - CGO disabled for static binary (smaller image, no C dependencies)

########################################
# Stage 1: Build the Go binaries
########################################
FROM golang:1.24-alpine AS builder

# Install git (needed for some Go modules)
RUN apk add --no-cache git

# Set working directory inside the container
WORKDIR /app

# Set GOTOOLCHAIN to use local version (prevents auto-upgrade to non-existent versions)
ENV GOTOOLCHAIN=local

# Copy module files first and download dependencies (better layer caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Update go.sum for any replace directives (but keep Go version as specified in go.mod)
RUN go mod tidy

# Build the API and worker binaries for Linux
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
   go build -o bin/api ./cmd/api && \
   CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
   go build -o bin/worker ./cmd/worker


########################################
# Stage 2: Runtime image
########################################
FROM alpine:3.20

# Install git (needed for cloning repositories)
# Install curl and tar (needed for downloading pack CLI)
RUN apk add --no-cache git curl tar

# Install pack CLI for Cloud Native Buildpacks (CNB)
# Pack CLI is required for CNB-based container image builds
# Download the latest pack CLI binary for Linux (Alpine uses musl, but pack works fine)
RUN PACK_VERSION="v0.39.1" && \
    curl -LO https://github.com/buildpacks/pack/releases/download/${PACK_VERSION}/pack-${PACK_VERSION}-linux.tgz && \
    tar -xzf pack-${PACK_VERSION}-linux.tgz && \
    chmod +x pack && \
    mv pack /usr/local/bin/pack && \
    rm pack-${PACK_VERSION}-linux.tgz && \
    pack version

# Create non-root user for security
RUN adduser -D -g '' appuser

WORKDIR /app

# Copy binaries from builder image
COPY --from=builder /app/bin/api /app/api
COPY --from=builder /app/bin/worker /app/worker

# Run as non-root user
USER appuser

# Default API port
EXPOSE 8080
ENV PORT=8080

# Environment variables expected by the app
# (override these at runtime as needed)
# ENV DATABASE_URL=postgres://user:password@host:5432/mvp?sslmode=disable
# ENV DOCKER_HOST=unix:///var/run/docker.sock
# ENV BASE_DOMAIN=localhost

# Default command: run the API server
# To run the worker instead, override CMD to ["/app/worker"]
CMD ["/app/api"]



