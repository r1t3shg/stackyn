#!/bin/bash

# Comprehensive Certificate Issue Diagnosis

set -e

echo "=========================================="
echo "Certificate Issue Diagnosis"
echo "=========================================="
echo ""

cd /opt/stackyn || { echo "❌ Not in /opt/stackyn directory"; exit 1; }

echo "1. Checking environment variable substitution..."
echo "   --- Checking if API_DOMAIN is substituted in docker-compose config ---"
docker-compose config | grep -A 5 "traefik.http.routers.api.rule" | head -10
echo ""

echo "2. Checking Traefik API for registered routers..."
echo "   --- HTTP Routers ---"
docker-compose exec traefik wget -qO- http://localhost:8080/api/http/routers 2>/dev/null | python3 -m json.tool 2>/dev/null | grep -A 10 -i "api" || echo "   Cannot access Traefik API or no API router found"
echo ""

echo "3. Checking Traefik logs for ACME errors..."
echo "   --- Last 50 lines with ACME/certificate mentions ---"
docker-compose logs traefik --tail=200 | grep -i -E "(acme|certificate|cert|letsencrypt|error|failed)" | tail -30
echo ""

echo "4. Checking if acme.json has any certificates..."
if docker-compose exec traefik test -f /letsencrypt/acme.json 2>/dev/null; then
    echo "   --- Checking acme.json content ---"
    docker-compose exec traefik cat /letsencrypt/acme.json | python3 -m json.tool 2>/dev/null | grep -A 5 -i "certificates" | head -20 || echo "   Cannot parse acme.json"
else
    echo "   ⚠️  acme.json does not exist"
fi
echo ""

echo "5. Testing HTTP challenge endpoint..."
echo "   --- Testing if Let's Encrypt can reach the server ---"
HTTP_TEST=$(curl -s -o /dev/null -w "%{http_code}" http://api.staging.stackyn.com/.well-known/acme-challenge/test 2>&1 || echo "000")
echo "   HTTP challenge test status: $HTTP_TEST"
echo "   (404 is OK - it means the server is reachable)"
echo ""

echo "6. Checking Traefik container environment..."
echo "   --- Traefik environment variables ---"
docker-compose exec traefik env | grep -i -E "(acme|email|domain)" || echo "   No relevant environment variables"
echo ""

echo "7. Verifying API container is running and has correct labels..."
echo "   --- API container status ---"
docker-compose ps api
echo ""
echo "   --- API container labels ---"
docker inspect stackyn-api 2>/dev/null | python3 -m json.tool | grep -A 20 "Labels" | grep -i traefik | head -15 || echo "   Cannot inspect API container"
echo ""

echo "=========================================="
echo "Key Things to Check:"
echo "=========================================="
echo "1. If API_DOMAIN is not substituted in docker-compose config:"
echo "   - Make sure .env file has API_DOMAIN=api.staging.stackyn.com"
echo "   - Restart containers: docker-compose down && docker-compose up -d"
echo ""
echo "2. If no API router in Traefik API:"
echo "   - API container labels might not be applied"
echo "   - Check: docker-compose config | grep -A 30 'api:'"
echo ""
echo "3. If ACME errors in logs:"
echo "   - Check for 'rate limit' (wait or use staging server)"
echo "   - Check for 'connection refused' (port 80 issue)"
echo "   - Check for 'DNS' errors (DNS not pointing correctly)"
echo ""

