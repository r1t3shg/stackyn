// Package logs provides utilities for parsing and processing deployment logs.
// Currently focused on parsing Docker build logs and runtime logs from streams.
package logs

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"io"
	"strings"
)

// BuildLogResult represents the result of parsing a Docker build log
type BuildLogResult struct {
	Log      string // Full build log as string
	HasError bool   // Whether the build log contains errors
	ErrorMsg string // Error message if build failed
}

// ParseBuildLog reads a stream of build logs, converts it to a string, and detects build failures.
// Docker build logs are in JSON format where each line is a JSON object with fields like:
//   - "stream": stdout/stderr output
//   - "error": error message if build failed
//   - "errorDetail": detailed error information
// The reader is automatically closed when the function returns.
//
// Parameters:
//   - reader: An io.ReadCloser containing the build log stream (typically from Docker build output)
//
// Returns:
//   - *BuildLogResult: Parsed build log with error detection, or nil on read error
//   - error: Error if reading or scanning fails
func ParseBuildLog(reader io.ReadCloser) (*BuildLogResult, error) {
	// Ensure the reader is closed when we're done
	defer reader.Close()

	// Store all log lines in a slice
	var logLines []string
	var hasError bool
	var errorMsg string
	
	// Use a scanner to read line by line (more efficient than reading all at once)
	scanner := bufio.NewScanner(reader)

	// Read each line from the stream
	for scanner.Scan() {
		line := scanner.Text()
		logLines = append(logLines, line)
		
		// Docker build logs are JSON - parse each line to check for errors
		var logEntry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &logEntry); err == nil {
			// Check for error field
			if errorVal, ok := logEntry["error"].(string); ok && errorVal != "" {
				hasError = true
				errorMsg = errorVal
			}
			// Check for errorDetail.message as fallback
			if !hasError {
				if errorDetail, ok := logEntry["errorDetail"].(map[string]interface{}); ok {
					if msg, ok := errorDetail["message"].(string); ok && msg != "" {
						hasError = true
						errorMsg = msg
					}
				}
			}
		}
	}

	// Check for scanning errors (not EOF, which is normal)
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Join all lines with newline characters to create the full log
	fullLog := strings.Join(logLines, "\n")
	
	return &BuildLogResult{
		Log:      fullLog,
		HasError: hasError,
		ErrorMsg: errorMsg,
	}, nil
}

// ParseBuildLogLegacy is a legacy function that returns just the log string for backward compatibility.
// It calls ParseBuildLog and returns only the log content.
func ParseBuildLogLegacy(reader io.ReadCloser) (string, error) {
	result, err := ParseBuildLog(reader)
	if err != nil {
		return "", err
	}
	return result.Log, nil
}

// ParseRuntimeLog reads a stream of runtime logs (Docker container logs) and converts it to a single string.
// Docker container logs use an 8-byte header format: [stream (1 byte)] [padding (3 bytes)] [size (4 bytes)] [message]
// Stream: 0=stdin, 1=stdout, 2=stderr
// This function parses this format and extracts the actual log messages.
// The reader is automatically closed when the function returns.
//
// Parameters:
//   - reader: An io.ReadCloser containing the container log stream (from Docker ContainerLogs API)
//
// Returns:
//   - string: All log lines joined with newlines, or empty string on error
//   - error: Error if reading fails
func ParseRuntimeLog(reader io.ReadCloser) (string, error) {
	// Ensure the reader is closed when we're done
	defer reader.Close()

	// Read all data from the stream
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}

	if len(data) == 0 {
		return "", nil
	}

	// Docker container logs format: 8-byte header followed by message
	// Header: [stream: 1 byte] [padding: 3 bytes] [size: 4 bytes (big-endian)]
	var logLines []string
	offset := 0

	for offset < len(data) {
		// Need at least 8 bytes for header
		if offset+8 > len(data) {
			// Not enough data for a complete header, skip remaining bytes
			break
		}

		// Read the 8-byte header
		stream := data[offset]
		// Skip padding bytes (offset+1 to offset+3)
		// Read size as big-endian uint32 (offset+4 to offset+7)
		size := binary.BigEndian.Uint32(data[offset+4 : offset+8])

		offset += 8 // Move past header

		// Validate size to prevent reading beyond data bounds
		if size == 0 {
			// Empty message, skip
			continue
		}
		
		if size > uint32(len(data)-offset) {
			// Size is larger than remaining data, this is likely corrupted
			// Try to read what we can and break
			if offset < len(data) {
				remaining := data[offset:]
				messageStr := string(remaining)
				if strings.TrimSpace(messageStr) != "" {
					line := strings.TrimRight(messageStr, "\r\n")
					if stream == 2 {
						line = "[stderr] " + line
					}
					if line != "" {
						logLines = append(logLines, line)
					}
				}
			}
			break
		}

		// Read the message
		message := data[offset : offset+int(size)]
		offset += int(size)

		// Convert message to string and split by newlines
		// Docker logs can contain both stdout (stream=1) and stderr (stream=2)
		// console.log in Node.js writes to stdout, so stream will be 1
		messageStr := string(message)
		lines := strings.Split(messageStr, "\n")
		for _, line := range lines {
			line = strings.TrimRight(line, "\r")
			// Only add prefix for stderr (stream=2), stdout (stream=1) and stdin (stream=0) are displayed as-is
			// This ensures console.log output (stdout) appears cleanly without prefixes
			if stream == 2 {
				line = "[stderr] " + line
			}
			// Skip empty lines but include all non-empty log lines (including stdout from console.log)
			if line != "" {
				logLines = append(logLines, line)
			}
		}
	}

	// Join all lines with newline characters
	return strings.Join(logLines, "\n"), nil
}

