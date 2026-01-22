# Fix Certificate Domain Mismatch in Deployment

## Problem

The deployment script is preserving the old SSL certificate for `api.dev.stackyn.com` when deploying to `api.staging.stackyn.com`. This prevents Let's Encrypt from generating a new certificate for the correct domain.

## Solution

### Option 1: Run the Fix Script (Recommended)

Run this script on your server to check and fix the certificate domain mismatch:

```bash
cd /opt/stackyn
bash fix-certificate-domain-mismatch.sh
```

This script will:
1. Check if the certificate domain matches `API_DOMAIN` from `.env`
2. If it doesn't match, offer to delete the old certificate
3. Restart Traefik to trigger new certificate generation

### Option 2: Manual Fix

If you prefer to fix it manually:

```bash
cd /opt/stackyn

# 1. Check current certificate domain
docker-compose exec traefik cat /letsencrypt/acme.json | grep -o '"main": "[^"]*"' | head -1

# 2. If it shows api.dev.stackyn.com (or wrong domain), delete it:
docker-compose exec traefik rm -f /letsencrypt/acme.json

# 3. Restart Traefik
docker-compose restart traefik

# 4. Trigger certificate generation
curl -k https://api.staging.stackyn.com/health

# 5. Monitor logs
docker-compose logs -f traefik | grep -i acme
```

### Option 3: Update Deployment Script

If you have a CI/CD pipeline or deployment script that preserves certificates, update it to:

1. Check the certificate domain before preserving
2. Only preserve if the domain matches `API_DOMAIN`
3. Delete and regenerate if domain mismatch

Example check:
```bash
# In your deployment script, before preserving acme.json:
CURRENT_DOMAIN=$(docker-compose exec traefik cat /letsencrypt/acme.json 2>/dev/null | grep -o '"main": "[^"]*"' | head -1 | cut -d'"' -f4 || echo "none")
EXPECTED_DOMAIN="${API_DOMAIN:-api.staging.stackyn.com}"

if [ "$CURRENT_DOMAIN" != "$EXPECTED_DOMAIN" ] && [ "$CURRENT_DOMAIN" != "none" ]; then
    echo "⚠️  Certificate domain mismatch: $CURRENT_DOMAIN != $EXPECTED_DOMAIN"
    echo "   Deleting old certificate to allow regeneration..."
    docker-compose exec traefik rm -f /letsencrypt/acme.json
fi
```

## Verification

After fixing, verify the certificate:

```bash
# Check certificate issuer (should be Let's Encrypt, not TRAEFIK DEFAULT CERT)
echo | openssl s_client -connect api.staging.stackyn.com:443 -servername api.staging.stackyn.com 2>/dev/null | openssl x509 -noout -issuer -subject

# Should show:
# issuer=C = US, O = Let's Encrypt, CN = R3
# subject=CN = api.staging.stackyn.com
```

## Why This Happens

Docker volumes persist data between container restarts. The `traefik_data` volume contains the `acme.json` file with the old certificate. When you change domains (e.g., from `api.dev.stackyn.com` to `api.staging.stackyn.com`), the old certificate is still there, and Traefik uses it instead of generating a new one.

## Prevention

For future deployments, ensure your deployment script:
1. Checks certificate domain before preserving
2. Only preserves certificates that match the current `API_DOMAIN`
3. Deletes mismatched certificates automatically

