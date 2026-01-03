# VPS Deployment Guide for Stackyn

This guide covers all the changes needed to deploy Stackyn to a VPS with `staging.stackyn.com` as the landing page.

## Prerequisites

1. **VPS Requirements:**
   - Ubuntu 20.04+ or similar Linux distribution
   - Docker and Docker Compose installed
   - At least 4GB RAM, 2 CPU cores, 50GB+ disk space
   - Root or sudo access

2. **DNS Configuration:**
   - Point `staging.stackyn.com` → Your VPS IP
   - Point `api.staging.stackyn.com` → Your VPS IP
   - Point `console.staging.stackyn.com` → Your VPS IP (optional, for console subdomain)
   - Point `*.staging.stackyn.com` → Your VPS IP (wildcard for user apps)

3. **Firewall:**
   - Open ports: 80 (HTTP), 443 (HTTPS)
   - Optional: 8080 (API direct access), 8081 (Traefik dashboard)

## Step 1: Environment Variables (.env file)

Create a `.env` file in the project root with these production values:

```bash
# ============================================
# PRODUCTION CONFIGURATION
# ============================================

# Database Configuration
POSTGRES_PASSWORD=<STRONG_PASSWORD_HERE>  # Generate: openssl rand -base64 32

# Redis Configuration
REDIS_PASSWORD=<STRONG_PASSWORD_HERE>  # Generate: openssl rand -base64 32

# JWT Secret (REQUIRED - Generate a new one!)
JWT_SECRET=<GENERATE_NEW_SECRET>  # openssl rand -base64 32

# Frontend Configuration
FRONTEND_API_URL=https://api.staging.stackyn.com
FRONTEND_DOMAIN=staging.stackyn.com
CONSOLE_DOMAIN=console.staging.stackyn.com

# API Domain (for Traefik routing)
API_DOMAIN=api.staging.stackyn.com

# Base Domain for User Apps (NEW - Required for subdomain generation)
APP_BASE_DOMAIN=staging.stackyn.com

# Let's Encrypt Email (REQUIRED for SSL certificates)
ACME_EMAIL=your-email@example.com

# Email Configuration
RESEND_API_KEY=re_6iU1KmCf_3p6MzQRbsDyerP736x1WWExj
EMAIL_FROM_EMAIL=noreply@stackyn.com
```

## Step 2: Code Changes Required

### 2.1 Update Subdomain Generation (server/internal/tasks/handlers.go)

The subdomain is currently hardcoded to `.stackyn.local`. We need to make it configurable:

**Current code (line ~645):**
```go
subdomain = fmt.Sprintf("%s.stackyn.local", payload.AppID)
```

**Change to:**
```go
// Get base domain from environment variable
baseDomain := os.Getenv("APP_BASE_DOMAIN")
if baseDomain == "" {
    baseDomain = "staging.stackyn.com" // Default fallback
}
subdomain = fmt.Sprintf("%s.%s", payload.AppID, baseDomain)
```

### 2.2 Update URL Generation (server/internal/tasks/handlers.go)

**Current code (line ~829):**
```go
appURL := fmt.Sprintf("http://%s", deployOpts.Subdomain)
```

**Change to:**
```go
// Generate URL based on domain type
var appURL string
if strings.HasSuffix(deployOpts.Subdomain, ".local") || strings.HasSuffix(deployOpts.Subdomain, ".localhost") {
    appURL = fmt.Sprintf("http://%s", deployOpts.Subdomain)
} else {
    appURL = fmt.Sprintf("https://%s", deployOpts.Subdomain)
}
```

### 2.3 Update CORS Configuration (server/internal/api/router.go)

The CORS already supports staging domains, but verify it includes your domain:

```go
// Should already have this, but verify:
allowedOrigins := []string{
    "http://localhost:3000",
    "http://localhost:3001",
    "http://localhost:5173",
    "https://staging.stackyn.com",
    "https://console.staging.stackyn.com",
    "https://api.staging.stackyn.com",
}
```

## Step 3: Docker Compose Configuration

The `docker-compose.yml` already has production support, but verify these settings:

### 3.1 Remove Local Development Port Exposures (Optional)

For production, you can remove direct port mappings:

**In `api` service (line ~113-114):**
```yaml
# Remove or comment out for production:
# ports:
#   - "8080:8080"
```

**In `frontend` service (line ~264-265):**
```yaml
# Remove or comment out for production:
# ports:
#   - "3000:3000"
```

### 3.2 Verify Traefik Configuration

The Traefik labels already support production domains. Verify:
- `API_DOMAIN` environment variable is set correctly
- Let's Encrypt email is configured
- Ports 80 and 443 are exposed

## Step 4: Frontend Build Configuration

The frontend needs to be built with the correct API URL:

**In `docker-compose.yml` (line ~262):**
```yaml
args:
  VITE_API_BASE_URL: ${FRONTEND_API_URL:-https://api.staging.stackyn.com}
```

This is already configured correctly.

## Step 5: Deployment Steps

### 5.1 On Your VPS:

```bash
# 1. Clone your repository
git clone <your-repo-url> stackyn
cd stackyn

# 2. Create .env file with production values (see Step 1)
nano .env

# 3. Make code changes (see Step 2)
# Edit server/internal/tasks/handlers.go

# 4. Build and start services
docker compose build
docker compose up -d

# 5. Check logs
docker compose logs -f

# 6. Verify services are running
docker compose ps
```

### 5.2 Verify DNS Resolution

```bash
# Test DNS resolution
dig staging.stackyn.com
dig api.staging.stackyn.com
dig *.staging.stackyn.com  # Should resolve to your VPS IP
```

### 5.3 Test SSL Certificate Generation

After starting services, check Traefik logs:

```bash
docker compose logs traefik | grep -i acme
```

You should see Let's Encrypt certificate generation attempts. It may take a few minutes.

## Step 6: Security Checklist

- [ ] Changed `POSTGRES_PASSWORD` from default
- [ ] Set `REDIS_PASSWORD` (not empty)
- [ ] Generated new `JWT_SECRET` (not using default)
- [ ] Set `ACME_EMAIL` to your real email
- [ ] Removed or secured port mappings (8080, 3000, 8081)
- [ ] Firewall configured (only 80, 443 open)
- [ ] `.env` file has proper permissions: `chmod 600 .env`
- [ ] Database backups configured (optional but recommended)

## Step 7: Monitoring & Maintenance

### 7.1 Check Service Health

```bash
# Check all services
docker compose ps

# Check specific service logs
docker compose logs api
docker compose logs traefik
docker compose logs build-worker
```

### 7.2 Monitor Disk Space

The cleanup worker should handle this, but monitor:

```bash
df -h
docker system df
```

### 7.3 Update Services

```bash
# Pull latest code
git pull

# Rebuild and restart
docker compose build
docker compose up -d

# Or restart specific service
docker compose restart api
```

## Step 8: Troubleshooting

### SSL Certificate Issues

If SSL certificates aren't generating:

1. **Check DNS:** Ensure all domains point to your VPS IP
2. **Check Traefik logs:** `docker compose logs traefik`
3. **Check firewall:** Ports 80 and 443 must be open
4. **Wait:** Let's Encrypt has rate limits, may take 5-10 minutes

### App Subdomains Not Working

1. **Check DNS:** Wildcard `*.staging.stackyn.com` must point to VPS
2. **Check Traefik labels:** Verify containers have correct labels
3. **Check logs:** `docker compose logs deploy-worker`

### Database Connection Issues

1. **Check password:** Verify `POSTGRES_PASSWORD` in `.env`
2. **Check network:** Services must be on `stackyn-network`
3. **Check health:** `docker compose ps` - postgres should be healthy

## Step 9: Post-Deployment Verification

1. **Landing Page:** `https://staging.stackyn.com` should load
2. **API:** `https://api.staging.stackyn.com/health` should return 200
3. **Console:** `https://console.staging.stackyn.com` should load (if configured)
4. **SSL:** All domains should show valid SSL certificates
5. **Deploy Test App:** Create and deploy an app, verify subdomain works

## Additional Notes

- **Traefik Dashboard:** Accessible at `http://your-vps-ip:8081` (consider securing this)
- **Database Access:** Use `docker compose exec postgres psql -U stackyn_user -d stackyn` for direct access
- **Logs Location:** `./server/logs/` directory contains application logs
- **Backups:** Consider setting up automated database backups

## Quick Reference: Environment Variables

| Variable | Local Dev | Production (VPS) |
|----------|-----------|------------------|
| `POSTGRES_PASSWORD` | `changeme` | `<strong-password>` |
| `REDIS_PASSWORD` | (empty) | `<strong-password>` |
| `JWT_SECRET` | (default) | `<new-secret>` |
| `API_DOMAIN` | `localhost` | `api.staging.stackyn.com` |
| `APP_BASE_DOMAIN` | (not set) | `staging.stackyn.com` |
| `FRONTEND_API_URL` | `http://localhost:8080` | `https://api.staging.stackyn.com` |
| `ACME_EMAIL` | (optional) | `your-email@example.com` |

## Support

If you encounter issues:
1. Check service logs: `docker compose logs <service-name>`
2. Verify environment variables: `docker compose config`
3. Check Traefik dashboard for routing issues
4. Verify DNS resolution for all domains

