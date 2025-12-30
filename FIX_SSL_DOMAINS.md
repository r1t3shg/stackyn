# Fix SSL Certificate Domain Error

## Problem
Traefik is trying to generate SSL certificates for `api.localhost`, which Let's Encrypt cannot issue.

## Solution

### Step 1: Check your `.env` file

Make sure your `.env` file has the correct domains (NOT `.localhost`):

```env
# Frontend Domain
FRONTEND_DOMAIN=staging.stackyn.com

# API Domain  
API_DOMAIN=api.staging.stackyn.com

# Let's Encrypt Email
ACME_EMAIL=your-email@example.com
```

**IMPORTANT:** Do NOT use `.localhost` domains - they won't work with Let's Encrypt!

### Step 2: Restart containers to apply new labels

```bash
cd /opt/stackyn

# Pull latest changes
git pull origin develop

# Restart all containers to apply new Traefik labels
docker-compose down
docker-compose up -d

# Check Traefik logs
docker-compose logs -f traefik
```

### Step 3: Verify DNS

Make sure your domains point to your VPS IP:

```bash
dig staging.stackyn.com
dig api.staging.stackyn.com
```

Both should return your VPS IP address.

### Step 4: Wait for certificate generation

Let's Encrypt certificates take 1-2 minutes to generate. Watch the logs:

```bash
docker-compose logs -f traefik | grep -i acme
```

You should see:
- `Certificate obtained` (success)
- `Unable to obtain ACME certificate` (error - check DNS)

### Step 5: Test HTTPS

After certificates are generated:
- https://staging.stackyn.com (should work)
- https://api.staging.stackyn.com (should work)

## Common Issues

1. **`.env` file has wrong domains**: Make sure `API_DOMAIN` and `FRONTEND_DOMAIN` are set correctly
2. **DNS not configured**: Domains must point to VPS IP before SSL works
3. **Port 80 blocked**: Let's Encrypt needs port 80 for verification
4. **Old containers running**: Restart containers to apply new labels

