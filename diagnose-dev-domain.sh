#!/bin/bash

echo "🔍 Diagnosing dev.stackyn.com issues..."
echo "=========================================="
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check DNS
echo "1. Checking DNS resolution..."
echo "----------------------------"
DOMAINS=("dev.stackyn.com" "api.dev.stackyn.com" "console.dev.stackyn.com")
VPS_IP=$(curl -s ifconfig.me 2>/dev/null || echo "UNKNOWN")

for domain in "${DOMAINS[@]}"; do
    DNS_IP=$(dig +short $domain | tail -1)
    if [ -z "$DNS_IP" ]; then
        echo -e "   ${RED}❌ $domain: NOT RESOLVING${NC}"
    elif [ "$DNS_IP" = "$VPS_IP" ]; then
        echo -e "   ${GREEN}✅ $domain: $DNS_IP (correct)${NC}"
    else
        echo -e "   ${YELLOW}⚠️  $domain: $DNS_IP (expected: $VPS_IP)${NC}"
    fi
done
echo ""

# Check Docker services
echo "2. Checking Docker services..."
echo "-------------------------------"
SERVICES=("stackyn-traefik" "stackyn-api" "stackyn-frontend")
for service in "${SERVICES[@]}"; do
    if docker ps --format "{{.Names}}" | grep -q "^${service}$"; then
        STATUS=$(docker inspect --format='{{.State.Status}}' $service 2>/dev/null)
        HEALTH=$(docker inspect --format='{{.State.Health.Status}}' $service 2>/dev/null || echo "no-healthcheck")
        if [ "$STATUS" = "running" ]; then
            if [ "$HEALTH" = "healthy" ] || [ "$HEALTH" = "no-healthcheck" ]; then
                echo -e "   ${GREEN}✅ $service: running${NC}"
            else
                echo -e "   ${YELLOW}⚠️  $service: running but $HEALTH${NC}"
            fi
        else
            echo -e "   ${RED}❌ $service: $STATUS${NC}"
        fi
    else
        echo -e "   ${RED}❌ $service: NOT RUNNING${NC}"
    fi
done
echo ""

# Check environment variables
echo "3. Checking environment variables..."
echo "------------------------------------"
if [ -f .env ]; then
    echo "   .env file exists"
    grep -E "^(FRONTEND_DOMAIN|API_DOMAIN|CONSOLE_DOMAIN|APP_BASE_DOMAIN|FRONTEND_API_URL)=" .env 2>/dev/null | while read line; do
        if echo "$line" | grep -q "dev.stackyn.com"; then
            echo -e "   ${GREEN}✅ $line${NC}"
        else
            echo -e "   ${YELLOW}⚠️  $line (should contain dev.stackyn.com)${NC}"
        fi
    done
else
    echo -e "   ${YELLOW}⚠️  .env file not found${NC}"
fi
echo ""

# Check Traefik routers
echo "4. Checking Traefik routers..."
echo "-------------------------------"
if docker ps | grep -q stackyn-traefik; then
    ROUTERS=$(curl -s http://localhost:8081/api/http/routers 2>/dev/null | grep -o '"name":"[^"]*"' | cut -d'"' -f4)
    if [ -z "$ROUTERS" ]; then
        echo -e "   ${RED}❌ Cannot access Traefik API${NC}"
    else
        echo "   Found routers:"
        echo "$ROUTERS" | while read router; do
            echo "     - $router"
        done
    fi
else
    echo -e "   ${RED}❌ Traefik not running${NC}"
fi
echo ""

# Check SSL certificates
echo "5. Checking SSL certificates..."
echo "--------------------------------"
if docker exec stackyn-traefik ls /letsencrypt/acme.json >/dev/null 2>&1; then
    CERT_SIZE=$(docker exec stackyn-traefik stat -c%s /letsencrypt/acme.json 2>/dev/null || echo "0")
    if [ "$CERT_SIZE" -gt 100 ]; then
        echo -e "   ${GREEN}✅ acme.json exists ($CERT_SIZE bytes)${NC}"
        
        # Check for dev.stackyn.com in certificates
        if docker exec stackyn-traefik cat /letsencrypt/acme.json 2>/dev/null | grep -q "dev.stackyn.com"; then
            echo -e "   ${GREEN}✅ dev.stackyn.com found in certificates${NC}"
        else
            echo -e "   ${YELLOW}⚠️  dev.stackyn.com NOT found in certificates${NC}"
        fi
    else
        echo -e "   ${YELLOW}⚠️  acme.json too small ($CERT_SIZE bytes) - may be empty${NC}"
    fi
else
    echo -e "   ${RED}❌ acme.json not found${NC}"
fi
echo ""

# Check Traefik logs for errors
echo "6. Recent Traefik errors..."
echo "----------------------------"
docker logs stackyn-traefik --tail 20 2>&1 | grep -i "error\|failed\|unable" | tail -5 | while read line; do
    echo -e "   ${RED}⚠️  $line${NC}"
done
if [ $? -ne 0 ]; then
    echo -e "   ${GREEN}✅ No recent errors found${NC}"
fi
echo ""

# Test HTTP connectivity
echo "7. Testing HTTP connectivity..."
echo "--------------------------------"
for domain in "${DOMAINS[@]}"; do
    HTTP_CODE=$(curl -k -s -o /dev/null -w "%{http_code}" --max-time 5 "http://$domain" 2>/dev/null || echo "000")
    HTTPS_CODE=$(curl -k -s -o /dev/null -w "%{http_code}" --max-time 5 "https://$domain" 2>/dev/null || echo "000")
    
    if [ "$HTTP_CODE" = "000" ] && [ "$HTTPS_CODE" = "000" ]; then
        echo -e "   ${RED}❌ $domain: Cannot connect${NC}"
    elif [ "$HTTP_CODE" = "301" ] || [ "$HTTP_CODE" = "302" ]; then
        echo -e "   ${GREEN}✅ $domain: HTTP redirects to HTTPS ($HTTP_CODE)${NC}"
    elif [ "$HTTPS_CODE" = "200" ]; then
        echo -e "   ${GREEN}✅ $domain: HTTPS working ($HTTPS_CODE)${NC}"
    else
        echo -e "   ${YELLOW}⚠️  $domain: HTTP=$HTTP_CODE HTTPS=$HTTPS_CODE${NC}"
    fi
done
echo ""

# Summary and recommendations
echo "📋 Summary and Recommendations"
echo "==============================="
echo ""
echo "If DNS is not resolving:"
echo "  1. Update DNS A records in your DNS provider"
echo "  2. Wait 5-15 minutes for DNS propagation"
echo "  3. Verify with: dig dev.stackyn.com"
echo ""
echo "If services are not running:"
echo "  1. Check logs: docker compose logs"
echo "  2. Restart: docker compose up -d"
echo ""
echo "If SSL certificates are missing:"
echo "  1. Check Traefik logs: docker compose logs traefik | tail -50"
echo "  2. Wait 2-3 minutes for Let's Encrypt to issue certificates"
echo "  3. Verify DNS is pointing to VPS before requesting certificates"
echo ""
echo "If environment variables are wrong:"
echo "  1. Update .env file with dev.stackyn.com domains"
echo "  2. Rebuild frontend: docker compose build frontend"
echo "  3. Restart: docker compose up -d"
echo ""

