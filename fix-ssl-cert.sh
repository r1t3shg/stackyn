#!/bin/bash

# SSL Certificate Fix Script for Stackyn Staging
# Run this on your server: /opt/stackyn

set -e

echo "=========================================="
echo "SSL Certificate Fix Script"
echo "=========================================="
echo ""

# Check if we're in the right directory
if [ ! -f "docker-compose.yml" ]; then
    echo "❌ Error: docker-compose.yml not found. Please run this script from /opt/stackyn"
    exit 1
fi

# Check if .env file exists
if [ ! -f ".env" ]; then
    echo "❌ Error: .env file not found"
    exit 1
fi

echo "1. Verifying environment variables..."
source .env
if [ -z "$API_DOMAIN" ]; then
    echo "❌ API_DOMAIN not set in .env file"
    exit 1
fi
echo "✓ API_DOMAIN: $API_DOMAIN"
echo "✓ ACME_EMAIL: ${ACME_EMAIL:-not set}"
echo ""

echo "2. Checking DNS resolution..."
API_IP=$(dig +short $API_DOMAIN | head -1)
if [ -z "$API_IP" ]; then
    echo "❌ DNS not resolving for $API_DOMAIN"
    echo "   Please ensure DNS A record points to your server IP"
    exit 1
fi
echo "✓ $API_DOMAIN resolves to: $API_IP"
echo ""

echo "3. Checking if port 80 is accessible..."
if ! curl -s -o /dev/null -w "%{http_code}" http://$API_DOMAIN | grep -q "200\|301\|302\|404"; then
    echo "⚠️  Port 80 might not be accessible. Checking firewall..."
    if command -v ufw &> /dev/null; then
        echo "   Opening port 80 in firewall..."
        sudo ufw allow 80/tcp
        sudo ufw allow 443/tcp
    fi
fi
echo "✓ Port 80 check complete"
echo ""

echo "4. Checking Traefik container..."
if ! docker-compose ps traefik | grep -q "Up"; then
    echo "⚠️  Traefik is not running. Starting Traefik..."
    docker-compose up -d traefik
    echo "   Waiting 10 seconds for Traefik to start..."
    sleep 10
fi
echo "✓ Traefik is running"
echo ""

echo "5. Restarting API container to ensure labels are applied..."
docker-compose restart api
echo "✓ API container restarted"
echo ""

echo "6. Checking Traefik logs for certificate generation..."
echo "   (This may take a minute for Let's Encrypt to generate certificate)"
echo ""
docker-compose logs traefik --tail=100 | grep -i -E "(cert|certificate|acme|letsencrypt|error|obtained)" || echo "No certificate logs found yet"
echo ""

echo "7. Waiting 30 seconds for certificate generation..."
sleep 30

echo ""
echo "8. Checking certificate status..."
docker-compose logs traefik --tail=50 | grep -i -E "(certificate obtained|unable to obtain|error)" || echo "Check logs manually"
echo ""

echo "9. Testing HTTPS connection..."
if curl -k -s -o /dev/null -w "%{http_code}" https://$API_DOMAIN/health | grep -q "200\|404"; then
    echo "✓ HTTPS connection works (certificate may still be self-signed)"
else
    echo "⚠️  HTTPS connection failed"
fi
echo ""

echo "=========================================="
echo "Next Steps:"
echo "=========================================="
echo "1. Monitor Traefik logs: docker-compose logs -f traefik"
echo "2. Look for 'Certificate obtained' message"
echo "3. If you see errors, check:"
echo "   - DNS is pointing to your server IP"
echo "   - Port 80 is accessible from internet"
echo "   - No rate limiting from Let's Encrypt (5 certs/week per domain)"
echo ""
echo "4. Test the certificate:"
echo "   curl -v https://$API_DOMAIN/health"
echo ""
echo "5. If certificate is still not working after 2-3 minutes, check:"
echo "   docker-compose exec traefik cat /letsencrypt/acme.json"
echo ""

