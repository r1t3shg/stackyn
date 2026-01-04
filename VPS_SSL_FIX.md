# SSL Certificate Fix - Manual Steps for VPS

## Problem
`https://staging.stackyn.com` is not accessible due to Let's Encrypt rate limiting.

## Solution
We've switched to Let's Encrypt **staging environment** temporarily to bypass the rate limit. Staging certificates will work but browsers will show warnings.

## Quick Fix (Run on VPS)

SSH into your VPS and run these commands:

```bash
cd /opt/stackyn

# 1. Verify the docker-compose.yml has staging configuration
grep -q "acme-staging-v02.api.letsencrypt.org" docker-compose.yml && echo "✅ Staging config found" || echo "❌ Need to update docker-compose.yml"

# 2. If staging config is missing, update it manually:
# Edit docker-compose.yml line 70 and change:
# FROM: https://acme-v02.api.letsencrypt.org/directory
# TO:   https://acme-staging-v02.api.letsencrypt.org/directory

# 3. Stop Traefik
docker compose stop traefik

# 4. Clear old certificate attempts
docker volume inspect stackyn_traefik_data >/dev/null 2>&1 || docker volume create stackyn_traefik_data
docker run --rm -v stackyn_traefik_data:/letsencrypt alpine sh -c "rm -f /letsencrypt/acme.json && touch /letsencrypt/acme.json && chmod 600 /letsencrypt/acme.json"

# 5. Restart Traefik
docker compose up -d traefik

# 6. Wait for certificates (30-60 seconds)
echo "Waiting for certificates to be issued..."
sleep 60

# 7. Check Traefik logs
docker compose logs traefik --tail 50 | grep -i "certificate\|acme"

# 8. Test the site
curl -k -I https://staging.stackyn.com

# 9. Check all services are running
docker compose ps
```

## Alternative: Use the Fix Script

If you've pushed the `fix-ssl.sh` script to the repo:

```bash
cd /opt/stackyn
git pull origin develop
chmod +x fix-ssl.sh
./fix-ssl.sh
```

## Verify Fix

After running the commands:

1. **Check Traefik logs:**
   ```bash
   docker compose logs traefik --tail 100 | grep -i "certificate\|acme\|error"
   ```

2. **Test HTTPS:**
   ```bash
   curl -k https://staging.stackyn.com
   ```
   Should return HTML (may show certificate warning, but site works)

3. **Check certificate in browser:**
   - Visit `https://staging.stackyn.com`
   - Click "Advanced" → "Proceed anyway" (staging certs show warnings)
   - Site should load

## Important Notes

1. **Staging Certificates:** Browsers will show "Your connection is not private" warnings because staging certificates aren't trusted. This is expected and the site will work.

2. **Switch Back to Production:** After the rate limit expires (around 2026-01-05 04:53:04 UTC), switch back:
   - Edit `docker-compose.yml` line 70
   - Change back to: `https://acme-v02.api.letsencrypt.org/directory`
   - Restart: `docker compose restart traefik`

3. **If Still Not Working:**
   - Check DNS: `dig staging.stackyn.com` (should point to your VPS IP)
   - Check firewall: Ensure ports 80 and 443 are open
   - Check Traefik: `docker compose logs traefik --tail 200`
   - Check frontend service: `docker compose logs frontend --tail 50`

## Troubleshooting

### Site returns "Connection refused"
- Check if Traefik is running: `docker compose ps traefik`
- Check if ports 80/443 are open: `netstat -tlnp | grep -E ':(80|443)'`

### Certificate not being issued
- Check Traefik logs for ACME errors
- Verify DNS is pointing to your VPS: `dig staging.stackyn.com`
- Ensure port 80 is accessible (Let's Encrypt needs HTTP-01 challenge)

### Services not starting
- Check all services: `docker compose ps`
- Check logs: `docker compose logs --tail 100`
- Verify .env file has all required variables

