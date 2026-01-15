# Configuration

This directory contains configuration files for the Stackyn backend.

## Configuration Loading

The backend uses [Viper](https://github.com/spf13/viper) for configuration management with support for:

1. **Environment Variables** (highest priority)
2. **`.env` files** (medium priority)
3. **Default values** (lowest priority)

## Configuration Sources

Viper will look for configuration files in the following order:
1. Current directory (`.`)
2. `./configs` directory
3. `../configs` directory

The config file should be named `.env` and use the `env` format.

## Environment Variables

All configuration can be set via environment variables. Use uppercase with underscores, replacing dots with underscores:

- `SERVER_ADDR` → `server.addr`
- `POSTGRES_HOST` → `postgres.host`
- `JWT_SECRET` → `jwt.secret`

## Required Configuration

The following configuration values are **required** and the application will fail to start if they are missing:

- `JWT_SECRET` - Secret key for JWT token signing (always required for security)
- `POSTGRES_DATABASE` - Database name
- `DOCKER_HOST` - Docker daemon connection string

If Docker TLS is enabled (`DOCKER_TLS_ENABLED=true`), the following are also required:
- `DOCKER_CERT_PATH` - Path to Docker TLS certificate
- `DOCKER_KEY_PATH` - Path to Docker TLS key
- `DOCKER_CA_PATH` - Path to Docker CA certificate

## Example Configuration

See `env.example` for a complete example configuration file.

## Quick Start

1. Copy the example file:
   ```bash
   cp configs/env.example .env
   ```

2. Edit `.env` and set required values:
   ```bash
   # Generate a secure JWT secret
   JWT_SECRET=$(openssl rand -hex 32)
   
   # Set database password
   POSTGRES_PASSWORD=your_secure_password
   ```

3. Or set environment variables:
   ```bash
   export JWT_SECRET=$(openssl rand -hex 32)
   export POSTGRES_PASSWORD=your_secure_password
   ```

## Configuration Structure

### Server
- `SERVER_ADDR` - Server bind address (default: `0.0.0.0`)
- `SERVER_PORT` - Server port (default: `8080`)

### Postgres
- `POSTGRES_HOST` - Database host (default: `localhost`)
- `POSTGRES_PORT` - Database port (default: `5432`)
- `POSTGRES_USER` - Database user (default: `postgres`)
- `POSTGRES_PASSWORD` - Database password (required if not localhost)
- `POSTGRES_DATABASE` - Database name (default: `stackyn`, required)
- `POSTGRES_SSLMODE` - SSL mode (default: `disable`)

### Redis
- `REDIS_HOST` - Redis host (default: `localhost`)
- `REDIS_PORT` - Redis port (default: `6379`)
- `REDIS_PASSWORD` - Redis password (optional)
- `REDIS_DB` - Redis database number (default: `0`)

### Docker
- `DOCKER_HOST` - Docker daemon connection (default: `unix:///var/run/docker.sock`, required)
- `DOCKER_API_VERSION` - Docker API version (default: `1.43`)
- `DOCKER_TLS_ENABLED` - Enable TLS for Docker (default: `false`)
- `DOCKER_CERT_PATH` - TLS certificate path (required if TLS enabled)
- `DOCKER_KEY_PATH` - TLS key path (required if TLS enabled)
- `DOCKER_CA_PATH` - TLS CA path (required if TLS enabled)

### Traefik
- `TRAEFIK_API_URL` - Traefik API URL (default: `http://localhost:8080`)
- `TRAEFIK_ENTRY_POINT` - Traefik entry point name (default: `web`)
- `TRAEFIK_NETWORK_NAME` - Docker network name (default: `traefik`)

### JWT
- `JWT_SECRET` - Secret key for JWT signing (**REQUIRED**)
- `JWT_EXPIRATION` - Token expiration in seconds (default: `3600`)

### Logging
- `LOG_LEVEL` - Log level: `debug`, `info`, `warn`, `error` (default: `info`)

### Workers
- `WORKER_CONCURRENCY` - Number of concurrent workers (default: `10`)

