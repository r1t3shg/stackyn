package infra

import (
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
	
	// Explicitly bind environment variables for email config
	viper.BindEnv("email.resend_api_key", "EMAIL_RESEND_API_KEY")
	viper.BindEnv("email.from_email", "EMAIL_FROM_EMAIL")

	// Set default values
	setDefaults()

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
			Host:     viper.GetString("redis.host"),
			Port:     viper.GetInt("redis.port"),
			Password: viper.GetString("redis.password"),
			DB:       viper.GetInt("redis.db"),
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
	}

	// Build computed connection strings
	config.Postgres.DSN = buildPostgresDSN(config.Postgres)
	config.Redis.Addr = buildRedisAddr(config.Redis)
	
	// Override Redis addr if explicitly set
	if viper.IsSet("redis.addr") {
		config.Redis.Addr = viper.GetString("redis.addr")
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
}

func buildPostgresDSN(pg PostgresConfig) string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		pg.Host, pg.Port, pg.User, pg.Password, pg.Database, pg.SSLMode)
}

func buildRedisAddr(redis RedisConfig) string {
	return fmt.Sprintf("%s:%d", redis.Host, redis.Port)
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
