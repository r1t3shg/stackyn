#!/bin/bash
# SSL Certificate Diagnostic Script for Stackyn VPS
# Run this on your VPS to diagnose SSL certificate issues

set -e

echo "🔍 SSL Certificate Diagnostic Script"
echo "===================================="
echo ""

cd /opt/stackyn || { echo "❌ Not in /opt/stackyn directory"; exit 1; }

# Step 1: Check if docker-compose.yml has staging configuration
echo "📋 Step 1: Checking Traefik configuration..."
if grep -q "acme-staging-v02.api.letsencrypt.org" docker-compose.yml; then
    echo "✅ Staging Let's Encrypt configuration found"
else
    echo "❌ Staging configuration NOT found!"
    echo "   Current CA server:"
    grep "caserver" docker-compose.yml || echo "   No CA server found!"
    echo ""
    echo "   Fix: Update docker-compose.yml line 70 to use staging"
fi

# Step 2: Check if services are running
echo ""
echo "📋 Step 2: Checking service status..."
docker compose ps

# Step 3: Check if Traefik is running
echo ""
echo "📋 Step 3: Checking Traefik container..."
if docker ps | grep -q "stackyn-traefik"; then
    echo "✅ Traefik container is running"
    TRAEFIK_STATUS=$(docker inspect stackyn-traefik --format='{{.State.Status}}')
    echo "   Status: $TRAEFIK_STATUS"
else
    echo "❌ Traefik container is NOT running!"
    echo "   Start it with: docker compose up -d traefik"
fi

# Step 4: Check Traefik logs for errors
echo ""
echo "📋 Step 4: Checking Traefik logs (last 50 lines)..."
echo "   Looking for certificate-related messages..."
docker compose logs traefik --tail 50 | grep -i -E "certificate|acme|letsencrypt|error|failed" || echo "   No certificate-related logs found"

# Step 5: Check if acme.json exists and has content
echo ""
echo "📋 Step 5: Checking certificate storage..."
if docker compose exec traefik test -f /letsencrypt/acme.json 2>/dev/null; then
    echo "✅ acme.json exists"
    ACME_SIZE=$(docker compose exec traefik stat -c%s /letsencrypt/acme.json 2>/dev/null || echo "0")
    if [ "$ACME_SIZE" -gt 10 ]; then
        echo "   Size: $ACME_SIZE bytes (has content)"
        # Check if it has certificates
        if docker compose exec traefik cat /letsencrypt/acme.json 2>/dev/null | grep -q "Certificates"; then
            echo "   ✅ Contains certificate data"
        else
            echo "   ⚠️  File exists but no certificates found"
        fi
    else
        echo "   ⚠️  File is empty or very small ($ACME_SIZE bytes)"
    fi
else
    echo "❌ acme.json does NOT exist!"
    echo "   Initialize it with:"
    echo "   docker run --rm -v stackyn_traefik_data:/letsencrypt alpine sh -c 'touch /letsencrypt/acme.json && chmod 600 /letsencrypt/acme.json && echo \"{}\" > /letsencrypt/acme.json'"
fi

# Step 6: Check DNS configuration
echo ""
echo "📋 Step 6: Checking DNS configuration..."
DOMAIN="staging.stackyn.com"
echo "   Checking DNS for $DOMAIN..."
DNS_IP=$(dig +short $DOMAIN | tail -1)
if [ -z "$DNS_IP" ]; then
    echo "   ❌ DNS lookup failed - domain may not be configured"
else
    echo "   DNS resolves to: $DNS_IP"
    VPS_IP=$(curl -s ifconfig.me || curl -s ipinfo.io/ip || echo "unknown")
    echo "   VPS IP: $VPS_IP"
    if [ "$DNS_IP" = "$VPS_IP" ]; then
        echo "   ✅ DNS points to this VPS"
    else
        echo "   ⚠️  DNS does NOT point to this VPS!"
        echo "      Update DNS A record for $DOMAIN to point to $VPS_IP"
    fi
fi

# Step 7: Check port accessibility
echo ""
echo "📋 Step 7: Checking port accessibility..."
if netstat -tlnp 2>/dev/null | grep -q ":80 "; then
    echo "   ✅ Port 80 is listening"
else
    echo "   ❌ Port 80 is NOT listening!"
fi

if netstat -tlnp 2>/dev/null | grep -q ":443 "; then
    echo "   ✅ Port 443 is listening"
else
    echo "   ❌ Port 443 is NOT listening!"
fi

# Step 8: Test HTTP access
echo ""
echo "📋 Step 8: Testing HTTP access..."
if curl -s -o /dev/null -w "%{http_code}" http://localhost/ | grep -q "200\|301\|302"; then
    echo "   ✅ HTTP (port 80) is accessible"
else
    HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" http://localhost/ || echo "000")
    echo "   ⚠️  HTTP returned code: $HTTP_CODE"
fi

# Step 9: Test HTTPS access
echo ""
echo "📋 Step 9: Testing HTTPS access..."
if curl -k -s -o /dev/null -w "%{http_code}" https://localhost/ | grep -q "200\|301\|302"; then
    echo "   ✅ HTTPS (port 443) is accessible (with -k flag)"
else
    HTTPS_CODE=$(curl -k -s -o /dev/null -w "%{http_code}" https://localhost/ || echo "000")
    echo "   ⚠️  HTTPS returned code: $HTTPS_CODE"
fi

# Step 10: Check environment variables
echo ""
echo "📋 Step 10: Checking critical environment variables..."
if [ -f .env ]; then
    echo "   ✅ .env file exists"
    if grep -q "^ACME_EMAIL=" .env; then
        ACME_EMAIL=$(grep "^ACME_EMAIL=" .env | cut -d'=' -f2)
        echo "   ACME_EMAIL: $ACME_EMAIL"
    else
        echo "   ⚠️  ACME_EMAIL not set in .env"
    fi
    if grep -q "^FRONTEND_DOMAIN=" .env; then
        FRONTEND_DOMAIN=$(grep "^FRONTEND_DOMAIN=" .env | cut -d'=' -f2)
        echo "   FRONTEND_DOMAIN: $FRONTEND_DOMAIN"
    else
        echo "   ⚠️  FRONTEND_DOMAIN not set in .env"
    fi
else
    echo "   ❌ .env file does NOT exist!"
fi

# Step 11: Check frontend service
echo ""
echo "📋 Step 11: Checking frontend service..."
if docker ps | grep -q "stackyn-frontend"; then
    echo "   ✅ Frontend container is running"
    FRONTEND_LOGS=$(docker compose logs frontend --tail 5 2>/dev/null | tail -3)
    if [ ! -z "$FRONTEND_LOGS" ]; then
        echo "   Recent logs:"
        echo "$FRONTEND_LOGS" | sed 's/^/      /'
    fi
else
    echo "   ❌ Frontend container is NOT running!"
fi

# Step 12: Check Traefik API
echo ""
echo "📋 Step 12: Checking Traefik API..."
if curl -s http://localhost:8081/api/overview >/dev/null 2>&1; then
    echo "   ✅ Traefik API is accessible"
    # Try to get certificate info
    CERT_INFO=$(curl -s http://localhost:8081/api/http/routers 2>/dev/null | grep -o "staging.stackyn.com" | head -1 || echo "")
    if [ ! -z "$CERT_INFO" ]; then
        echo "   ✅ Router for staging.stackyn.com found"
    else
        echo "   ⚠️  Router for staging.stackyn.com not found in Traefik API"
    fi
else
    echo "   ⚠️  Traefik API not accessible (may be normal if dashboard is disabled)"
fi

# Summary
echo ""
echo "===================================="
echo "📊 Diagnostic Summary"
echo "===================================="
echo ""
echo "Next steps:"
echo "1. If Traefik is not running: docker compose up -d traefik"
echo "2. If acme.json is missing: Run the initialization command shown above"
echo "3. If DNS is wrong: Update your DNS A record"
echo "4. Check full Traefik logs: docker compose logs traefik --tail 100"
echo "5. Restart Traefik: docker compose restart traefik"
echo ""
echo "To see real-time Traefik logs:"
echo "   docker compose logs -f traefik"
echo ""

