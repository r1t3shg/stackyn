# Stackyn VPS Deployment Checklist

Use this checklist to ensure you complete all deployment steps.

## Pre-Deployment

- [ ] VPS provisioned (Ubuntu 20.04+ recommended)
- [ ] Root/sudo access confirmed
- [ ] SSH access configured
- [ ] Domain name configured (optional)

## Step 1: System Setup

- [ ] System updated (`apt update && apt upgrade`)
- [ ] Basic tools installed (git, curl, wget, build-essential)

## Step 2: Install Dependencies

- [ ] Go installed and in PATH
- [ ] Docker installed and user added to docker group
- [ ] PostgreSQL installed and running
- [ ] Redis installed and running
- [ ] Traefik installed (optional)

## Step 3: Database Setup

- [ ] PostgreSQL database `stackyn` created
- [ ] PostgreSQL user `stackyn_user` created
- [ ] Database permissions granted
- [ ] Database connection tested

## Step 4: Project Setup

- [ ] Repository cloned to `/opt/stackyn`
- [ ] Environment file created from `configs/env.example`
- [ ] All environment variables configured:
  - [ ] `POSTGRES_PASSWORD` set
  - [ ] `REDIS_PASSWORD` set (if using password)
  - [ ] `JWT_SECRET` generated (use `openssl rand -base64 32`)
  - [ ] `DOCKER_HOST` set correctly
  - [ ] Other variables configured

## Step 5: Build Application

- [ ] Go dependencies downloaded (`go mod download`)
- [ ] API binary built (`go build -o bin/api ./cmd/api`)
- [ ] Build worker binary built
- [ ] Deploy worker binary built
- [ ] Cleanup worker binary built
- [ ] All binaries verified in `bin/` directory

## Step 6: Database Migrations

- [ ] Migrations run successfully
- [ ] Database tables created
- [ ] Database connection verified from application

## Step 7: Systemd Services

- [ ] `stackyn-api.service` created
- [ ] `stackyn-build-worker.service` created
- [ ] `stackyn-deploy-worker.service` created
- [ ] `stackyn-cleanup-worker.service` created
- [ ] Services enabled (`systemctl enable`)
- [ ] Services started (`systemctl start`)
- [ ] All services show "active (running)" status

## Step 8: Verification

- [ ] API server responding (test with `curl http://localhost:8080`)
- [ ] Workers running (check logs)
- [ ] Docker accessible
- [ ] PostgreSQL accessible
- [ ] Redis accessible
- [ ] No errors in service logs

## Step 9: Security

- [ ] Firewall configured (UFW or similar)
- [ ] Only necessary ports exposed
- [ ] Strong passwords set for all services
- [ ] SSH key authentication configured (disable password auth)
- [ ] Regular user created (not using root for daily tasks)

## Step 10: Monitoring & Maintenance

- [ ] Log rotation configured
- [ ] Backup strategy planned
- [ ] Monitoring setup (optional)
- [ ] Update script created (`/opt/stackyn/update.sh`)

## Post-Deployment

- [ ] Test creating an app via API
- [ ] Test build process
- [ ] Test deployment process
- [ ] Verify logs are being captured
- [ ] Check resource usage (CPU, RAM, disk)

## Troubleshooting Notes

If services fail to start:
1. Check logs: `sudo journalctl -u stackyn-api -n 50`
2. Verify environment variables: `sudo systemctl show stackyn-api --property=Environment`
3. Test binary manually: `cd /opt/stackyn/server && ./bin/api`
4. Check dependencies: PostgreSQL, Redis, Docker all running

## Quick Commands Reference

```bash
# Check all services
sudo systemctl status stackyn-api stackyn-build-worker stackyn-deploy-worker stackyn-cleanup-worker

# View logs
sudo journalctl -u stackyn-api -f

# Restart services
sudo systemctl restart stackyn-api stackyn-build-worker stackyn-deploy-worker stackyn-cleanup-worker

# Update deployment
cd /opt/stackyn && git pull && cd server && go build -o bin/api ./cmd/api && sudo systemctl restart stackyn-api
```

