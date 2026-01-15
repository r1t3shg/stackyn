#!/bin/bash
# Fix Traefik service registration
# Run this on your VPS

set -e

echo "🔧 Fixing Traefik Service Registration"
echo "======================================"
echo ""

cd /opt/stackyn

# 1. Check current service status
echo "1. Checking current service status..."
curl -s http://localhost:8081/api/http/services/frontend 2>/dev/null | python3 -m json.tool || echo "   Service not found"
echo ""

# 2. Check what labels Traefik sees
echo "2. Checking Docker labels on frontend container..."
docker inspect stackyn-frontend | grep -A 50 "Labels" | grep "traefik" | head -20
echo ""

# 3. Stop and recreate frontend to force Traefik to re-read labels
echo "3. Recreating frontend container to force Traefik to re-read labels..."
docker compose stop frontend
docker compose rm -f frontend
docker compose up -d frontend
sleep 5
echo "   ✅ Frontend recreated"
echo ""

# 4. Restart Traefik to ensure it picks up changes
echo "4. Restarting Traefik..."
docker compose restart traefik
sleep 10
echo "   ✅ Traefik restarted"
echo ""

# 5. Check if service is now registered
echo "5. Checking if service is now registered..."
sleep 5
SERVICE_CHECK=$(curl -s http://localhost:8081/api/http/services/frontend 2>/dev/null)
if echo "$SERVICE_CHECK" | grep -q "loadBalancer"; then
    echo "   ✅ Service is now registered!"
    echo "$SERVICE_CHECK" | python3 -m json.tool | head -20
else
    echo "   ❌ Service still not found"
    echo "   Response: $SERVICE_CHECK"
    echo ""
    echo "   Checking all services:"
    curl -s http://localhost:8081/api/http/services 2>/dev/null | python3 -m json.tool | grep -E '"name"|"server"' | head -20
fi
echo ""

# 6. Test HTTPS
echo "6. Testing HTTPS access..."
HTTPS_CODE=$(curl -k -s -o /dev/null -w "%{http_code}" https://staging.stackyn.com 2>&1)
echo "   HTTPS Status: $HTTPS_CODE"
if [ "$HTTPS_CODE" = "200" ]; then
    echo "   ✅ Site is working!"
    curl -k -s https://staging.stackyn.com | head -c 200
    echo ""
elif [ "$HTTPS_CODE" = "404" ]; then
    echo "   ❌ Still getting 404"
    echo "   Checking Traefik logs..."
    docker compose logs traefik --tail 20 | grep -i "frontend\|error"
else
    echo "   Status: $HTTPS_CODE"
fi
echo ""

echo "✅ Fix complete!"

