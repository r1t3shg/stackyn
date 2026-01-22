#!/bin/bash

echo "=========================================="
echo "SSL Certificate Diagnostic Script"
echo "=========================================="
echo ""

# Check if we're in the right directory
if [ ! -f "docker-compose.yml" ]; then
    echo "❌ Error: docker-compose.yml not found. Please run this script from /opt/stackyn"
    exit 1
fi

echo "1. Checking Traefik container status..."
docker-compose ps traefik
echo ""

echo "2. Checking Traefik logs for certificate-related messages..."
echo "--- Recent certificate/ACME logs ---"
docker-compose logs traefik --tail=50 | grep -i -E "(cert|certificate|acme|letsencrypt|error)" || echo "No certificate-related logs found"
echo ""

echo "3. Checking DNS configuration..."
echo "--- DNS for api.staging.stackyn.com ---"
dig +short api.staging.stackyn.com || nslookup api.staging.stackyn.com || echo "dig/nslookup not available"
echo ""

echo "4. Checking if port 80 is accessible..."
echo "--- Testing HTTP connection to api.staging.stackyn.com ---"
curl -I http://api.staging.stackyn.com 2>&1 | head -5 || echo "curl failed or port 80 not accessible"
echo ""

echo "5. Checking Traefik certificate storage..."
echo "--- Checking acme.json ---"
docker-compose exec traefik ls -la /letsencrypt/ 2>/dev/null || echo "Cannot access Traefik container"
docker-compose exec traefik cat /letsencrypt/acme.json 2>/dev/null | grep -o "api.staging.stackyn.com" | head -1 && echo "✓ Certificate found in storage" || echo "✗ Certificate not found in storage"
echo ""

echo "6. Checking environment variables in docker-compose..."
echo "--- API_DOMAIN check ---"
grep -E "API_DOMAIN|api.staging" docker-compose.yml | head -5 || echo "Not found in docker-compose.yml"
echo ""

echo "7. Testing HTTPS connection (this will show the certificate error)..."
echo "--- Testing HTTPS ---"
curl -v https://api.staging.stackyn.com/health 2>&1 | grep -i -E "(certificate|SSL|TLS|error)" | head -10 || echo "Connection failed"
echo ""

echo "=========================================="
echo "Diagnostic Complete"
echo "=========================================="
echo ""
echo "Next steps:"
echo "1. If DNS is not pointing to your server IP, fix DNS records"
echo "2. If port 80 is not accessible, open it: sudo ufw allow 80/tcp"
echo "3. If certificate is not being generated, restart Traefik: docker-compose restart traefik"
echo "4. Watch Traefik logs: docker-compose logs -f traefik"

