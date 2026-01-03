# CI/CD .env File Setup for VPS

## Current Workflow Behavior

The `deploy-develop.yml` workflow **will automatically** create `.env` from `env.example` if it doesn't exist, but **deployment will fail** if required variables are not set.

### What Happens During Deployment

1. **Workflow checks for `.env` file**
   - If `.env` exists → Continues
   - If `.env` doesn't exist → Creates from `env.example`

2. **Workflow validates required variables**
   - Checks: `POSTGRES_PASSWORD`, `JWT_SECRET`, `API_DOMAIN`, `APP_BASE_DOMAIN`, `ACME_EMAIL`
   - If any are missing → **Deployment fails**

## ✅ Solution: Set up .env BEFORE first deployment

### Option 1: Manual Setup (Recommended)

**Before the first CI/CD deployment, SSH into your VPS and set up .env:**

```bash
# SSH into VPS
ssh root@your-vps-ip

# Navigate to project directory
cd /opt/stackyn

# Create .env from template
cp env.example .env

# Edit .env and set all required values
nano .env

# Set these values (from your env.example):
# - POSTGRES_PASSWORD=Ritesh@7033
# - REDIS_PASSWORD=imGLZh/9GJcJqBpiIoMSz8dOH1sPWPhF6LcM4lCQsY8=
# - JWT_SECRET=hx36Lh9uewWtzDI4pCWMHbMDiWPK3luiZALY56ssg1Q=
# - APP_BASE_DOMAIN=staging.stackyn.com
# - API_DOMAIN=api.staging.stackyn.com
# - ACME_EMAIL=kaviriteshgupta@gmail.com

# Secure the file
chmod 600 .env
```

### Option 2: Let CI/CD Create It (Then Edit)

If you let CI/CD create the `.env` file:

1. **First deployment will fail** (expected - variables not set)
2. **SSH into VPS** after first failed deployment
3. **Edit the auto-created `.env` file** with actual values
4. **Redeploy** - second deployment will succeed

## Updated Workflow

I've updated the workflow to:
- ✅ Check for `env.example` first (the actual file name)
- ✅ Fall back to `.env.production.example` if it exists
- ✅ Create `.env` automatically if missing
- ✅ Validate required variables before deployment
- ✅ Fail with clear error message if variables are missing

## Verification Commands

After setting up `.env` on VPS, verify:

```bash
# Check .env exists
ls -la .env

# Verify Docker Compose can read variables
docker compose config | grep -E "APP_BASE_DOMAIN|POSTGRES_PASSWORD"

# Should show:
#   APP_BASE_DOMAIN: staging.stackyn.com
#   POSTGRES_PASSWORD: Ritesh@7033
```

## Summary

**Answer: The workflow will try to create `.env` automatically, but you MUST set the values manually before deployment succeeds.**

**Best Practice:** Set up `.env` manually on VPS before the first CI/CD deployment to avoid failed deployments.

