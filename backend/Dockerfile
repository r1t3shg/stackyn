########################################
# Stage 1: Build the Go binaries
########################################
FROM golang:1.23-alpine AS builder

# Install git (needed for some Go modules)
RUN apk add --no-cache git

# Set working directory inside the container
WORKDIR /app

# Copy module files first and download dependencies (better layer caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the API and worker binaries for Linux
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
   go build -o bin/api ./cmd/api && \
   CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
   go build -o bin/worker ./cmd/worker


########################################
# Stage 2: Runtime image
########################################
FROM alpine:3.20

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



