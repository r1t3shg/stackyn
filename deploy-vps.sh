#!/bin/bash

# ============================================
# Stackyn VPS Deployment Script
# ============================================
# This script helps deploy Stackyn to a VPS
# Usage: ./deploy-vps.sh

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Stackyn VPS Deployment Script${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""

# Check if .env file exists
if [ ! -f .env ]; then
    echo -e "${YELLOW}⚠️  .env file not found!${NC}"
    echo "Creating .env from .env.production.example..."
    
    if [ -f .env.production.example ]; then
        cp .env.production.example .env
        echo -e "${GREEN}✅ Created .env file${NC}"
        echo -e "${YELLOW}⚠️  IMPORTANT: Edit .env file and set all required values!${NC}"
        echo "   - POSTGRES_PASSWORD"
        echo "   - REDIS_PASSWORD"
        echo "   - JWT_SECRET"
        echo "   - ACME_EMAIL"
        echo "   - APP_BASE_DOMAIN"
        echo ""
        echo "Press Enter to continue after editing .env file..."
        read
    else
        echo -e "${RED}❌ .env.production.example not found!${NC}"
        exit 1
    fi
fi

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo -e "${RED}❌ Docker is not installed!${NC}"
    echo "Please install Docker first: https://docs.docker.com/get-docker/"
    exit 1
fi

# Check if Docker Compose is installed
if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
    echo -e "${RED}❌ Docker Compose is not installed!${NC}"
    echo "Please install Docker Compose first"
    exit 1
fi

# Check required environment variables
echo -e "${GREEN}Checking environment variables...${NC}"
source .env

REQUIRED_VARS=("POSTGRES_PASSWORD" "JWT_SECRET" "API_DOMAIN" "APP_BASE_DOMAIN" "ACME_EMAIL")
MISSING_VARS=()

for var in "${REQUIRED_VARS[@]}"; do
    if [ -z "${!var}" ] || [[ "${!var}" == *"<"* ]]; then
        MISSING_VARS+=("$var")
    fi
done

if [ ${#MISSING_VARS[@]} -ne 0 ]; then
    echo -e "${RED}❌ Missing or unset required environment variables:${NC}"
    for var in "${MISSING_VARS[@]}"; do
        echo -e "   ${RED}- $var${NC}"
    done
    echo ""
    echo "Please edit .env file and set all required values."
    exit 1
fi

echo -e "${GREEN}✅ All required environment variables are set${NC}"
echo ""

# Generate passwords if needed
if [[ "$POSTGRES_PASSWORD" == *"<"* ]] || [ -z "$POSTGRES_PASSWORD" ]; then
    echo -e "${YELLOW}Generating POSTGRES_PASSWORD...${NC}"
    NEW_PASSWORD=$(openssl rand -base64 32)
    sed -i "s|POSTGRES_PASSWORD=.*|POSTGRES_PASSWORD=$NEW_PASSWORD|" .env
    echo -e "${GREEN}✅ Generated POSTGRES_PASSWORD${NC}"
fi

if [[ "$REDIS_PASSWORD" == *"<"* ]] || [ -z "$REDIS_PASSWORD" ]; then
    echo -e "${YELLOW}Generating REDIS_PASSWORD...${NC}"
    NEW_PASSWORD=$(openssl rand -base64 32)
    sed -i "s|REDIS_PASSWORD=.*|REDIS_PASSWORD=$NEW_PASSWORD|" .env
    echo -e "${GREEN}✅ Generated REDIS_PASSWORD${NC}"
fi

if [[ "$JWT_SECRET" == *"<"* ]] || [ -z "$JWT_SECRET" ]; then
    echo -e "${YELLOW}Generating JWT_SECRET...${NC}"
    NEW_SECRET=$(openssl rand -base64 32)
    sed -i "s|JWT_SECRET=.*|JWT_SECRET=$NEW_SECRET|" .env
    echo -e "${GREEN}✅ Generated JWT_SECRET${NC}"
fi

echo ""

# Build images
echo -e "${GREEN}Building Docker images...${NC}"
docker compose build

echo ""

# Start services
echo -e "${GREEN}Starting services...${NC}"
docker compose up -d

echo ""

# Wait for services to be healthy
echo -e "${GREEN}Waiting for services to be healthy...${NC}"
sleep 10

# Check service status
echo -e "${GREEN}Service Status:${NC}"
docker compose ps

echo ""

# Check if services are running
if docker compose ps | grep -q "Up"; then
    echo -e "${GREEN}✅ Services are running!${NC}"
    echo ""
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}Deployment Complete!${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo ""
    echo "Next steps:"
    echo "1. Verify DNS configuration:"
    echo "   - staging.stackyn.com → $(curl -s ifconfig.me 2>/dev/null || echo 'YOUR_VPS_IP')"
    echo "   - api.staging.stackyn.com → $(curl -s ifconfig.me 2>/dev/null || echo 'YOUR_VPS_IP')"
    echo "   - *.staging.stackyn.com → $(curl -s ifconfig.me 2>/dev/null || echo 'YOUR_VPS_IP')"
    echo ""
    echo "2. Check service logs:"
    echo "   docker compose logs -f"
    echo ""
    echo "3. Verify SSL certificates (may take 5-10 minutes):"
    echo "   docker compose logs traefik | grep -i acme"
    echo ""
    echo "4. Test endpoints:"
    echo "   - Landing: https://staging.stackyn.com"
    echo "   - API: https://api.staging.stackyn.com/health"
    echo ""
else
    echo -e "${RED}❌ Some services failed to start!${NC}"
    echo "Check logs with: docker compose logs"
    exit 1
fi
