package errors

import (
	"fmt"
)

// ErrorCode represents a standardized error code for Stackyn MVP
type ErrorCode string

// Error codes for Stackyn MVP
const (
	// Git & Repo Errors
	ErrorCodeRepoNotFound            ErrorCode = "REPO_NOT_FOUND"
	ErrorCodeRepoPrivateUnsupported  ErrorCode = "REPO_PRIVATE_UNSUPPORTED"
	ErrorCodeRepoTooLarge            ErrorCode = "REPO_TOO_LARGE"
	ErrorCodeMonorepoDetected        ErrorCode = "MONOREPO_DETECTED"

	// Build Detection Errors
	ErrorCodeRuntimeNotDetected      ErrorCode = "RUNTIME_NOT_DETECTED"
	ErrorCodeUnsupportedLanguage     ErrorCode = "UNSUPPORTED_LANGUAGE"
	ErrorCodeCustomSystemDependency  ErrorCode = "CUSTOM_SYSTEM_DEPENDENCY"

	// Docker / Buildpack Errors
	ErrorCodeDockerfilePresent        ErrorCode = "DOCKERFILE_PRESENT"
	ErrorCodeDockerComposePresent     ErrorCode = "DOCKER_COMPOSE_PRESENT"
	ErrorCodeBuildFailed             ErrorCode = "BUILD_FAILED"
	ErrorCodeBuildTimeout            ErrorCode = "BUILD_TIMEOUT"
	ErrorCodeImageTooLarge           ErrorCode = "IMAGE_TOO_LARGE"

	// Runtime & Startup Errors
	ErrorCodeAppCrashOnStart         ErrorCode = "APP_CRASH_ON_START"
	ErrorCodePortNotListening        ErrorCode = "PORT_NOT_LISTENING"
	ErrorCodeHardcodedPort           ErrorCode = "HARDCODED_PORT"
	ErrorCodeMultiplePortsDetected   ErrorCode = "MULTIPLE_PORTS_DETECTED"

	// Resource Limit Errors
	ErrorCodeMemoryLimitExceeded     ErrorCode = "MEMORY_LIMIT_EXCEEDED"
	ErrorCodeCPULimitExceeded       ErrorCode = "CPU_LIMIT_EXCEEDED"
	ErrorCodeDiskLimitExceeded      ErrorCode = "DISK_LIMIT_EXCEEDED"

	// Networking Errors
	ErrorCodeHealthcheckFailed       ErrorCode = "HEALTHCHECK_FAILED"
	ErrorCodeRoutingError            ErrorCode = "ROUTING_ERROR"
	ErrorCodeInternalNetworkError    ErrorCode = "INTERNAL_NETWORK_ERROR"

	// Deployment Flow Errors
	ErrorCodeDeployLocked            ErrorCode = "DEPLOY_LOCKED"
	ErrorCodeZeroDowntimeNotSupported ErrorCode = "ZERO_DOWNTIME_NOT_SUPPORTED"
	ErrorCodePlanLimitExceeded       ErrorCode = "PLAN_LIMIT_EXCEEDED"

	// Logging & Observability Errors
	ErrorCodeLogStreamFailed         ErrorCode = "LOG_STREAM_FAILED"
	ErrorCodeLogsNotAvailable        ErrorCode = "LOGS_NOT_AVAILABLE"

	// Environment & Config Errors
	ErrorCodeEnvVarMissing           ErrorCode = "ENV_VAR_MISSING"
	ErrorCodeInvalidEnvVar           ErrorCode = "INVALID_ENV_VAR"

	// Platform / Infra Errors
	ErrorCodeHostOutOfMemory         ErrorCode = "HOST_OUT_OF_MEMORY"
	ErrorCodeBuildNodeUnavailable     ErrorCode = "BUILD_NODE_UNAVAILABLE"
	ErrorCodeInternalPlatformError    ErrorCode = "INTERNAL_PLATFORM_ERROR"
)

// Error messages map
var errorMessages = map[ErrorCode]string{
	// Git & Repo Errors
	ErrorCodeRepoNotFound:            "Repository not found. Please check the GitHub URL and branch.",
	ErrorCodeRepoPrivateUnsupported:   "Private repositories are not supported in Stackyn MVP.",
	ErrorCodeRepoTooLarge:            "Repository is too large to build on Stackyn MVP.",
	ErrorCodeMonorepoDetected:        "Monorepos are not supported in Stackyn MVP.",

	// Build Detection Errors
	ErrorCodeRuntimeNotDetected:      "Couldn't detect a supported runtime. Supported: Node.js, Python, Go, Java.",
	ErrorCodeUnsupportedLanguage:     "This runtime is not supported yet.",
	ErrorCodeCustomSystemDependency:   "This app requires system dependencies not supported in MVP.",

	// Docker / Buildpack Errors
	ErrorCodeDockerfilePresent:       "Custom Dockerfiles are not supported in Stackyn MVP.",
	ErrorCodeDockerComposePresent:    "Multi-container apps are not supported in Stackyn MVP.",
	ErrorCodeBuildFailed:             "Build failed during dependency installation.",
	ErrorCodeBuildTimeout:            "Build exceeded the maximum allowed time.",
	ErrorCodeImageTooLarge:           "Built image exceeds size limits.",

	// Runtime & Startup Errors
	ErrorCodeAppCrashOnStart:         "App crashed during startup.",
	ErrorCodePortNotListening:        "App must listen on the $PORT environment variable.",
	ErrorCodeHardcodedPort:           "Hardcoded ports are not supported. Use $PORT.",
	ErrorCodeMultiplePortsDetected:   "Only one exposed port is allowed in Stackyn MVP.",

	// Resource Limit Errors
	ErrorCodeMemoryLimitExceeded:     "App exceeded its memory limit.",
	ErrorCodeCPULimitExceeded:        "App exceeded allowed CPU usage.",
	ErrorCodeDiskLimitExceeded:       "App exceeded ephemeral disk limits.",

	// Networking Errors
	ErrorCodeHealthcheckFailed:       "App failed health checks.",
	ErrorCodeRoutingError:            "Routing error while exposing your app.",
	ErrorCodeInternalNetworkError:    "Internal networking error occurred.",

	// Deployment Flow Errors
	ErrorCodeDeployLocked:            "A deployment is already running for this app.",
	ErrorCodeZeroDowntimeNotSupported: "Zero-downtime deploys are not available on your plan.",
	ErrorCodePlanLimitExceeded:       "You've reached the maximum number of apps for your plan.",

	// Logging & Observability Errors
	ErrorCodeLogStreamFailed:         "Failed to stream application logs.",
	ErrorCodeLogsNotAvailable:        "Logs are unavailable because the app failed to start.",

	// Environment & Config Errors
	ErrorCodeEnvVarMissing:           "Required environment variables are missing.",
	ErrorCodeInvalidEnvVar:           "One or more environment variables are invalid.",

	// Platform / Infra Errors
	ErrorCodeHostOutOfMemory:          "Temporary infrastructure issue. Please retry later.",
	ErrorCodeBuildNodeUnavailable:     "No build capacity available right now.",
	ErrorCodeInternalPlatformError:    "Something went wrong on Stackyn's side.",
}

// StackynError represents a structured error with code and message
type StackynError struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Details string    `json:"details,omitempty"` // Additional context for debugging
	Err     error     `json:"-"`                 // Original error (not serialized)
}

// Error implements the error interface
func (e *StackynError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("[%s] %s: %s", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap returns the underlying error
func (e *StackynError) Unwrap() error {
	return e.Err
}

// New creates a new StackynError with the given code
func New(code ErrorCode, details ...string) *StackynError {
	msg, ok := errorMessages[code]
	if !ok {
		msg = "An unknown error occurred."
	}

	err := &StackynError{
		Code:    code,
		Message: msg,
	}

	if len(details) > 0 {
		err.Details = details[0]
	}

	return err
}

// Wrap wraps an existing error with a StackynError code
func Wrap(code ErrorCode, err error, details ...string) *StackynError {
	if err == nil {
		return New(code, details...)
	}
	stackynErr := New(code, details...)
	stackynErr.Err = err
	if err != nil {
		if stackynErr.Details == "" {
			stackynErr.Details = err.Error()
		} else {
			stackynErr.Details = fmt.Sprintf("%s: %s", stackynErr.Details, err.Error())
		}
	}
	return stackynErr
}

// GetMessage returns the user-friendly message for an error code
func GetMessage(code ErrorCode) string {
	if msg, ok := errorMessages[code]; ok {
		return msg
	}
	return "An unknown error occurred."
}

// IsStackynError checks if an error is a StackynError
func IsStackynError(err error) bool {
	_, ok := err.(*StackynError)
	return ok
}

// AsStackynError converts an error to StackynError if possible
func AsStackynError(err error) (*StackynError, bool) {
	if err == nil {
		return nil, false
	}
	stackynErr, ok := err.(*StackynError)
	return stackynErr, ok
}

