#!/bin/bash

# Force Let's Encrypt Certificate Generation
# This script will diagnose and fix the certificate generation issue

set -e

echo "=========================================="
echo "Force Let's Encrypt Certificate Generation"
echo "=========================================="
echo ""

cd /opt/stackyn || { echo "❌ Not in /opt/stackyn directory"; exit 1; }

echo "1. Checking current certificate status..."
CERT_ISSUER=$(echo | openssl s_client -connect api.staging.stackyn.com:443 -servername api.staging.stackyn.com 2>/dev/null | openssl x509 -noout -issuer 2>/dev/null || echo "unknown")
echo "   Current certificate issuer: $CERT_ISSUER"
if echo "$CERT_ISSUER" | grep -q "TRAEFIK DEFAULT CERT"; then
    echo "   ⚠️  Using Traefik default (self-signed) certificate"
else
    echo "   ✓ Not using default certificate"
fi
echo ""

echo "2. Checking environment variables..."
source .env
echo "   API_DOMAIN: ${API_DOMAIN:-NOT SET}"
echo "   ACME_EMAIL: ${ACME_EMAIL:-NOT SET}"
if [ -z "$API_DOMAIN" ] || [ "$API_DOMAIN" != "api.staging.stackyn.com" ]; then
    echo "   ❌ API_DOMAIN is not set correctly!"
    exit 1
fi
echo ""

echo "3. Checking if acme.json exists..."
if docker-compose exec traefik test -f /letsencrypt/acme.json 2>/dev/null; then
    ACME_SIZE=$(docker-compose exec traefik stat -c%s /letsencrypt/acme.json 2>/dev/null || echo "0")
    echo "   acme.json exists ($ACME_SIZE bytes)"
    
    # Check if it has the correct domain
    if docker-compose exec traefik grep -q "api.staging.stackyn.com" /letsencrypt/acme.json 2>/dev/null; then
        echo "   ✓ Domain found in acme.json"
    else
        echo "   ⚠️  Domain NOT found in acme.json - deleting to force regeneration"
        docker-compose exec traefik rm -f /letsencrypt/acme.json
        echo "   ✓ Deleted acme.json"
    fi
else
    echo "   ⚠️  acme.json does not exist (will be created)"
fi
echo ""

echo "4. Verifying API container Traefik labels..."
echo "   Checking if API_DOMAIN is used in labels..."
docker-compose config | grep -A 30 "api:" | grep -i "traefik.http.routers.api.rule" || echo "   ⚠️  Router rule not found"
echo ""

echo "5. Checking Traefik logs for recent ACME activity..."
echo "   --- Recent ACME logs ---"
docker-compose logs traefik --tail=100 | grep -i -E "(acme|certificate|cert|letsencrypt)" | tail -20 || echo "   No ACME logs found"
echo ""

echo "6. Verifying DNS and port 80..."
API_IP=$(dig +short api.staging.stackyn.com | head -1)
echo "   DNS resolves to: $API_IP"
HTTP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" http://api.staging.stackyn.com/health || echo "000")
echo "   HTTP status: $HTTP_STATUS"
if [ "$HTTP_STATUS" = "000" ]; then
    echo "   ⚠️  Port 80 might not be accessible - checking firewall..."
    if command -v ufw &> /dev/null; then
        sudo ufw allow 80/tcp 2>/dev/null || true
        sudo ufw allow 443/tcp 2>/dev/null || true
        echo "   ✓ Firewall rules updated"
    fi
fi
echo ""

echo "7. Stopping services to ensure clean restart..."
docker-compose stop api traefik
echo "   Waiting 5 seconds..."
sleep 5
echo ""

echo "8. Starting Traefik first..."
docker-compose up -d traefik
echo "   Waiting 10 seconds for Traefik to start..."
sleep 10
echo ""

echo "9. Starting API container..."
docker-compose up -d api
echo "   Waiting 5 seconds..."
sleep 5
echo ""

echo "10. Checking Traefik configuration..."
echo "    Verifying API router is registered..."
sleep 3
docker-compose exec traefik wget -qO- http://localhost:8080/api/http/routers 2>/dev/null | grep -i "api" || echo "    ⚠️  API router not found in Traefik API"
echo ""

echo "11. Making a test HTTPS request to trigger certificate generation..."
echo "    (This will fail with certificate error, but should trigger ACME)"
curl -k -s https://api.staging.stackyn.com/health > /dev/null 2>&1 || true
echo "    ✓ Request sent"
echo ""

echo "12. Monitoring Traefik logs for certificate generation..."
echo "    (This may take 1-2 minutes)"
echo "    Press Ctrl+C to stop monitoring early"
echo ""

# Monitor for 2 minutes
timeout 120 docker-compose logs -f traefik 2>&1 | grep --line-buffered -i -E "(certificate obtained|unable to obtain|acme.*api.staging|error.*cert|retrieving.*certificate)" || true

echo ""
echo "13. Checking certificate status again..."
sleep 5

NEW_CERT_ISSUER=$(echo | openssl s_client -connect api.staging.stackyn.com:443 -servername api.staging.stackyn.com 2>/dev/null | openssl x509 -noout -issuer 2>/dev/null || echo "unknown")
NEW_CERT_SUBJECT=$(echo | openssl s_client -connect api.staging.stackyn.com:443 -servername api.staging.stackyn.com 2>/dev/null | openssl x509 -noout -subject 2>/dev/null || echo "unknown")

echo "   Certificate issuer: $NEW_CERT_ISSUER"
echo "   Certificate subject: $NEW_CERT_SUBJECT"

if echo "$NEW_CERT_ISSUER" | grep -q "Let's Encrypt"; then
    echo ""
    echo "✓✓✓ SUCCESS! Let's Encrypt certificate is now active! ✓✓✓"
elif echo "$NEW_CERT_SUBJECT" | grep -q "api.staging.stackyn.com" && ! echo "$NEW_CERT_ISSUER" | grep -q "TRAEFIK DEFAULT"; then
    echo ""
    echo "⚠️  Certificate is for correct domain but not from Let's Encrypt"
    echo "   This might be a temporary certificate"
elif echo "$NEW_CERT_ISSUER" | grep -q "TRAEFIK DEFAULT"; then
    echo ""
    echo "❌ Still using Traefik default certificate"
    echo ""
    echo "   Troubleshooting steps:"
    echo "   1. Check Traefik logs: docker-compose logs traefik | grep -i acme | tail -50"
    echo "   2. Verify API container labels: docker-compose config | grep -A 20 'api:' | grep traefik"
    echo "   3. Check if API_DOMAIN is being substituted: docker-compose config | grep api.staging"
    echo "   4. Verify DNS: dig api.staging.stackyn.com"
    echo "   5. Test HTTP challenge: curl -I http://api.staging.stackyn.com/.well-known/acme-challenge/test"
fi

echo ""
echo "=========================================="
echo "If certificate generation failed:"
echo "=========================================="
echo "1. Check detailed logs: docker-compose logs traefik | tail -100"
echo "2. Look for specific errors (rate limit, DNS, connection refused)"
echo "3. Verify Traefik can access port 80: docker-compose exec traefik wget -O- http://api.staging.stackyn.com"
echo "4. Check acme.json: docker-compose exec traefik cat /letsencrypt/acme.json | jq ."
echo ""

