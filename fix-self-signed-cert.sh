#!/bin/bash

# Fix Self-Signed Certificate Issue
# This script will help fix the self-signed certificate problem

set -e

echo "=========================================="
echo "Fixing Self-Signed Certificate Issue"
echo "=========================================="
echo ""

cd /opt/stackyn || { echo "❌ Not in /opt/stackyn directory"; exit 1; }

echo "1. Checking current certificate status..."
if curl -k -s https://api.staging.stackyn.com/health > /dev/null 2>&1; then
    echo "✓ Server is responding (with self-signed cert)"
else
    echo "❌ Server is not responding"
    exit 1
fi
echo ""

echo "2. Checking Traefik logs for ACME errors..."
echo "--- Looking for ACME errors ---"
docker-compose logs traefik --tail=500 | grep -i -E "(acme|letsencrypt|certificate|error|failed|rate limit)" | tail -20
echo ""

echo "3. Checking acme.json file..."
if docker-compose exec traefik test -f /letsencrypt/acme.json 2>/dev/null; then
    ACME_SIZE=$(docker-compose exec traefik stat -c%s /letsencrypt/acme.json 2>/dev/null || echo "0")
    echo "   acme.json size: $ACME_SIZE bytes"
    
    if [ "$ACME_SIZE" -lt 100 ]; then
        echo "⚠️  acme.json is too small or empty"
        echo "   This might be causing the self-signed certificate issue"
        read -p "   Do you want to delete acme.json and retry? (y/n) " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            echo "   Deleting acme.json..."
            docker-compose exec traefik rm -f /letsencrypt/acme.json
            echo "   ✓ Deleted"
        fi
    else
        echo "   Checking if domain is in acme.json..."
        if docker-compose exec traefik grep -q "api.staging.stackyn.com" /letsencrypt/acme.json 2>/dev/null; then
            echo "   ✓ Domain found in acme.json"
            echo "   But certificate is still self-signed - might be corrupted"
            read -p "   Do you want to delete acme.json and retry? (y/n) " -n 1 -r
            echo
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                echo "   Deleting acme.json..."
                docker-compose exec traefik rm -f /letsencrypt/acme.json
                echo "   ✓ Deleted"
            fi
        else
            echo "   ⚠️  Domain NOT found in acme.json"
            echo "   Deleting acme.json to retry..."
            docker-compose exec traefik rm -f /letsencrypt/acme.json
            echo "   ✓ Deleted"
        fi
    fi
else
    echo "   ⚠️  acme.json does not exist"
fi
echo ""

echo "4. Verifying DNS and port 80..."
API_IP=$(dig +short api.staging.stackyn.com | head -1)
if [ -z "$API_IP" ]; then
    echo "❌ DNS not resolving for api.staging.stackyn.com"
    echo "   Please fix DNS first!"
    exit 1
fi
echo "   ✓ DNS resolves to: $API_IP"

# Test HTTP connection
if curl -s -o /dev/null -w "%{http_code}" http://api.staging.stackyn.com/health | grep -q "200\|404"; then
    echo "   ✓ Port 80 is accessible"
else
    echo "   ⚠️  Port 80 might not be accessible"
    echo "   Checking firewall..."
    if command -v ufw &> /dev/null; then
        sudo ufw allow 80/tcp
        sudo ufw allow 443/tcp
        echo "   ✓ Firewall rules updated"
    fi
fi
echo ""

echo "5. Restarting Traefik to trigger certificate generation..."
docker-compose restart traefik
echo "   Waiting 10 seconds for Traefik to start..."
sleep 10
echo ""

echo "6. Restarting API container to ensure labels are applied..."
docker-compose restart api
echo "   Waiting 5 seconds..."
sleep 5
echo ""

echo "7. Monitoring Traefik logs for certificate generation..."
echo "   (This may take 1-2 minutes for Let's Encrypt)"
echo "   Press Ctrl+C to stop monitoring"
echo ""
timeout 120 docker-compose logs -f traefik 2>&1 | grep -i -E "(certificate obtained|unable to obtain|acme|error)" || true
echo ""

echo "8. Testing certificate..."
sleep 5
if curl -s https://api.staging.stackyn.com/health > /dev/null 2>&1; then
    CERT_ISSUER=$(echo | openssl s_client -connect api.staging.stackyn.com:443 -servername api.staging.stackyn.com 2>/dev/null | openssl x509 -noout -issuer 2>/dev/null || echo "unknown")
    if echo "$CERT_ISSUER" | grep -q "Let's Encrypt"; then
        echo "✓ SUCCESS! Let's Encrypt certificate is now active!"
        echo "   Certificate issuer: $CERT_ISSUER"
    else
        echo "⚠️  Certificate is still self-signed"
        echo "   Issuer: $CERT_ISSUER"
        echo ""
        echo "   Check Traefik logs for errors:"
        echo "   docker-compose logs traefik | grep -i acme"
    fi
else
    echo "❌ Server is not responding"
fi
echo ""

echo "=========================================="
echo "If certificate is still self-signed:"
echo "=========================================="
echo "1. Check Traefik logs: docker-compose logs traefik | grep -i acme"
echo "2. Verify DNS: dig api.staging.stackyn.com"
echo "3. Verify port 80: curl -I http://api.staging.stackyn.com"
echo "4. Check Let's Encrypt rate limits (5 certs/week per domain)"
echo "5. Try using Let's Encrypt staging server first to test"
echo ""

