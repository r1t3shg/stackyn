# Stackyn MVP Error Catalog

This document describes the error catalog implementation for Stackyn MVP.

## Overview

The error catalog provides standardized error codes and user-friendly messages for all error scenarios in Stackyn MVP. Errors are returned to the frontend in a structured format and logged with proper context.

## Error Response Format

All API errors are returned in the following format:

```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "User-friendly error message",
    "details": "Additional context (optional)"
  }
}
```

## Error Codes

### Git & Repo Errors

- **REPO_NOT_FOUND**: Repository not found. Please check the GitHub URL and branch.
- **REPO_PRIVATE_UNSUPPORTED**: Private repositories are not supported in Stackyn MVP.
- **REPO_TOO_LARGE**: Repository is too large to build on Stackyn MVP.
- **MONOREPO_DETECTED**: Monorepos are not supported in Stackyn MVP.

### Build Detection Errors

- **RUNTIME_NOT_DETECTED**: Couldn't detect a supported runtime. Supported: Node.js, Python, Go, Java.
- **UNSUPPORTED_LANGUAGE**: This runtime is not supported yet.
- **CUSTOM_SYSTEM_DEPENDENCY**: This app requires system dependencies not supported in MVP.

### Docker / Buildpack Errors

- **DOCKERFILE_PRESENT**: Custom Dockerfiles are not supported in Stackyn MVP.
- **DOCKER_COMPOSE_PRESENT**: Multi-container apps are not supported in Stackyn MVP.
- **BUILD_FAILED**: Build failed during dependency installation.
- **BUILD_TIMEOUT**: Build exceeded the maximum allowed time.
- **IMAGE_TOO_LARGE**: Built image exceeds size limits.

### Runtime & Startup Errors

- **APP_CRASH_ON_START**: App crashed during startup.
- **PORT_NOT_LISTENING**: App must listen on the $PORT environment variable.
- **HARDCODED_PORT**: Hardcoded ports are not supported. Use $PORT.
- **MULTIPLE_PORTS_DETECTED**: Only one exposed port is allowed in Stackyn MVP.

### Resource Limit Errors

- **MEMORY_LIMIT_EXCEEDED**: App exceeded its memory limit.
- **CPU_LIMIT_EXCEEDED**: App exceeded allowed CPU usage.
- **DISK_LIMIT_EXCEEDED**: App exceeded ephemeral disk limits.

### Networking Errors

- **HEALTHCHECK_FAILED**: App failed health checks.
- **ROUTING_ERROR**: Routing error while exposing your app.
- **INTERNAL_NETWORK_ERROR**: Internal networking error occurred.

### Deployment Flow Errors

- **DEPLOY_LOCKED**: A deployment is already running for this app.
- **ZERO_DOWNTIME_NOT_SUPPORTED**: Zero-downtime deploys are not available on your plan.
- **PLAN_LIMIT_EXCEEDED**: You've reached the maximum number of apps for your plan.

### Logging & Observability Errors

- **LOG_STREAM_FAILED**: Failed to stream application logs.
- **LOGS_NOT_AVAILABLE**: Logs are unavailable because the app failed to start.

### Environment & Config Errors

- **ENV_VAR_MISSING**: Required environment variables are missing.
- **INVALID_ENV_VAR**: One or more environment variables are invalid.

### Platform / Infra Errors

- **HOST_OUT_OF_MEMORY**: Temporary infrastructure issue. Please retry later.
- **BUILD_NODE_UNAVAILABLE**: No build capacity available right now.
- **INTERNAL_PLATFORM_ERROR**: Something went wrong on Stackyn's side.

## Implementation

### Error Catalog Package

The error catalog is implemented in `server/internal/errors/catalog.go`:

- `ErrorCode`: Type for error codes
- `StackynError`: Structured error type with code, message, and details
- `New()`: Create a new error with code and optional details
- `Wrap()`: Wrap an existing error with a StackynError code

### Usage in Services

Services detect errors and return `StackynError` instances:

```go
import stackynerrors "stackyn/server/internal/errors"

// Example: Git service
if resp.StatusCode == http.StatusNotFound {
    return stackynerrors.New(stackynerrors.ErrorCodeRepoNotFound, 
        fmt.Sprintf("Repository %s not found", repoURL))
}
```

### Usage in API Handlers

Handlers use `handleError()` to process errors and return structured responses:

```go
if err != nil {
    h.handleError(w, r, err, http.StatusBadRequest)
    return
}
```

The `handleError()` function:
1. Checks if error is a `StackynError` and returns it directly
2. Maps error codes to appropriate HTTP status codes
3. Logs errors with full context (request_id, error_code, etc.)
4. Returns structured error response to frontend

### Logging

All errors are logged with:
- Error code
- Error message
- Error details
- Request ID (for API errors)
- App ID / Build Job ID (for task errors)
- Original error (if wrapped)

Example log entry:
```
{
  "level": "error",
  "error_code": "REPO_NOT_FOUND",
  "error_message": "Repository not found. Please check the GitHub URL and branch.",
  "error_details": "Repository https://github.com/user/repo not found",
  "request_id": "abc123",
  "app_id": "xyz789"
}
```

## Frontend Integration

The frontend receives errors in the standardized format and can:
1. Display user-friendly messages
2. Show error codes for debugging
3. Handle specific error types differently (e.g., retry for infrastructure errors)

## Future Enhancements

- Add error recovery suggestions
- Add error code documentation in API docs
- Add error analytics and monitoring
- Add error rate limiting per error type

