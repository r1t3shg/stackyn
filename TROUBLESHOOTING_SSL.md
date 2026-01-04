# SSL Certificate Troubleshooting Guide

## Quick Diagnostic Commands

Run these commands on your VPS to diagnose the issue:

### 1. Check if changes were deployed
```bash
cd /opt/stackyn
git pull origin develop
grep "acme-staging-v02" docker-compose.yml
```

### 2. Check Traefik status
```bash
docker compose ps traefik
docker compose logs traefik --tail 100
```

### 3. Check if certificates are being requested
```bash
docker compose logs traefik --tail 200 | grep -i "acme\|certificate\|letsencrypt"
```

### 4. Check certificate storage
```bash
docker compose exec traefik ls -la /letsencrypt/
docker compose exec traefik cat /letsencrypt/acme.json | head -20
```

### 5. Test site accessibility
```bash
# Test HTTP
curl -I http://staging.stackyn.com

# Test HTTPS (ignore certificate errors)
curl -k -I https://staging.stackyn.com

# Check DNS
dig staging.stackyn.com
```

### 6. Check if services are running
```bash
docker compose ps
docker compose logs frontend --tail 50
```

## Common Issues and Fixes

### Issue 1: Changes not deployed to VPS
**Symptoms:** `docker-compose.yml` still has production Let's Encrypt URL

**Fix:**
```bash
cd /opt/stackyn
git pull origin develop
docker compose restart traefik
```

### Issue 2: Traefik not restarted after config change
**Symptoms:** Old configuration still active

**Fix:**
```bash
docker compose stop traefik
docker compose up -d traefik
# Wait 60 seconds
docker compose logs traefik --tail 50
```

### Issue 3: acme.json missing or corrupted
**Symptoms:** No certificates, Traefik errors about acme.json

**Fix:**
```bash
# Stop Traefik
docker compose stop traefik

# Initialize acme.json
docker volume inspect stackyn_traefik_data >/dev/null 2>&1 || docker volume create stackyn_traefik_data
docker run --rm -v stackyn_traefik_data:/letsencrypt alpine sh -c "rm -f /letsencrypt/acme.json && touch /letsencrypt/acme.json && chmod 600 /letsencrypt/acme.json && echo '{}' > /letsencrypt/acme.json"

# Restart Traefik
docker compose up -d traefik
```

### Issue 4: DNS not pointing to VPS
**Symptoms:** Domain doesn't resolve or points to wrong IP

**Fix:**
1. Get your VPS IP: `curl ifconfig.me`
2. Update DNS A record for `staging.stackyn.com` to point to your VPS IP
3. Wait for DNS propagation (5-60 minutes)
4. Verify: `dig staging.stackyn.com`

### Issue 5: Ports 80/443 blocked by firewall
**Symptoms:** Can't access site, ports not listening

**Fix:**
```bash
# Check if ports are listening
netstat -tlnp | grep -E ':(80|443)'

# If using UFW
ufw allow 80/tcp
ufw allow 443/tcp

# If using iptables
iptables -A INPUT -p tcp --dport 80 -j ACCEPT
iptables -A INPUT -p tcp --dport 443 -j ACCEPT
```

### Issue 6: Frontend service not running
**Symptoms:** Traefik works but frontend not accessible

**Fix:**
```bash
docker compose ps frontend
docker compose logs frontend --tail 100
docker compose up -d frontend
```

### Issue 7: Traefik can't reach Let's Encrypt
**Symptoms:** ACME errors in logs, connection timeouts

**Fix:**
```bash
# Test connectivity
docker compose exec traefik wget -O- https://acme-staging-v02.api.letsencrypt.org/directory

# Check firewall rules
# Ensure outbound HTTPS (443) is allowed
```

## Complete Reset Procedure

If nothing works, try a complete reset:

```bash
cd /opt/stackyn

# 1. Stop all services
docker compose down

# 2. Clear certificate storage (WARNING: This deletes existing certificates)
docker volume rm stackyn_traefik_data || true
docker volume create stackyn_traefik_data

# 3. Initialize acme.json
docker run --rm -v stackyn_traefik_data:/letsencrypt alpine sh -c "touch /letsencrypt/acme.json && chmod 600 /letsencrypt/acme.json && echo '{}' > /letsencrypt/acme.json"

# 4. Verify docker-compose.yml has staging config
grep "acme-staging-v02" docker-compose.yml || echo "ERROR: Need to update docker-compose.yml"

# 5. Start services
docker compose up -d

# 6. Wait for certificates (60-120 seconds)
sleep 60

# 7. Check logs
docker compose logs traefik --tail 100 | grep -i "certificate\|acme"
```

## Expected Behavior

After fixing, you should see:

1. **Traefik logs show:**
   ```
   level=info msg="Certificate obtained for domain staging.stackyn.com"
   ```

2. **acme.json contains:**
   ```json
   {
     "letsencrypt": {
       "Certificates": [...]
     }
   }
   ```

3. **Site is accessible:**
   ```bash
   curl -k https://staging.stackyn.com
   # Returns HTML (may show certificate warning with staging certs)
   ```

## Still Not Working?

1. Run the diagnostic script: `./diagnose-ssl.sh`
2. Share the output
3. Check Traefik dashboard: `http://YOUR_VPS_IP:8081`
4. Review full logs: `docker compose logs --tail 200`

