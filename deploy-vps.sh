#!/bin/bash
# Stackyn VPS Quick Deployment Script
# This script helps automate the deployment process

set -e

echo "=========================================="
echo "Stackyn VPS Deployment Script"
echo "=========================================="
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if running as root
if [ "$EUID" -ne 0 ]; then 
    echo -e "${RED}Please run as root (use sudo)${NC}"
    exit 1
fi

# Variables
PROJECT_DIR="/opt/stackyn"
SERVER_DIR="$PROJECT_DIR/server"
ENV_FILE="$SERVER_DIR/.env"

echo -e "${GREEN}Step 1: Checking prerequisites...${NC}"

# Check Go
if ! command -v go &> /dev/null; then
    echo -e "${YELLOW}Go not found. Please install Go first.${NC}"
    exit 1
fi
echo "✓ Go installed: $(go version)"

# Check Docker
if ! command -v docker &> /dev/null; then
    echo -e "${YELLOW}Docker not found. Please install Docker first.${NC}"
    exit 1
fi
echo "✓ Docker installed: $(docker --version)"

# Check PostgreSQL
if ! command -v psql &> /dev/null; then
    echo -e "${YELLOW}PostgreSQL not found. Please install PostgreSQL first.${NC}"
    exit 1
fi
echo "✓ PostgreSQL installed"

# Check Redis
if ! command -v redis-cli &> /dev/null; then
    echo -e "${YELLOW}Redis not found. Please install Redis first.${NC}"
    exit 1
fi
echo "✓ Redis installed"

echo ""
echo -e "${GREEN}Step 2: Setting up project directory...${NC}"

# Create project directory if it doesn't exist
if [ ! -d "$PROJECT_DIR" ]; then
    mkdir -p "$PROJECT_DIR"
    echo "Created directory: $PROJECT_DIR"
fi

# Check if git repo exists
if [ ! -d "$PROJECT_DIR/.git" ]; then
    echo -e "${YELLOW}Git repository not found in $PROJECT_DIR${NC}"
    echo "Please clone your repository first:"
    echo "  cd /opt && git clone <your-repo-url> stackyn"
    exit 1
fi

cd "$PROJECT_DIR"
echo "✓ Project directory ready"

echo ""
echo -e "${GREEN}Step 3: Setting up environment variables...${NC}"

cd "$SERVER_DIR"

# Check if .env exists
if [ ! -f "$ENV_FILE" ]; then
    if [ -f "configs/env.example" ]; then
        cp configs/env.example "$ENV_FILE"
        echo "✓ Created .env from example"
        echo -e "${YELLOW}IMPORTANT: Please edit $ENV_FILE and set your configuration values${NC}"
        echo "Press Enter to continue after editing .env file..."
        read
    else
        echo -e "${RED}No env.example found. Please create .env manually.${NC}"
        exit 1
    fi
else
    echo "✓ .env file already exists"
fi

echo ""
echo -e "${GREEN}Step 4: Building Go binaries...${NC}"

cd "$SERVER_DIR"

# Download dependencies
echo "Downloading Go dependencies..."
go mod download

# Build binaries
echo "Building API server..."
go build -o bin/api ./cmd/api

echo "Building build worker..."
go build -o bin/build-worker ./cmd/build-worker

echo "Building deploy worker..."
go build -o bin/deploy-worker ./cmd/deploy-worker

echo "Building cleanup worker..."
go build -o bin/cleanup-worker ./cmd/cleanup-worker

echo "✓ All binaries built successfully"

echo ""
echo -e "${GREEN}Step 5: Creating systemd services...${NC}"

# Create systemd service files
cat > /etc/systemd/system/stackyn-api.service << 'EOF'
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
EOF

cat > /etc/systemd/system/stackyn-build-worker.service << 'EOF'
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
EOF

cat > /etc/systemd/system/stackyn-deploy-worker.service << 'EOF'
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
EOF

cat > /etc/systemd/system/stackyn-cleanup-worker.service << 'EOF'
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
EOF

echo "✓ Systemd service files created"

echo ""
echo -e "${GREEN}Step 6: Enabling and starting services...${NC}"

# Reload systemd
systemctl daemon-reload

# Enable services
systemctl enable stackyn-api
systemctl enable stackyn-build-worker
systemctl enable stackyn-deploy-worker
systemctl enable stackyn-cleanup-worker

echo "✓ Services enabled"

# Ask if user wants to start services now
read -p "Start services now? (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    systemctl start stackyn-api
    systemctl start stackyn-build-worker
    systemctl start stackyn-deploy-worker
    systemctl start stackyn-cleanup-worker
    
    echo "✓ Services started"
    
    # Wait a moment and check status
    sleep 2
    
    echo ""
    echo -e "${GREEN}Service Status:${NC}"
    systemctl status stackyn-api --no-pager -l
    systemctl status stackyn-build-worker --no-pager -l
    systemctl status stackyn-deploy-worker --no-pager -l
    systemctl status stackyn-cleanup-worker --no-pager -l
fi

echo ""
echo -e "${GREEN}=========================================="
echo "Deployment Complete!"
echo "==========================================${NC}"
echo ""
echo "Useful commands:"
echo "  Check status:  sudo systemctl status stackyn-api"
echo "  View logs:     sudo journalctl -u stackyn-api -f"
echo "  Restart:       sudo systemctl restart stackyn-api"
echo ""
echo "See DEPLOYMENT.md for more details and troubleshooting."

