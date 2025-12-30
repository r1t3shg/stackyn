# Stackyn VPS Deployment Guide

This guide will help you deploy Stackyn to a VPS (Virtual Private Server).

## Prerequisites

- A VPS with Ubuntu 20.04+ or similar Linux distribution
- Root or sudo access
- At least 4GB RAM, 2 CPU cores, 50GB disk space
- Domain name (optional, for production)

## Step 1: Initial VPS Setup

### 1.1 Update System

```bash
sudo apt update && sudo apt upgrade -y
```

### 1.2 Install Basic Tools

```bash
sudo apt install -y git curl wget build-essential
```

## Step 2: Install Required Services

### 2.1 Install Go

```bash
# Download Go (check latest version at https://go.dev/dl/)
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz

# Add to PATH
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Verify installation
go version
```

### 2.2 Install Docker

```bash
# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh

# Add your user to docker group (replace $USER with your username)
sudo usermod -aG docker $USER

# Start Docker service
sudo systemctl start docker
sudo systemctl enable docker

# Verify installation
docker --version
```

### 2.3 Install PostgreSQL

```bash
# Install PostgreSQL
sudo apt install -y postgresql postgresql-contrib

# Start PostgreSQL
sudo systemctl start postgresql
sudo systemctl enable postgresql

# Create database and user
sudo -u postgres psql << EOF
CREATE DATABASE stackyn;
CREATE USER stackyn_user WITH PASSWORD 'your_secure_password_here';
GRANT ALL PRIVILEGES ON DATABASE stackyn TO stackyn_user;
\q
EOF
```

### 2.4 Install Redis

```bash
# Install Redis
sudo apt install -y redis-server

# Configure Redis (optional: set password)
sudo nano /etc/redis/redis.conf
# Find and uncomment: requirepass your_redis_password

# Start Redis
sudo systemctl start redis-server
sudo systemctl enable redis-server

# Verify
redis-cli ping
```

### 2.5 Install Traefik (Reverse Proxy)

```bash
# Create Traefik directory
sudo mkdir -p /opt/traefik
cd /opt/traefik

# Download Traefik (check latest version)
wget https://github.com/traefik/traefik/releases/download/v2.10/traefik_v2.10_linux_amd64.tar.gz
tar -xzf traefik_v2.10_linux_amd64.tar.gz
sudo mv traefik /usr/local/bin/
sudo chmod +x /usr/local/bin/traefik

# Create Traefik network
docker network create traefik
```

## Step 3: Clone and Setup Project

### 3.1 Clone Repository

```bash
# Create project directory
sudo mkdir -p /opt/stackyn
cd /opt/stackyn

# Clone your repository (replace with your repo URL)
git clone https://github.com/yourusername/stackyn.git .
# OR if you already have it, just pull:
git pull origin main
```

### 3.2 Setup Environment Variables

```bash
cd /opt/stackyn/server

# Copy example env file
cp configs/env.example .env

# Edit environment variables
nano .env
```

**Required Environment Variables:**

```bash
# Server Configuration
SERVER_ADDR=0.0.0.0
SERVER_PORT=8080

# Postgres Configuration
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=stackyn_user
POSTGRES_PASSWORD=your_secure_password_here
POSTGRES_DATABASE=stackyn
POSTGRES_SSLMODE=disable

# Redis Configuration
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=your_redis_password_here
REDIS_DB=0

# Docker Configuration
DOCKER_HOST=unix:///var/run/docker.sock
DOCKER_API_VERSION=1.43
DOCKER_TLS_ENABLED=false

# Traefik Configuration
TRAEFIK_API_URL=http://localhost:8080
TRAEFIK_ENTRY_POINT=web
TRAEFIK_NETWORK_NAME=traefik

# JWT Configuration (IMPORTANT: Generate a secure random string)
JWT_SECRET=$(openssl rand -base64 32)
JWT_EXPIRATION=3600

# Logging Configuration
LOG_LEVEL=info

# Worker Configuration
WORKER_CONCURRENCY=10
```

**Generate JWT Secret:**
```bash
openssl rand -base64 32
# Copy the output and paste it as JWT_SECRET in .env
```

## Step 4: Build Application

### 4.1 Build Go Binaries

```bash
cd /opt/stackyn/server

# Download dependencies
go mod download

# Build all binaries
go build -o bin/api ./cmd/api
go build -o bin/build-worker ./cmd/build-worker
go build -o bin/deploy-worker ./cmd/deploy-worker
go build -o bin/cleanup-worker ./cmd/cleanup-worker

# Verify binaries were created
ls -lh bin/
```

### 4.2 Run Database Migrations

```bash
cd /opt/stackyn/server

# Set database URL (or use .env file)
export DATABASE_URL="postgres://stackyn_user:your_password@localhost:5432/stackyn?sslmode=disable"

# Run migrations (if you have a migrate command)
# OR manually run migrations using golang-migrate
# Install golang-migrate:
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Run migrations
migrate -path ./internal/db/migrations -database "$DATABASE_URL" up
```

## Step 5: Create Systemd Services

### 5.1 Create API Service

```bash
sudo nano /etc/systemd/system/stackyn-api.service
```

**Content:**
```ini
[Unit]
Description=Stackyn API Server
After=network.target postgresql.service redis-server.service docker.service

[Service]
Type=simple
User=root
WorkingDirectory=/opt/stackyn/server
EnvironmentFile=/opt/stackyn/server/.env
ExecStart=/opt/stackyn/server/bin/api
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
```

### 5.2 Create Build Worker Service

```bash
sudo nano /etc/systemd/system/stackyn-build-worker.service
```

**Content:**
```ini
[Unit]
Description=Stackyn Build Worker
After=network.target postgresql.service redis-server.service docker.service

[Service]
Type=simple
User=root
WorkingDirectory=/opt/stackyn/server
EnvironmentFile=/opt/stackyn/server/.env
ExecStart=/opt/stackyn/server/bin/build-worker
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
```

### 5.3 Create Deploy Worker Service

```bash
sudo nano /etc/systemd/system/stackyn-deploy-worker.service
```

**Content:**
```ini
[Unit]
Description=Stackyn Deploy Worker
After=network.target postgresql.service redis-server.service docker.service

[Service]
Type=simple
User=root
WorkingDirectory=/opt/stackyn/server
EnvironmentFile=/opt/stackyn/server/.env
ExecStart=/opt/stackyn/server/bin/deploy-worker
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
```

### 5.4 Create Cleanup Worker Service

```bash
sudo nano /etc/systemd/system/stackyn-cleanup-worker.service
```

**Content:**
```ini
[Unit]
Description=Stackyn Cleanup Worker
After=network.target postgresql.service redis-server.service docker.service

[Service]
Type=simple
User=root
WorkingDirectory=/opt/stackyn/server
EnvironmentFile=/opt/stackyn/server/.env
ExecStart=/opt/stackyn/server/bin/cleanup-worker
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
```

### 5.5 Enable and Start Services

```bash
# Reload systemd
sudo systemctl daemon-reload

# Enable services (start on boot)
sudo systemctl enable stackyn-api
sudo systemctl enable stackyn-build-worker
sudo systemctl enable stackyn-deploy-worker
sudo systemctl enable stackyn-cleanup-worker

# Start services
sudo systemctl start stackyn-api
sudo systemctl start stackyn-build-worker
sudo systemctl start stackyn-deploy-worker
sudo systemctl start stackyn-cleanup-worker

# Check status
sudo systemctl status stackyn-api
sudo systemctl status stackyn-build-worker
sudo systemctl status stackyn-deploy-worker
sudo systemctl status stackyn-cleanup-worker
```

## Step 6: Setup Traefik (Optional but Recommended)

### 6.1 Create Traefik Configuration

```bash
sudo mkdir -p /opt/traefik
sudo nano /opt/traefik/traefik.yml
```

**Content:**
```yaml
api:
  dashboard: true
  insecure: true

entryPoints:
  web:
    address: ":80"

providers:
  docker:
    endpoint: "unix:///var/run/docker.sock"
    network: "traefik"
    exposedByDefault: false
```

### 6.2 Create Traefik Systemd Service

```bash
sudo nano /etc/systemd/system/traefik.service
```

**Content:**
```ini
[Unit]
Description=Traefik Reverse Proxy
After=network.target docker.service

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/traefik --configfile=/opt/traefik/traefik.yml
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable traefik
sudo systemctl start traefik
```

## Step 7: Verify Deployment

### 7.1 Check Service Logs

```bash
# API logs
sudo journalctl -u stackyn-api -f

# Build worker logs
sudo journalctl -u stackyn-build-worker -f

# Deploy worker logs
sudo journalctl -u stackyn-deploy-worker -f

# Cleanup worker logs
sudo journalctl -u stackyn-cleanup-worker -f
```

### 7.2 Test API

```bash
# Test API health (if you have a health endpoint)
curl http://localhost:8080/health

# Or test any endpoint
curl http://localhost:8080/api/v1/apps
```

### 7.3 Check Services Status

```bash
# Check all services
sudo systemctl status stackyn-api stackyn-build-worker stackyn-deploy-worker stackyn-cleanup-worker

# Check Docker
docker ps

# Check PostgreSQL
sudo -u postgres psql -c "SELECT version();"

# Check Redis
redis-cli ping
```

## Step 8: Firewall Configuration

```bash
# Allow HTTP (if using Traefik)
sudo ufw allow 80/tcp

# Allow HTTPS (if using SSL)
sudo ufw allow 443/tcp

# Allow API port (if exposing directly)
sudo ufw allow 8080/tcp

# Enable firewall
sudo ufw enable
```

## Step 9: Frontend Deployment (Optional)

If you want to deploy the frontend:

```bash
cd /opt/stackyn/frontend

# Install dependencies
npm install

# Build (set API URL)
export VITE_API_BASE_URL=http://your-vps-ip:8080
npm run build

# Serve with nginx or use Docker
# See frontend/Dockerfile for Docker deployment
```

## Step 10: Update Deployment Script

Create a simple update script:

```bash
sudo nano /opt/stackyn/update.sh
```

**Content:**
```bash
#!/bin/bash
set -e

cd /opt/stackyn
echo "Pulling latest changes..."
git pull origin main

cd server
echo "Building binaries..."
go build -o bin/api ./cmd/api
go build -o bin/build-worker ./cmd/build-worker
go build -o bin/deploy-worker ./cmd/deploy-worker
go build -o bin/cleanup-worker ./cmd/cleanup-worker

echo "Restarting services..."
sudo systemctl restart stackyn-api
sudo systemctl restart stackyn-build-worker
sudo systemctl restart stackyn-deploy-worker
sudo systemctl restart stackyn-cleanup-worker

echo "Deployment complete!"
```

```bash
chmod +x /opt/stackyn/update.sh
```

## Troubleshooting

### Services Not Starting

```bash
# Check logs
sudo journalctl -u stackyn-api -n 50

# Check environment variables
sudo systemctl show stackyn-api --property=Environment

# Test binary manually
cd /opt/stackyn/server
./bin/api
```

### Database Connection Issues

```bash
# Test PostgreSQL connection
psql -h localhost -U stackyn_user -d stackyn

# Check PostgreSQL is running
sudo systemctl status postgresql

# Check PostgreSQL logs
sudo tail -f /var/log/postgresql/postgresql-*.log
```

### Docker Issues

```bash
# Check Docker is running
sudo systemctl status docker

# Test Docker access
docker ps

# Check Docker socket permissions
ls -la /var/run/docker.sock
```

### Redis Issues

```bash
# Test Redis connection
redis-cli ping

# Check Redis is running
sudo systemctl status redis-server
```

## Quick Reference Commands

```bash
# Restart all services
sudo systemctl restart stackyn-api stackyn-build-worker stackyn-deploy-worker stackyn-cleanup-worker

# Stop all services
sudo systemctl stop stackyn-api stackyn-build-worker stackyn-deploy-worker stackyn-cleanup-worker

# View logs
sudo journalctl -u stackyn-api -f

# Check service status
sudo systemctl status stackyn-api

# Update and restart
/opt/stackyn/update.sh
```

## Security Recommendations

1. **Change default passwords** - Use strong passwords for PostgreSQL and Redis
2. **Use firewall** - Only expose necessary ports
3. **Use SSL/TLS** - Setup Let's Encrypt for HTTPS
4. **Regular updates** - Keep system and dependencies updated
5. **Backup database** - Setup regular PostgreSQL backups
6. **Monitor logs** - Regularly check service logs for issues

## Next Steps

- Setup SSL certificates with Let's Encrypt
- Configure domain name DNS
- Setup monitoring and alerting
- Configure automated backups
- Setup log rotation

