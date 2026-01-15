#!/bin/bash
# Test frontend connectivity
# Run this on your VPS

echo "🔍 Testing Frontend Connectivity"
echo "================================"
echo ""

cd /opt/stackyn

# 1. Check if frontend is running
echo "1. Frontend container status:"
docker compose ps frontend
echo ""

# 2. Check frontend logs
echo "2. Frontend logs (last 20 lines):"
docker compose logs frontend --tail 20
echo ""

# 3. Test if frontend is listening on port 3000
echo "3. Checking if frontend is listening on port 3000:"
if docker compose exec frontend netstat -tlnp 2>/dev/null | grep -q ":3000"; then
    echo "   ✅ Frontend is listening on port 3000"
    docker compose exec frontend netstat -tlnp 2>/dev/null | grep ":3000"
elif docker compose exec frontend ss -tlnp 2>/dev/null | grep -q ":3000"; then
    echo "   ✅ Frontend is listening on port 3000 (ss)"
    docker compose exec frontend ss -tlnp 2>/dev/null | grep ":3000"
else
    echo "   ❌ Frontend is NOT listening on port 3000!"
fi
echo ""

# 4. Test direct access from host
echo "4. Testing direct access from host (localhost:3000):"
if curl -s -o /dev/null -w "%{http_code}" http://localhost:3000/ | grep -q "200"; then
    echo "   ✅ Frontend responds on localhost:3000"
    curl -s http://localhost:3000/ | head -c 200
    echo ""
else
    HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:3000/ || echo "000")
    echo "   ❌ Frontend returned: $HTTP_CODE"
fi
echo ""

# 5. Test from Traefik container
echo "5. Testing connectivity from Traefik container:"
if docker compose exec traefik wget -q -O- --timeout=5 http://frontend:3000/ 2>/dev/null | head -c 200 >/dev/null; then
    echo "   ✅ Traefik can reach frontend"
    echo "   Response preview:"
    docker compose exec traefik wget -q -O- --timeout=5 http://frontend:3000/ 2>/dev/null | head -c 200
    echo ""
else
    echo "   ❌ Traefik CANNOT reach frontend!"
    echo "   Testing network connectivity..."
    docker compose exec traefik ping -c 2 frontend 2>/dev/null || echo "   ❌ Cannot ping frontend"
    docker compose exec traefik nslookup frontend 2>/dev/null || echo "   ❌ Cannot resolve frontend"
fi
echo ""

# 6. Check Traefik service configuration
echo "6. Checking Traefik service configuration:"
SERVICE_CONFIG=$(curl -s http://localhost:8081/api/http/services/frontend 2>/dev/null)
if [ ! -z "$SERVICE_CONFIG" ] && echo "$SERVICE_CONFIG" | grep -q "frontend"; then
    echo "   ✅ Frontend service found in Traefik"
    echo "$SERVICE_CONFIG" | python3 -m json.tool 2>/dev/null | head -20 || echo "$SERVICE_CONFIG" | head -10
else
    echo "   ❌ Frontend service NOT found in Traefik!"
    echo "   Available services:"
    curl -s http://localhost:8081/api/http/services 2>/dev/null | python3 -m json.tool 2>/dev/null | grep -E '"name"|"server"' | head -10
fi
echo ""

# 7. Check Traefik router configuration
echo "7. Checking Traefik router configuration:"
ROUTER_CONFIG=$(curl -s http://localhost:8081/api/http/routers 2>/dev/null | python3 -m json.tool 2>/dev/null | grep -A 20 '"name".*frontend' | head -30)
if [ ! -z "$ROUTER_CONFIG" ]; then
    echo "   ✅ Frontend router found"
    echo "$ROUTER_CONFIG"
else
    echo "   ❌ Frontend router NOT found!"
fi
echo ""

# 8. Check network connectivity
echo "8. Checking Docker network:"
if docker network inspect stackyn-network 2>/dev/null | grep -q "frontend"; then
    echo "   ✅ Frontend is in stackyn-network"
    echo "   Frontend IP in network:"
    docker network inspect stackyn-network 2>/dev/null | grep -A 10 "frontend" | grep -E "IPv4Address|Name" | head -5
else
    echo "   ❌ Frontend is NOT in stackyn-network!"
fi
echo ""

# 9. Test actual site access
echo "9. Testing actual site access:"
echo "   HTTP:"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" http://localhost/ 2>&1)
echo "   Status: $HTTP_CODE"
if [ "$HTTP_CODE" != "200" ] && [ "$HTTP_CODE" != "301" ] && [ "$HTTP_CODE" != "302" ]; then
    echo "   Response:"
    curl -s http://localhost/ | head -c 200
    echo ""
fi

echo "   HTTPS:"
HTTPS_CODE=$(curl -k -s -o /dev/null -w "%{http_code}" https://localhost/ 2>&1)
echo "   Status: $HTTPS_CODE"
if [ "$HTTPS_CODE" != "200" ] && [ "$HTTPS_CODE" != "301" ] && [ "$HTTPS_CODE" != "302" ]; then
    echo "   Response:"
    curl -k -s https://localhost/ | head -c 200
    echo ""
fi
echo ""

echo "✅ Testing complete!"
echo ""
echo "If Traefik cannot reach frontend, try:"
echo "1. Rebuild frontend: docker compose build frontend"
echo "2. Restart frontend: docker compose restart frontend"
echo "3. Restart Traefik: docker compose restart traefik"

