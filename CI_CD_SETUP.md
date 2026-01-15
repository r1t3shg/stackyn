# CI/CD Setup Guide for VPS Deployment

## Overview

The project uses GitHub Actions to automatically build, test, and deploy to your VPS when code is pushed to the `develop` branch.

## Workflow: `.github/workflows/deploy-develop.yml`

### What It Does

1. **CI Job (Build & Test)**
   - Lints and builds the frontend
   - Verifies backend Go code compiles
   - Runs on every push to `develop`

2. **Deploy Job (VPS Deployment)**
   - Only runs if CI passes
   - Connects to VPS via SSH
   - Pulls latest code
   - Builds and deploys Docker containers
   - Verifies deployment health

## Required GitHub Secrets

Configure these secrets in your GitHub repository settings (`Settings > Secrets and variables > Actions`):

### 1. `STAGING_SSH_KEY`
- **Description**: Private SSH key for connecting to your VPS
- **How to get**: 
  ```bash
  # On your local machine, generate a key pair if you don't have one
  ssh-keygen -t ed25519 -C "github-actions" -f ~/.ssh/github_actions
  
  # Copy the PRIVATE key (id_ed25519 or id_rsa)
  cat ~/.ssh/github_actions
  
  # Add the PUBLIC key to your VPS authorized_keys
  ssh-copy-id -i ~/.ssh/github_actions.pub root@your-vps-ip
  ```
- **Format**: Full private key including headers:
  ```
  -----BEGIN OPENSSH PRIVATE KEY-----
  ...
  -----END OPENSSH PRIVATE KEY-----
  ```

### 2. `STAGING_HOST`
- **Description**: IP address or hostname of your VPS
- **Example**: `123.45.67.89` or `staging.stackyn.com`

### 3. `FRONTEND_API_URL` (Optional)
- **Description**: API URL for frontend build
- **Default**: `https://api.staging.stackyn.com`
- **Only set if different from default**

## VPS Setup Requirements

### 1. Initial Server Setup

On your VPS, ensure:

```bash
# 1. Clone repository to /opt/stackyn
cd /opt
git clone <your-repo-url> stackyn
cd stackyn

# 2. Create .env file with production values
cp .env.production.example .env
nano .env  # Edit and set all required values

# 3. Ensure Docker and Docker Compose are installed
docker --version
docker compose version

# 4. Add SSH public key to authorized_keys
# (The public key from STAGING_SSH_KEY secret)
```

### 2. Required Environment Variables in `.env`

Make sure your VPS `.env` file has:

```bash
# Required
POSTGRES_PASSWORD=<strong-password>
REDIS_PASSWORD=<strong-password>
JWT_SECRET=<strong-secret>
API_DOMAIN=api.staging.stackyn.com
APP_BASE_DOMAIN=staging.stackyn.com
ACME_EMAIL=your-email@example.com

# Optional (has defaults)
FRONTEND_API_URL=https://api.staging.stackyn.com
FRONTEND_DOMAIN=staging.stackyn.com
CONSOLE_DOMAIN=console.staging.stackyn.com
RESEND_API_KEY=re_6iU1KmCf_3p6MzQRbsDyerP736x1WWExj
EMAIL_FROM_EMAIL=noreply@stackyn.com
```

## Deployment Process

### Automatic Deployment

1. **Push to `develop` branch**
   ```bash
   git checkout develop
   git push origin develop
   ```

2. **GitHub Actions will:**
   - Run CI (build & test)
   - If CI passes, deploy to VPS
   - Pull latest code on VPS
   - Build Docker containers
   - Start services
   - Verify health

### Manual Deployment (if needed)

If you need to deploy manually on the VPS:

```bash
cd /opt/stackyn
git pull origin develop
docker compose down
docker compose build
docker compose up -d
docker compose ps
docker compose logs -f
```

## Troubleshooting

### CI Build Fails

**Error: "undefined: hasDockerCompose"**
- **Cause**: Build cache issue or wrong directory
- **Solution**: The workflow uses `working-directory: ./server` which should fix this
- **If persists**: Check that `go.mod` is in the `server` directory

### Deployment Fails - SSH Connection

**Error: "SSH connection test failed"**
- Check `STAGING_SSH_KEY` secret includes full private key with headers
- Verify public key is in VPS `~/.ssh/authorized_keys`
- Check `STAGING_HOST` is correct
- Ensure VPS firewall allows SSH (port 22)

### Deployment Fails - Missing Environment Variables

**Error: "Missing required environment variables"**
- The workflow now checks for required variables
- Ensure `.env` file exists on VPS with all required values
- If missing, the workflow will create from template but you must edit it

### Services Don't Start

**Error: "Some services failed to start"**
- Check logs: `docker compose logs`
- Verify `.env` file has correct values
- Check DNS is configured correctly
- Ensure ports 80 and 443 are open in firewall

### SSL Certificates Not Working

**Issue: SSL certificates not generating**
- Wait 5-10 minutes for Let's Encrypt
- Check Traefik logs: `docker compose logs traefik | grep -i acme`
- Verify DNS points to VPS IP
- Check `ACME_EMAIL` is set correctly

## Workflow Improvements Made

The workflow now includes:

1. ✅ **Environment Variable Validation**
   - Checks for required variables before deployment
   - Creates `.env` from template if missing (with warning)

2. ✅ **Better Error Handling**
   - More detailed error messages
   - Health checks after deployment
   - API health verification

3. ✅ **Improved Logging**
   - Shows deployment URLs
   - Better status messages
   - Helpful troubleshooting tips

## Monitoring Deployments

### View Workflow Runs

1. Go to GitHub repository
2. Click "Actions" tab
3. Select "Deploy to Develop" workflow
4. View logs for each step

### Check VPS Status

```bash
# SSH into VPS
ssh root@your-vps-ip

# Check service status
cd /opt/stackyn
docker compose ps

# View logs
docker compose logs -f

# Check specific service
docker compose logs api
docker compose logs traefik
```

## Next Steps

1. ✅ Configure GitHub Secrets (`STAGING_SSH_KEY`, `STAGING_HOST`)
2. ✅ Set up VPS with repository cloned to `/opt/stackyn`
3. ✅ Create `.env` file on VPS with all required variables
4. ✅ Push to `develop` branch to trigger deployment
5. ✅ Monitor deployment in GitHub Actions
6. ✅ Verify services are running on VPS

## Security Notes

- 🔒 Never commit `.env` file to repository
- 🔒 Use strong passwords for production
- 🔒 Rotate SSH keys regularly
- 🔒 Keep GitHub secrets secure
- 🔒 Use firewall to restrict access (only ports 80, 443 open)

