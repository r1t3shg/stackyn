# VPS Deployment Checklist

Quick checklist for deploying Stackyn to VPS with `staging.stackyn.com`.

## ✅ Pre-Deployment Checklist

### DNS Configuration
- [ ] `staging.stackyn.com` → VPS IP
- [ ] `api.staging.stackyn.com` → VPS IP  
- [ ] `console.staging.stackyn.com` → VPS IP (optional)
- [ ] `*.staging.stackyn.com` → VPS IP (wildcard for user apps)

### VPS Setup
- [ ] Docker installed
- [ ] Docker Compose installed
- [ ] Firewall: Ports 80, 443 open
- [ ] Git access to repository

## ✅ Code Changes (Already Done)

- [x] Subdomain generation now uses `APP_BASE_DOMAIN` env var
- [x] URL generation uses HTTPS for production domains

## ✅ Environment Variables (.env file)

Create `.env` file with these values:

```bash
# Required - Generate strong passwords
POSTGRES_PASSWORD=<generate-with-openssl-rand-base64-32>
REDIS_PASSWORD=<generate-with-openssl-rand-base64-32>
JWT_SECRET=<generate-with-openssl-rand-base64-32>

# Required - Domain configuration
API_DOMAIN=api.staging.stackyn.com
APP_BASE_DOMAIN=staging.stackyn.com
FRONTEND_API_URL=https://api.staging.stackyn.com
FRONTEND_DOMAIN=staging.stackyn.com
CONSOLE_DOMAIN=console.staging.stackyn.com

# Required - SSL certificates
ACME_EMAIL=your-email@example.com

# Optional - Email
RESEND_API_KEY=re_6iU1KmCf_3p6MzQRbsDyerP736x1WWExj
EMAIL_FROM_EMAIL=noreply@stackyn.com
```

## ✅ Deployment Steps

```bash
# 1. Clone repository
git clone <repo-url> stackyn && cd stackyn

# 2. Create .env file (see above)

# 3. Build and start
docker compose build
docker compose up -d

# 4. Check status
docker compose ps
docker compose logs -f
```

## ✅ Post-Deployment Verification

- [ ] `https://staging.stackyn.com` loads
- [ ] `https://api.staging.stackyn.com/health` returns 200
- [ ] SSL certificates are valid (green lock in browser)
- [ ] Can create and deploy an app
- [ ] App subdomain works (e.g., `https://{app-id}.staging.stackyn.com`)

## ✅ Security Checklist

- [ ] `.env` file permissions: `chmod 600 .env`
- [ ] Removed default passwords
- [ ] Generated new JWT_SECRET
- [ ] Firewall configured (only 80, 443 open)
- [ ] Traefik dashboard secured (port 8081) or disabled

## 📝 Quick Commands

```bash
# View logs
docker compose logs -f api
docker compose logs -f traefik

# Restart service
docker compose restart api

# Update code
git pull && docker compose build && docker compose up -d

# Check disk space
df -h
docker system df

# Database access
docker compose exec postgres psql -U stackyn_user -d stackyn
```

## 🐛 Troubleshooting

**SSL not working?**
- Check DNS: `dig staging.stackyn.com`
- Check Traefik logs: `docker compose logs traefik | grep acme`
- Wait 5-10 minutes for Let's Encrypt

**Apps not deploying?**
- Check deploy-worker logs: `docker compose logs deploy-worker`
- Verify wildcard DNS: `dig *.staging.stackyn.com`

**Database connection issues?**
- Check password in `.env`
- Check service health: `docker compose ps`

## 📚 Full Guide

See `VPS_DEPLOYMENT_GUIDE.md` for detailed instructions.

