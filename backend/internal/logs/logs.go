// Package logs provides utilities for parsing and processing deployment logs.
// Currently focused on parsing Docker build logs and runtime logs from streams.
package logs

import (
	"bufio"
	"encoding/binary"
	"io"
	"strings"
)

// ParseBuildLog reads a stream of build logs and converts it to a single string.
// This is used to capture Docker build output and store it in the database.
// The reader is automatically closed when the function returns.
//
// Parameters:
//   - reader: An io.ReadCloser containing the build log stream (typically from Docker build output)
//
// Returns:
//   - string: All log lines joined with newlines, or empty string on error
//   - error: Error if reading or scanning fails
func ParseBuildLog(reader io.ReadCloser) (string, error) {
	// Ensure the reader is closed when we're done
	defer reader.Close()

	// Store all log lines in a slice
	var logLines []string
	
	// Use a scanner to read line by line (more efficient than reading all at once)
	scanner := bufio.NewScanner(reader)

	// Read each line from the stream
	for scanner.Scan() {
		line := scanner.Text()
		logLines = append(logLines, line)
	}

	// Check for scanning errors (not EOF, which is normal)
	if err := scanner.Err(); err != nil {
		return "", err
	}

	// Join all lines with newline characters to create the full log
	return strings.Join(logLines, "\n"), nil
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
			break
		}

		// Read the 8-byte header
		stream := data[offset]
		// Skip padding bytes (offset+1 to offset+3)
		// Read size as big-endian uint32 (offset+4 to offset+7)
		size := binary.BigEndian.Uint32(data[offset+4 : offset+8])

		offset += 8 // Move past header

		// Read the message
		if offset+int(size) > len(data) {
			// Not enough data, break
			break
		}

		message := data[offset : offset+int(size)]
		offset += int(size)

		// Convert message to string and split by newlines
		messageStr := string(message)
		lines := strings.Split(messageStr, "\n")
		for _, line := range lines {
			line = strings.TrimRight(line, "\r")
			// Add stream prefix for stderr
			if stream == 2 {
				line = "[stderr] " + line
			}
			if line != "" {
				logLines = append(logLines, line)
			}
		}
	}

	// Join all lines with newline characters
	return strings.Join(logLines, "\n"), nil
}

