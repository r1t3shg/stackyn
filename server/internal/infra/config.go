package infra

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Config holds application configuration
type Config struct {
	// Server configuration
	Server ServerConfig

	// Database configuration
	Postgres PostgresConfig

	// Redis configuration
	Redis RedisConfig

	// Docker configuration
	Docker DockerConfig

	// Traefik configuration
	Traefik TraefikConfig

	// JWT configuration
	JWT JWTConfig

	// Logging configuration
	LogLevel string

	// Worker configuration
	WorkerConcurrency int

	// Email configuration
	Email EmailConfig

	// Lemon Squeezy configuration
	LemonSqueezy LemonSqueezyConfig
}

type ServerConfig struct {
	Addr string
	Port string
}

type PostgresConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string
	// Computed connection string
	DSN string
}

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
	// Computed connection string
	Addr string
}

type DockerConfig struct {
	Host       string
	APIVersion string
	TLSEnabled bool
	CertPath   string
	KeyPath    string
	CAPath     string
}

type TraefikConfig struct {
	APIURL      string
	EntryPoint  string
	NetworkName string
}

type JWTConfig struct {
	Secret     string
	Expiration int // in seconds
}

type EmailConfig struct {
	ResendAPIKey string
	FromEmail   string
}

type LemonSqueezyConfig struct {
	APIKey        string
	StoreID       string
	TestMode      bool
	TestVariantIDs map[string]string // Map of plan names to variant IDs for test mode
	LiveVariantIDs map[string]string // Map of plan names to variant IDs for live mode
	FrontendBaseURL string
	WebhookSecret string // Lemon Squeezy webhook signing secret
}

// LoadConfig loads configuration using viper with support for:
// - Environment variables
// - .env files
// - Default values
// Fails fast on missing required configs
func LoadConfig() (*Config, error) {
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath("../configs")

	// Enable environment variable support
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	
	// Explicitly bind environment variables for Redis (must be before setDefaults)
	viper.BindEnv("redis.host", "REDIS_HOST")
	viper.BindEnv("redis.port", "REDIS_PORT")
	viper.BindEnv("redis.password", "REDIS_PASSWORD")
	viper.BindEnv("redis.db", "REDIS_DB")
	viper.BindEnv("redis.addr", "REDIS_ADDR")
	
	// Explicitly bind environment variables for email config
	viper.BindEnv("email.resend_api_key", "EMAIL_RESEND_API_KEY")
	viper.BindEnv("email.from_email", "EMAIL_FROM_EMAIL")
	
	// Explicitly bind environment variables for Lemon Squeezy config
	viper.BindEnv("lemon_squeezy.api_key", "LEMON_API_KEY")
	viper.BindEnv("lemon_squeezy.store_id", "LEMON_STORE_ID")
	viper.BindEnv("lemon_squeezy.test_mode", "LEMON_TEST_MODE")
	viper.BindEnv("lemon_squeezy.frontend_base_url", "FRONTEND_BASE_URL")

	// Set default values (env vars will override these)
	setDefaults()
	
	// Force read environment variables by explicitly checking them
	// This ensures env vars take precedence over defaults
	if redisHost := os.Getenv("REDIS_HOST"); redisHost != "" {
		viper.Set("redis.host", redisHost)
	}
	if redisPort := os.Getenv("REDIS_PORT"); redisPort != "" {
		// Parse port as integer
		var port int
		if _, err := fmt.Sscanf(redisPort, "%d", &port); err == nil {
			viper.Set("redis.port", port)
		}
	}
	if redisPassword := os.Getenv("REDIS_PASSWORD"); redisPassword != "" {
		viper.Set("redis.password", redisPassword)
	}
	if redisDB := os.Getenv("REDIS_DB"); redisDB != "" {
		// Parse DB as integer
		var db int
		if _, err := fmt.Sscanf(redisDB, "%d", &db); err == nil {
			viper.Set("redis.db", db)
		}
	}

	// Try to read config file (optional - env vars take precedence)
	if err := viper.ReadInConfig(); err != nil {
		// Config file not found is OK if we have env vars
		// Check if it's a ConfigFileNotFoundError
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if errors.As(err, &configFileNotFoundError) {
			// File not found is fine - we'll use env vars and defaults
		} else {
			// Only error if it's not a "file not found" error
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	// Build config struct
	config := &Config{
		Server: ServerConfig{
			Addr: viper.GetString("server.addr"),
			Port: viper.GetString("server.port"),
		},
		Postgres: PostgresConfig{
			Host:     viper.GetString("postgres.host"),
			Port:     viper.GetInt("postgres.port"),
			User:     viper.GetString("postgres.user"),
			Password: viper.GetString("postgres.password"),
			Database: viper.GetString("postgres.database"),
			SSLMode:  viper.GetString("postgres.sslmode"),
		},
		Redis: RedisConfig{
			// Read directly from environment variables (bypass viper completely)
			Host: func() string {
				if h := os.Getenv("REDIS_HOST"); h != "" {
					return h
				}
				return "localhost"
			}(),
			Port: func() int {
				if p := os.Getenv("REDIS_PORT"); p != "" {
					var port int
					if _, err := fmt.Sscanf(p, "%d", &port); err == nil {
						return port
					}
				}
				return 6379
			}(),
			Password: os.Getenv("REDIS_PASSWORD"),
			DB: func() int {
				if d := os.Getenv("REDIS_DB"); d != "" {
					var db int
					if _, err := fmt.Sscanf(d, "%d", &db); err == nil {
						return db
					}
				}
				return 0
			}(),
		},
		Docker: DockerConfig{
			Host:       viper.GetString("docker.host"),
			APIVersion: viper.GetString("docker.api_version"),
			TLSEnabled: viper.GetBool("docker.tls_enabled"),
			CertPath:   viper.GetString("docker.cert_path"),
			KeyPath:    viper.GetString("docker.key_path"),
			CAPath:     viper.GetString("docker.ca_path"),
		},
		Traefik: TraefikConfig{
			APIURL:      viper.GetString("traefik.api_url"),
			EntryPoint:  viper.GetString("traefik.entry_point"),
			NetworkName: viper.GetString("traefik.network_name"),
		},
		JWT: JWTConfig{
			Secret:     viper.GetString("jwt.secret"),
			Expiration: viper.GetInt("jwt.expiration"),
		},
		LogLevel:          viper.GetString("log.level"),
		WorkerConcurrency: viper.GetInt("worker.concurrency"),
		Email: EmailConfig{
			// Check both dot notation and direct env var name
			ResendAPIKey: viper.GetString("email.resend_api_key"),
			FromEmail:   viper.GetString("email.from_email"),
		},
		LemonSqueezy: LemonSqueezyConfig{
			APIKey:        os.Getenv("LEMON_API_KEY"),
			StoreID:       os.Getenv("LEMON_STORE_ID"),
			TestMode:      os.Getenv("LEMON_TEST_MODE") == "true",
			TestVariantIDs: parseVariantIDs(os.Getenv("LEMON_TEST_VARIANT_IDS")),
			LiveVariantIDs: parseVariantIDs(os.Getenv("LEMON_LIVE_VARIANT_IDS")),
			FrontendBaseURL: os.Getenv("FRONTEND_BASE_URL"),
			WebhookSecret: os.Getenv("LEMON_WEBHOOK_SECRET"),
		},
	}

	// Build computed connection strings
	config.Postgres.DSN = buildPostgresDSN(config.Postgres)
	
	// Check for REDIS_ADDR first (most direct way to set Redis address)
	if redisAddr := os.Getenv("REDIS_ADDR"); redisAddr != "" {
		config.Redis.Addr = redisAddr
	} else {
		// Build from host and port
		config.Redis.Addr = buildRedisAddr(config.Redis)
	}

	// Validate required configs (fail fast)
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return config, nil
}

func setDefaults() {
	// Server defaults
	viper.SetDefault("server.addr", "0.0.0.0")
	viper.SetDefault("server.port", "8080")

	// Postgres defaults
	viper.SetDefault("postgres.host", "localhost")
	viper.SetDefault("postgres.port", 5432)
	viper.SetDefault("postgres.user", "postgres")
	viper.SetDefault("postgres.password", "")
	viper.SetDefault("postgres.database", "stackyn")
	viper.SetDefault("postgres.sslmode", "disable")

	// Redis defaults
	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", 6379)
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)
	viper.SetDefault("redis.addr", "localhost:6379") // Computed from host:port

	// Docker defaults
	viper.SetDefault("docker.host", "unix:///var/run/docker.sock")
	viper.SetDefault("docker.api_version", "1.43")
	viper.SetDefault("docker.tls_enabled", false)
	viper.SetDefault("docker.cert_path", "")
	viper.SetDefault("docker.key_path", "")
	viper.SetDefault("docker.ca_path", "")

	// Traefik defaults
	viper.SetDefault("traefik.api_url", "http://localhost:8080")
	viper.SetDefault("traefik.entry_point", "web")
	viper.SetDefault("traefik.network_name", "traefik")

	// JWT defaults
	viper.SetDefault("jwt.secret", "")
	viper.SetDefault("jwt.expiration", 3600) // 1 hour

	// Logging defaults
	viper.SetDefault("log.level", "info")

	// Worker defaults
	viper.SetDefault("worker.concurrency", 10)

	// Email defaults
	viper.SetDefault("email.resend_api_key", "")
	viper.SetDefault("email.from_email", "noreply@stackyn.com")
	
	// Lemon Squeezy defaults
	viper.SetDefault("lemon_squeezy.api_key", "")
	viper.SetDefault("lemon_squeezy.store_id", "")
	viper.SetDefault("lemon_squeezy.test_mode", false)
	viper.SetDefault("lemon_squeezy.frontend_base_url", "http://localhost:3000")
}

func buildPostgresDSN(pg PostgresConfig) string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		pg.Host, pg.Port, pg.User, pg.Password, pg.Database, pg.SSLMode)
}

func buildRedisAddr(redis RedisConfig) string {
	return fmt.Sprintf("%s:%d", redis.Host, redis.Port)
}

// getEnvWithDefault gets value from environment variable or returns default
func getEnvWithDefault(envKey, defaultValue string) string {
	if val := os.Getenv(envKey); val != "" {
		return val
	}
	return defaultValue
}

// getEnvIntWithDefault gets integer value from environment variable or returns default
func getEnvIntWithDefault(envKey string, defaultValue int) int {
	if val := os.Getenv(envKey); val != "" {
		var intVal int
		if _, err := fmt.Sscanf(val, "%d", &intVal); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func validateConfig(config *Config) error {
	var missing []string

	// Required: Postgres password (if not using default/local dev)
	if config.Postgres.Password == "" && config.Postgres.Host != "localhost" {
		missing = append(missing, "POSTGRES_PASSWORD")
	}

	// Required: Postgres database name
	if config.Postgres.Database == "" {
		missing = append(missing, "POSTGRES_DATABASE")
	}

	// Required: JWT secret (always required for security)
	if config.JWT.Secret == "" {
		missing = append(missing, "JWT_SECRET")
	}

	// Required: Docker host
	if config.Docker.Host == "" {
		missing = append(missing, "DOCKER_HOST")
	}

	// If TLS is enabled, require cert paths
	if config.Docker.TLSEnabled {
		if config.Docker.CertPath == "" {
			missing = append(missing, "DOCKER_CERT_PATH")
		}
		if config.Docker.KeyPath == "" {
			missing = append(missing, "DOCKER_KEY_PATH")
		}
		if config.Docker.CAPath == "" {
			missing = append(missing, "DOCKER_CA_PATH")
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required configuration: %s", strings.Join(missing, ", "))
	}

	return nil
}

// GetEnv returns the value of an environment variable or default
func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// parseVariantIDs parses variant IDs from environment variable
// Supports two formats:
// 1. JSON format: {"starter":"123","pro":"456"}
// 2. Comma-separated format: "starter=123,pro=456"
func parseVariantIDs(variantIDsStr string) map[string]string {
	result := make(map[string]string)
	if variantIDsStr == "" {
		return result
	}
	
	// Try parsing as JSON first
	var jsonMap map[string]string
	if err := json.Unmarshal([]byte(variantIDsStr), &jsonMap); err == nil {
		// Successfully parsed as JSON
		return jsonMap
	}
	
	// Fall back to comma-separated format: "starter=123,pro=456"
	pairs := strings.Split(variantIDsStr, ",")
	for _, pair := range pairs {
		// Split by equals sign
		parts := strings.SplitN(strings.TrimSpace(pair), "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			if key != "" && value != "" {
				result[key] = value
			}
		}
	}
	return result
}
