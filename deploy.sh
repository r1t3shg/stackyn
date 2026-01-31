#!/bin/bash

# Safe Deployment Script for Stackyn
# This script ensures SSL certificates are preserved during deployment

set -e

echo "=========================================="
echo "Stackyn Safe Deployment Script"
echo "=========================================="
echo ""

cd /opt/stackyn || { echo "❌ Not in /opt/stackyn directory"; exit 1; }

# Check if SSL certificates exist
echo "1. Checking SSL certificate status..."
if docker compose exec traefik test -f /letsencrypt/acme.json 2>/dev/null; then
    CERT_SIZE=$(docker compose exec traefik stat -c%s /letsencrypt/acme.json 2>/dev/null || echo "0")
    if [ "$CERT_SIZE" -gt 100 ]; then
        echo "   ✓ SSL certificates found ($CERT_SIZE bytes)"
        # Backup certificates before deployment
        echo "   Creating backup..."
        docker compose exec traefik cp /letsencrypt/acme.json /letsencrypt/acme.json.backup.$(date +%Y%m%d_%H%M%S) 2>/dev/null || true
        echo "   ✓ Backup created"
    else
        echo "   ⚠️  acme.json exists but is small ($CERT_SIZE bytes) - may be empty"
    fi
else
    echo "   ⚠️  No SSL certificates found (will be generated on first HTTPS request)"
fi
echo ""

# Pull latest code
echo "2. Pulling latest code..."
if git rev-parse --git-dir > /dev/null 2>&1; then
    CURRENT_BRANCH=$(git branch --show-current)
    echo "   Current branch: $CURRENT_BRANCH"
    git pull origin "$CURRENT_BRANCH" || {
        echo "   ⚠️  Git pull failed, continuing with existing code..."
    }
else
    echo "   ⚠️  Not a git repository, skipping pull"
fi
echo ""

# Check if rebuild is needed
echo "3. Checking if rebuild is needed..."
REBUILD_NEEDED=false

# Check if docker-compose.yml changed
if git diff --quiet HEAD HEAD~1 docker-compose.yml 2>/dev/null; then
    echo "   docker-compose.yml unchanged"
else
    echo "   ⚠️  docker-compose.yml changed - rebuild recommended"
    REBUILD_NEEDED=true
fi

# Check if server code changed
if [ -d "server" ]; then
    if git diff --quiet HEAD HEAD~1 server/ 2>/dev/null; then
        echo "   Server code unchanged"
    else
        echo "   ⚠️  Server code changed - rebuild recommended"
        REBUILD_NEEDED=true
    fi
fi

# Check if frontend code changed
if [ -d "frontend" ]; then
    if git diff --quiet HEAD HEAD~1 frontend/ 2>/dev/null; then
        echo "   Frontend code unchanged"
    else
        echo "   ⚠️  Frontend code changed - rebuild recommended"
        REBUILD_NEEDED=true
    fi
fi

# Check if cms code changed
if [ -d "cms" ]; then
    if git diff --quiet HEAD HEAD~1 cms/ 2>/dev/null; then
        echo "   CMS code unchanged"
    else
        echo "   ⚠️  CMS code changed - rebuild recommended"
        REBUILD_NEEDED=true
    fi
fi
echo ""

# Ask user if they want to rebuild
if [ "$REBUILD_NEEDED" = true ]; then
    read -p "4. Rebuild containers? (y/n) " -n 1 -r
    echo ""
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo "   Rebuilding containers..."
        docker compose build
        echo "   ✓ Rebuild complete"
    else
        echo "   Skipping rebuild"
    fi
else
    echo "4. No rebuild needed"
fi
echo ""

# Restart services (preserves volumes)
echo "5. Restarting services (preserving volumes)..."
echo "   ⚠️  Using 'docker compose up -d' to preserve volumes"
echo "   ⚠️  NOT using 'docker compose down -v' (which would delete SSL certificates)"
docker compose up -d
echo "   ✓ Services restarted"
echo ""

# Wait for services to be healthy
echo "6. Waiting for services to be healthy..."
sleep 5

# Check service status
echo "   Service status:"
docker compose ps --format "table {{.Name}}\t{{.Status}}" | grep -E "(stackyn|NAME)" || true
echo ""

# Verify SSL certificates are still there
echo "7. Verifying SSL certificates are preserved..."
if docker compose exec traefik test -f /letsencrypt/acme.json 2>/dev/null; then
    NEW_CERT_SIZE=$(docker compose exec traefik stat -c%s /letsencrypt/acme.json 2>/dev/null || echo "0")
    if [ "$NEW_CERT_SIZE" -gt 100 ]; then
        echo "   ✓ SSL certificates preserved ($NEW_CERT_SIZE bytes)"
    else
        echo "   ⚠️  acme.json exists but is small - certificates may need regeneration"
    fi
else
    echo "   ⚠️  acme.json not found - certificates will be generated on next HTTPS request"
fi
echo ""

# Test HTTPS endpoint
echo "8. Testing HTTPS endpoint..."
if curl -s -o /dev/null -w "%{http_code}" --max-time 5 https://api.staging.stackyn.com/health 2>/dev/null | grep -q "200"; then
    echo "   ✓ HTTPS endpoint is working"
else
    echo "   ⚠️  HTTPS endpoint test failed (may need a moment to start)"
fi
echo ""

echo "=========================================="
echo "Deployment Complete!"
echo "=========================================="
echo ""
echo "Next steps:"
echo "1. Check service logs: docker compose logs -f"
echo "2. Verify SSL certificates: docker compose exec traefik cat /letsencrypt/acme.json | grep Certificates"
echo "3. Test endpoints: curl https://api.staging.stackyn.com/health"
echo ""
