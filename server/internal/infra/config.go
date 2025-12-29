package infra

import (
	"os"
	"strconv"
)

// Config holds application configuration
type Config struct {
	// Server configuration
	ServerAddr string
	ServerPort string

	// Database configuration
	DatabaseURL string

	// Logging configuration
	LogLevel string

	// Worker configuration
	WorkerConcurrency int
}

// LoadConfig loads configuration from environment variables
func LoadConfig() *Config {
	return &Config{
		ServerAddr:        getEnv("SERVER_ADDR", "0.0.0.0"),
		ServerPort:        getEnv("SERVER_PORT", "8080"),
		DatabaseURL:       getEnv("DATABASE_URL", ""),
		LogLevel:          getEnv("LOG_LEVEL", "info"),
		WorkerConcurrency: getEnvAsInt("WORKER_CONCURRENCY", 10),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

