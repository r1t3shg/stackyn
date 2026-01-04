#!/bin/bash
# Emergency fix script for staging.stackyn.com
# Run this on the VPS: bash fix-staging.sh

set -e

echo "🔧 Fixing staging.stackyn.com - Starting emergency repair..."

cd /opt/stackyn || { echo "❌ Error: /opt/stackyn not found"; exit 1; }

echo "📦 Step 1: Pulling latest changes..."
git pull origin develop || echo "⚠️  Warning: git pull failed, continuing anyway..."

echo "🛑 Step 2: Stopping all services..."
docker compose down

echo "🧹 Step 3: Cleaning up old containers and networks..."
docker container prune -f
docker network prune -f

echo "🔨 Step 4: Rebuilding frontend (this may take a few minutes)..."
docker compose build --no-cache frontend

echo "🚀 Step 5: Starting all services..."
docker compose up -d

echo "⏳ Step 6: Waiting for services to be healthy (30 seconds)..."
sleep 30

echo "📊 Step 7: Checking service status..."
docker compose ps

echo "🔍 Step 8: Checking if frontend is running..."
if docker ps | grep -q stackyn-frontend; then
    echo "✅ Frontend container is running"
else
    echo "❌ Frontend container is NOT running"
    echo "📋 Frontend logs:"
    docker compose logs frontend --tail 50
    exit 1
fi

echo "🌐 Step 9: Checking Traefik routers..."
sleep 5
ROUTERS=$(curl -s http://localhost:8081/api/http/routers 2>/dev/null | python3 -c "import sys, json; data=json.load(sys.stdin); routers=[r for r in data if 'frontend' in r.get('name', '').lower()]; print('Found', len(routers), 'frontend routers'); [print(f\"  - {r['name']}: {r.get('rule', 'N/A')}\") for r in routers]" 2>/dev/null || echo "Could not parse routers")
echo "$ROUTERS"

echo "🔒 Step 10: Testing HTTPS endpoint..."
sleep 5
if curl -k -I https://staging.stackyn.com 2>&1 | grep -q "HTTP"; then
    echo "✅ HTTPS endpoint is responding"
else
    echo "⚠️  HTTPS endpoint not responding yet (certificates may still be provisioning)"
    echo "   This is normal - Let's Encrypt certificates take 1-2 minutes to issue"
fi

echo ""
echo "✅ Fix script completed!"
echo ""
echo "📝 Next steps:"
echo "   1. Wait 1-2 minutes for Let's Encrypt certificates to be issued"
echo "   2. Check Traefik logs: docker compose logs traefik | grep -i acme"
echo "   3. Test the site: curl -I https://staging.stackyn.com"
echo "   4. If still not working, check: docker compose logs frontend"
echo ""

