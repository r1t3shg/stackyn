#!/bin/bash

# Fix Certificate Domain Mismatch
# This script checks if the certificate in acme.json is for the wrong domain
# and deletes it if needed to allow regeneration for the correct domain

set -e

echo "=========================================="
echo "Fix Certificate Domain Mismatch"
echo "=========================================="
echo ""

cd /opt/stackyn || { echo "❌ Not in /opt/stackyn directory"; exit 1; }

# Load environment variables
source .env

if [ -z "$API_DOMAIN" ]; then
    echo "❌ API_DOMAIN not set in .env file"
    exit 1
fi

echo "Expected API domain: $API_DOMAIN"
echo ""

# Check if Traefik container is running
if ! docker-compose ps traefik | grep -q "Up"; then
    echo "⚠️  Traefik is not running. Starting Traefik..."
    docker-compose up -d traefik
    sleep 5
fi

# Check if acme.json exists
if ! docker-compose exec traefik test -f /letsencrypt/acme.json 2>/dev/null; then
    echo "✓ acme.json does not exist - will be created on first certificate request"
    exit 0
fi

# Get the domain from acme.json
CURRENT_DOMAIN=$(docker-compose exec traefik cat /letsencrypt/acme.json 2>/dev/null | grep -o '"main": "[^"]*"' | head -1 | cut -d'"' -f4 || echo "none")

echo "Current certificate domain: $CURRENT_DOMAIN"
echo ""

if [ "$CURRENT_DOMAIN" = "none" ] || [ -z "$CURRENT_DOMAIN" ]; then
    echo "⚠️  No certificate found in acme.json (or file is empty)"
    echo "   This is OK - certificate will be generated on first request"
    exit 0
fi

if [ "$CURRENT_DOMAIN" != "$API_DOMAIN" ]; then
    echo "❌ Certificate domain mismatch!"
    echo "   Certificate is for: $CURRENT_DOMAIN"
    echo "   But API_DOMAIN is: $API_DOMAIN"
    echo ""
    echo "   This will prevent SSL certificate generation for the correct domain."
    echo ""
    read -p "   Delete old certificate to allow regeneration? (y/n) " -n 1 -r
    echo ""
    
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo "   Backing up old certificate..."
        docker-compose exec traefik cp /letsencrypt/acme.json /letsencrypt/acme.json.backup.$(date +%Y%m%d_%H%M%S) 2>/dev/null || true
        
        echo "   Deleting old certificate..."
        docker-compose exec traefik rm -f /letsencrypt/acme.json
        echo "   ✓ Deleted old certificate"
        echo ""
        echo "   Restarting Traefik to trigger new certificate generation..."
        docker-compose restart traefik
        echo "   ✓ Traefik restarted"
        echo ""
        echo "   New certificate will be generated on first HTTPS request to $API_DOMAIN"
        echo "   This may take 1-2 minutes."
    else
        echo "   Certificate not deleted. SSL will continue to fail for $API_DOMAIN"
        exit 1
    fi
else
    echo "✓ Certificate domain matches API_DOMAIN"
    echo "   Certificate is for: $CURRENT_DOMAIN"
fi

echo ""
echo "=========================================="
echo "Next Steps:"
echo "=========================================="
echo "1. Make a request to trigger certificate generation:"
echo "   curl -k https://$API_DOMAIN/health"
echo ""
echo "2. Monitor certificate generation:"
echo "   docker-compose logs -f traefik | grep -i -E '(certificate obtained|acme|error)'"
echo ""
echo "3. Verify certificate after 1-2 minutes:"
echo "   echo | openssl s_client -connect $API_DOMAIN:443 -servername $API_DOMAIN 2>/dev/null | openssl x509 -noout -issuer -subject"
echo ""

