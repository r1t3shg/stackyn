# Deployment Guide

## Preserving SSL Certificates During Deployment

**IMPORTANT:** SSL certificates are stored in the `traefik_data` Docker volume. To preserve certificates across deployments, **NEVER** use `docker compose down -v` as this will delete all volumes including your SSL certificates.

### Safe Deployment Steps

1. **Pull latest code:**
   ```bash
   git pull origin develop
   ```

2. **Rebuild containers (if needed):**
   ```bash
   docker compose build
   ```

3. **Restart services (preserves volumes):**
   ```bash
   docker compose up -d
   ```

4. **Or restart specific services:**
   ```bash
   docker compose restart api frontend
   ```

### What NOT to Do

❌ **DO NOT** run:
```bash
docker compose down -v    # This deletes ALL volumes including SSL certificates!
docker compose down --volumes  # Same as above
```

✅ **Instead**, use:
```bash
docker compose down        # Stops containers but preserves volumes
docker compose up -d       # Restarts with existing volumes
```

### Verifying SSL Certificates Are Preserved

After deployment, verify your certificates are still there:

```bash
# Check if acme.json exists and has certificates
docker compose exec traefik cat /letsencrypt/acme.json | grep -q "Certificates" && echo "✓ Certificates preserved" || echo "✗ No certificates found"

# Check certificate details
echo | openssl s_client -connect api.staging.stackyn.com:443 -servername api.staging.stackyn.com 2>/dev/null | openssl x509 -noout -issuer -subject
```

### Volume Backup (Optional but Recommended)

Before major deployments, you can backup the SSL certificates:

```bash
# Backup acme.json
docker compose exec traefik cp /letsencrypt/acme.json /letsencrypt/acme.json.backup.$(date +%Y%m%d_%H%M%S)

# Or backup the entire volume
docker run --rm -v stackyn_traefik_data:/data -v $(pwd):/backup alpine tar czf /backup/traefik_data_backup_$(date +%Y%m%d_%H%M%S).tar.gz /data
```

### Restoring SSL Certificates

If certificates are accidentally deleted:

1. **Restore from backup:**
   ```bash
   docker compose exec traefik cp /letsencrypt/acme.json.backup.YYYYMMDD_HHMMSS /letsencrypt/acme.json
   docker compose restart traefik
   ```

2. **Or let Traefik regenerate (will take 1-2 minutes):**
   ```bash
   # Certificates will auto-regenerate on next HTTPS request
   curl https://api.staging.stackyn.com/health
   ```

### Docker Volume Management

List all volumes:
```bash
docker volume ls | grep stackyn
```

Inspect traefik_data volume:
```bash
docker volume inspect stackyn_traefik_data
```

Remove a volume (⚠️ **ONLY** if you want to delete SSL certificates):
```bash
docker volume rm stackyn_traefik_data
```

