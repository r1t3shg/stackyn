#!/bin/bash

# Check Traefik Certificate Status
# Run this on your server: /opt/stackyn

echo "=========================================="
echo "Traefik Certificate Status Check"
echo "=========================================="
echo ""

echo "1. Checking Traefik logs for ACME/certificate errors..."
echo "--- Recent ACME logs ---"
docker-compose logs traefik --tail=200 | grep -i -E "(acme|certificate|cert|letsencrypt|error|failed|obtained)" | tail -30
echo ""

echo "2. Checking if acme.json exists and has content..."
if docker-compose exec traefik test -f /letsencrypt/acme.json 2>/dev/null; then
    echo "✓ acme.json exists"
    ACME_SIZE=$(docker-compose exec traefik stat -c%s /letsencrypt/acme.json 2>/dev/null || echo "0")
    if [ "$ACME_SIZE" -gt 100 ]; then
        echo "✓ acme.json has content ($ACME_SIZE bytes)"
        # Check if it contains the domain
        if docker-compose exec traefik grep -q "api.staging.stackyn.com" /letsencrypt/acme.json 2>/dev/null; then
            echo "✓ Domain found in acme.json"
        else
            echo "⚠️  Domain NOT found in acme.json"
        fi
    else
        echo "⚠️  acme.json is too small ($ACME_SIZE bytes) - may be empty or corrupted"
    fi
else
    echo "❌ acme.json does not exist"
fi
echo ""

echo "3. Checking Traefik configuration..."
docker-compose exec traefik traefik version 2>/dev/null || echo "Cannot access Traefik container"
echo ""

echo "4. Checking Traefik API for certificate status..."
# Try to access Traefik API (insecure mode on port 8081)
echo "--- Checking Traefik API ---"
curl -s http://localhost:8081/api/http/routers | grep -i "api" | head -5 || echo "Cannot access Traefik API"
echo ""

echo "5. Testing HTTP (port 80) to verify Let's Encrypt challenge works..."
curl -I http://api.staging.stackyn.com/health 2>&1 | head -10
echo ""

echo "6. Checking DNS and connectivity..."
echo "--- DNS Resolution ---"
dig +short api.staging.stackyn.com
echo "--- Testing HTTP connection ---"
curl -s -o /dev/null -w "HTTP Status: %{http_code}\n" http://api.staging.stackyn.com/health
echo ""

echo "=========================================="
echo "Common Issues to Check:"
echo "=========================================="
echo "1. If acme.json is empty or missing:"
echo "   - Let's Encrypt certificate generation failed"
echo "   - Check logs above for specific errors"
echo ""
echo "2. If you see 'rate limit' errors:"
echo "   - Let's Encrypt has rate limits (5 certs/week per domain)"
echo "   - Wait or use staging Let's Encrypt server"
echo ""
echo "3. If you see 'connection refused' or 'timeout':"
echo "   - Port 80 might not be accessible from internet"
echo "   - Check firewall: sudo ufw status"
echo ""
echo "4. If DNS is wrong:"
echo "   - api.staging.stackyn.com must point to your server IP"
echo "   - Check with: dig api.staging.stackyn.com"
echo ""

