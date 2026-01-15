#!/bin/bash
# SSL Certificate Fix Script for Stackyn VPS
# This script fixes SSL certificate issues by:
# 1. Clearing old certificate attempts
# 2. Restarting Traefik with staging Let's Encrypt
# 3. Verifying certificate issuance

set -e

echo "🔧 SSL Certificate Fix Script"
echo "=============================="
echo ""

cd /opt/stackyn

# Step 1: Verify docker-compose.yml has staging configuration
echo "📋 Step 1: Verifying Traefik configuration..."
if grep -q "acme-staging-v02.api.letsencrypt.org" docker-compose.yml; then
    echo "✅ Staging Let's Encrypt configuration found"
else
    echo "❌ Staging configuration not found in docker-compose.yml"
    echo "   Please ensure the file has been updated with staging CA server"
    exit 1
fi

# Step 2: Stop Traefik
echo ""
echo "📋 Step 2: Stopping Traefik..."
docker compose stop traefik || true
sleep 2

# Step 3: Clear acme.json
echo ""
echo "📋 Step 3: Clearing old certificate attempts..."
docker compose exec traefik rm -f /letsencrypt/acme.json 2>/dev/null || true
docker volume inspect stackyn_traefik_data >/dev/null 2>&1 || docker volume create stackyn_traefik_data
docker run --rm -v stackyn_traefik_data:/letsencrypt alpine sh -c "rm -f /letsencrypt/acme.json && touch /letsencrypt/acme.json && chmod 600 /letsencrypt/acme.json" || true
echo "✅ Cleared acme.json"

# Step 4: Restart Traefik
echo ""
echo "📋 Step 4: Restarting Traefik with staging configuration..."
docker compose up -d traefik
echo "✅ Traefik restarted"

# Step 5: Wait for Traefik to initialize
echo ""
echo "📋 Step 5: Waiting for Traefik to initialize..."
sleep 10

# Step 6: Check Traefik logs for certificate requests
echo ""
echo "📋 Step 6: Checking Traefik logs for certificate requests..."
echo "   (This may take 30-60 seconds for certificates to be requested)"
sleep 30

docker compose logs traefik --tail 50 | grep -i "certificate\|acme\|letsencrypt" || echo "   No certificate logs yet (this is normal, may take more time)"

# Step 7: Verify services are running
echo ""
echo "📋 Step 7: Verifying services are running..."
docker compose ps

# Step 8: Test HTTPS access
echo ""
echo "📋 Step 8: Testing HTTPS access..."
echo "   Testing staging.stackyn.com..."
if curl -k -I https://staging.stackyn.com 2>&1 | head -5; then
    echo "✅ Site is accessible (may show certificate warning with staging certs)"
else
    echo "⚠️  Site may not be accessible yet. Check Traefik logs:"
    echo "   docker compose logs traefik --tail 100"
fi

# Step 9: Check certificate status
echo ""
echo "📋 Step 9: Checking certificate status in Traefik..."
docker compose exec traefik cat /letsencrypt/acme.json 2>/dev/null | grep -q "Certificates" && echo "✅ Certificates found in acme.json" || echo "⚠️  No certificates in acme.json yet (may take 1-2 minutes)"

echo ""
echo "✅ SSL Fix Script Complete!"
echo ""
echo "📝 Next Steps:"
echo "   1. Wait 1-2 minutes for certificates to be issued"
echo "   2. Check Traefik logs: docker compose logs traefik --tail 100"
echo "   3. Test the site: curl -k https://staging.stackyn.com"
echo "   4. Note: Staging certificates will show browser warnings"
echo "   5. After rate limit expires (2026-01-05), switch back to production:"
echo "      Edit docker-compose.yml line 70:"
echo "      Change: acme-staging-v02.api.letsencrypt.org"
echo "      To: acme-v02.api.letsencrypt.org"
echo "      Then restart: docker compose restart traefik"

