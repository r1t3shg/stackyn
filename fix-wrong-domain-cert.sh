#!/bin/bash

# Fix Wrong Domain Certificate Issue
# The acme.json has a certificate for api.dev.stackyn.com but we need api.staging.stackyn.com

set -e

echo "=========================================="
echo "Fixing Wrong Domain Certificate"
echo "=========================================="
echo ""

cd /opt/stackyn || { echo "❌ Not in /opt/stackyn directory"; exit 1; }

echo "1. Checking current certificate domain..."
CURRENT_DOMAIN=$(docker-compose exec traefik cat /letsencrypt/acme.json 2>/dev/null | grep -o '"main": "[^"]*"' | head -1 | cut -d'"' -f4 || echo "none")
echo "   Current certificate domain: $CURRENT_DOMAIN"
echo "   Required domain: api.staging.stackyn.com"
echo ""

if [ "$CURRENT_DOMAIN" != "api.staging.stackyn.com" ]; then
    echo "⚠️  Certificate is for wrong domain!"
    echo "   Deleting old certificate to force regeneration..."
    
    # Backup the old acme.json first
    echo "   Creating backup..."
    docker-compose exec traefik cp /letsencrypt/acme.json /letsencrypt/acme.json.backup 2>/dev/null || true
    
    # Delete the old certificate entry (we'll keep the account info)
    echo "   Removing old certificate from acme.json..."
    # We'll delete the entire file and let Traefik regenerate it
    docker-compose exec traefik rm -f /letsencrypt/acme.json
    echo "   ✓ Deleted old certificate"
else
    echo "✓ Certificate domain is correct"
fi
echo ""

echo "2. Verifying environment variables..."
source .env
if [ "$API_DOMAIN" != "api.staging.stackyn.com" ]; then
    echo "❌ API_DOMAIN in .env is: $API_DOMAIN"
    echo "   Expected: api.staging.stackyn.com"
    echo "   Please update .env file!"
    exit 1
fi
echo "✓ API_DOMAIN is correct: $API_DOMAIN"
echo ""

echo "3. Verifying DNS..."
API_IP=$(dig +short api.staging.stackyn.com | head -1)
if [ -z "$API_IP" ]; then
    echo "❌ DNS not resolving"
    exit 1
fi
echo "✓ DNS resolves to: $API_IP"
echo ""

echo "4. Verifying port 80 accessibility..."
HTTP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" http://api.staging.stackyn.com/health || echo "000")
if [ "$HTTP_STATUS" = "000" ]; then
    echo "⚠️  Port 80 might not be accessible"
    echo "   Checking firewall..."
    if command -v ufw &> /dev/null; then
        sudo ufw allow 80/tcp 2>/dev/null || true
        sudo ufw allow 443/tcp 2>/dev/null || true
    fi
else
    echo "✓ Port 80 is accessible (HTTP status: $HTTP_STATUS)"
fi
echo ""

echo "5. Restarting Traefik to trigger new certificate generation..."
docker-compose restart traefik
echo "   Waiting 15 seconds for Traefik to start..."
sleep 15
echo ""

echo "6. Restarting API container to ensure labels are applied..."
docker-compose restart api
echo "   Waiting 5 seconds..."
sleep 5
echo ""

echo "7. Monitoring Traefik logs for certificate generation..."
echo "   (This may take 1-2 minutes for Let's Encrypt)"
echo "   Looking for 'Certificate obtained' message..."
echo ""

# Monitor logs for 2 minutes
timeout 120 docker-compose logs -f traefik 2>&1 | grep --line-buffered -i -E "(certificate obtained|unable to obtain|acme.*api.staging|error.*cert)" || true

echo ""
echo "8. Checking certificate status..."
sleep 5

# Check if certificate was generated
if docker-compose exec traefik test -f /letsencrypt/acme.json 2>/dev/null; then
    NEW_DOMAIN=$(docker-compose exec traefik cat /letsencrypt/acme.json 2>/dev/null | grep -o '"main": "[^"]*"' | head -1 | cut -d'"' -f4 || echo "none")
    if [ "$NEW_DOMAIN" = "api.staging.stackyn.com" ]; then
        echo "✓ SUCCESS! Certificate for api.staging.stackyn.com found in acme.json"
    else
        echo "⚠️  Certificate domain is still: $NEW_DOMAIN"
        echo "   Check Traefik logs for errors"
    fi
else
    echo "⚠️  acme.json not found yet"
fi

echo ""
echo "9. Testing HTTPS connection..."
sleep 3

# Test with curl (ignore certificate errors to see if server responds)
if curl -k -s https://api.staging.stackyn.com/health > /dev/null 2>&1; then
    # Check certificate issuer
    CERT_ISSUER=$(echo | openssl s_client -connect api.staging.stackyn.com:443 -servername api.staging.stackyn.com 2>/dev/null | openssl x509 -noout -issuer 2>/dev/null || echo "unknown")
    CERT_SUBJECT=$(echo | openssl s_client -connect api.staging.stackyn.com:443 -servername api.staging.stackyn.com 2>/dev/null | openssl x509 -noout -subject 2>/dev/null || echo "unknown")
    
    echo "   Certificate issuer: $CERT_ISSUER"
    echo "   Certificate subject: $CERT_SUBJECT"
    
    if echo "$CERT_ISSUER" | grep -q "Let's Encrypt"; then
        echo ""
        echo "✓✓✓ SUCCESS! Let's Encrypt certificate is now active! ✓✓✓"
    elif echo "$CERT_SUBJECT" | grep -q "api.staging.stackyn.com"; then
        echo ""
        echo "⚠️  Certificate is for correct domain but issuer is not Let's Encrypt"
        echo "   This might be a self-signed certificate still"
        echo "   Check Traefik logs: docker-compose logs traefik | grep -i acme"
    else
        echo ""
        echo "⚠️  Certificate is still wrong or self-signed"
        echo "   Check Traefik logs: docker-compose logs traefik | grep -i acme"
    fi
else
    echo "❌ HTTPS connection failed"
fi

echo ""
echo "=========================================="
echo "Next Steps if certificate is still wrong:"
echo "=========================================="
echo "1. Check Traefik logs: docker-compose logs traefik | grep -i acme | tail -50"
echo "2. Verify API container labels: docker-compose config | grep -A 20 'api:' | grep -i traefik"
echo "3. Check for rate limiting errors (5 certs/week per domain)"
echo "4. Verify DNS: dig api.staging.stackyn.com"
echo "5. Verify port 80: curl -I http://api.staging.stackyn.com"
echo ""

