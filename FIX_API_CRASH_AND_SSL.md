# Fix API Crash and SSL Certificate Issues

## Issues Found

1. **API Container Crashing**: Duplicate `/api/billing` route registration causing panic
2. **Traefik Router Conflicts**: Multiple router definitions causing conflicts
3. **SSL Certificate**: Still using default self-signed certificate

## Fixes Applied

### 1. Fixed API Crash (Duplicate Billing Route)

**Problem**: Two `r.Route("/api/billing", ...)` blocks were defined:
- Line 315: Authenticated billing routes
- Line 363: Webhook route

**Solution**: Combined both into a single route block with webhook route first (no auth) and authenticated routes using `r.With()`.

**File**: `server/internal/api/router.go`

### 2. Next Steps for SSL Certificate

After fixing the API crash, you need to:

1. **Rebuild and restart the API container**:
   ```bash
   cd /opt/stackyn
   docker-compose build api
   docker-compose up -d
   ```

2. **Check for duplicate containers** (causing Traefik router conflicts):
   ```bash
   # Check for old/duplicate containers
   docker ps -a | grep stackyn
   
   # Remove any old containers
   docker rm -f <old-container-names>
   ```

3. **Clean up Traefik and restart**:
   ```bash
   # Stop everything
   docker-compose down
   
   # Remove any orphaned networks
   docker network prune -f
   
   # Start fresh
   docker-compose up -d
   ```

4. **Monitor API startup**:
   ```bash
   docker-compose logs -f api
   ```
   Should see API starting without panic errors.

5. **Trigger certificate generation**:
   ```bash
   # Make a request to trigger ACME
   curl -k https://api.staging.stackyn.com/health
   
   # Monitor Traefik logs for certificate generation
   docker-compose logs -f traefik | grep -i -E "(certificate obtained|acme|error)"
   ```

## Verification

After applying fixes:

1. **API should start without panics**:
   ```bash
   docker-compose ps api
   # Should show "Up" status
   ```

2. **No duplicate router errors in Traefik**:
   ```bash
   docker-compose logs traefik | grep "Router defined multiple times"
   # Should be empty or minimal
   ```

3. **Certificate should generate**:
   ```bash
   # Wait 1-2 minutes, then check
   echo | openssl s_client -connect api.staging.stackyn.com:443 -servername api.staging.stackyn.com 2>/dev/null | openssl x509 -noout -issuer
   # Should show "Let's Encrypt" not "TRAEFIK DEFAULT CERT"
   ```

## Troubleshooting

If API still crashes:
- Check logs: `docker-compose logs api | tail -50`
- Verify the fix was applied: `grep -A 10 "/api/billing" server/internal/api/router.go`

If Traefik still has router conflicts:
- Check for duplicate containers: `docker ps -a | grep stackyn`
- Remove old containers and restart: `docker-compose down && docker-compose up -d`

If certificate doesn't generate:
- Check Traefik logs: `docker-compose logs traefik | grep -i acme | tail -30`
- Verify DNS: `dig api.staging.stackyn.com`
- Verify port 80: `curl -I http://api.staging.stackyn.com`

