# Frontend Deployment & Access Guide

This guide explains how to deploy the frontend and access it.

## Frontend Access URLs

### Option 1: Direct Access (Development/Testing)
**URL:** `http://YOUR_VPS_IP:3000`

- **Port:** 3000 (default)
- **No domain required**
- **HTTP only** (no HTTPS)
- **Best for:** Testing, development, internal use

### Option 2: Via Traefik with Domain (Production)
**URL:** `http://yourdomain.com` or `https://yourdomain.com` (if SSL configured)

- **Port:** 80 (HTTP) or 443 (HTTPS)
- **Requires:** Domain name and Traefik setup
- **Best for:** Production deployment

---

## Quick Setup: Direct Access (Port 3000)

### Method 1: Run with Docker (Recommended)

```bash
cd /opt/stackyn/frontend

# Build the frontend Docker image
# Replace YOUR_VPS_IP with your actual VPS IP address
docker build \
  --build-arg VITE_API_BASE_URL=http://YOUR_VPS_IP:8080 \
  -t stackyn-frontend .

# Run the container
docker run -d \
  --name stackyn-frontend \
  -p 3000:3000 \
  --restart unless-stopped \
  stackyn-frontend
```

**Access at:** `http://YOUR_VPS_IP:3000`

### Method 2: Run with Node.js (Development)

```bash
cd /opt/stackyn/frontend

# Install dependencies
npm install

# Set API URL
export VITE_API_BASE_URL=http://YOUR_VPS_IP:8080

# Build
npm run build

# Install serve globally
npm install -g serve

# Run
serve -s dist -l 3000
```

**Access at:** `http://YOUR_VPS_IP:3000`

### Method 3: Run with Systemd Service

```bash
# Create systemd service
sudo nano /etc/systemd/system/stackyn-frontend.service
```

**Content:**
```ini
[Unit]
Description=Stackyn Frontend
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/opt/stackyn/frontend
Environment="VITE_API_BASE_URL=http://YOUR_VPS_IP:8080"
ExecStart=/usr/bin/npx serve -s dist -l 3000
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

```bash
# Build first
cd /opt/stackyn/frontend
export VITE_API_BASE_URL=http://YOUR_VPS_IP:8080
npm install
npm run build

# Enable and start
sudo systemctl daemon-reload
sudo systemctl enable stackyn-frontend
sudo systemctl start stackyn-frontend
```

**Access at:** `http://YOUR_VPS_IP:3000`

---

## Production Setup: Via Traefik with Domain

### Step 1: Build Frontend with Production API URL

```bash
cd /opt/stackyn/frontend

# Build with your API domain
# Replace api.yourdomain.com with your actual API domain
docker build \
  --build-arg VITE_API_BASE_URL=https://api.yourdomain.com \
  -t stackyn-frontend:latest .
```

### Step 2: Run with Traefik Labels

```bash
# Create Traefik network if it doesn't exist
docker network create traefik 2>/dev/null || true

# Run frontend container with Traefik labels
docker run -d \
  --name stackyn-frontend \
  --network traefik \
  --label "traefik.enable=true" \
  --label "traefik.docker.network=traefik" \
  --label "traefik.http.routers.frontend-http.rule=Host(\`yourdomain.com\`)" \
  --label "traefik.http.routers.frontend-http.entrypoints=web" \
  --label "traefik.http.routers.frontend-http.middlewares=frontend-redirect" \
  --label "traefik.http.routers.frontend.rule=Host(\`yourdomain.com\`)" \
  --label "traefik.http.routers.frontend.entrypoints=websecure" \
  --label "traefik.http.routers.frontend.tls=true" \
  --label "traefik.http.routers.frontend.tls.certresolver=letsencrypt" \
  --label "traefik.http.services.frontend.loadbalancer.server.port=3000" \
  --label "traefik.http.middlewares.frontend-redirect.redirectscheme.scheme=https" \
  --label "traefik.http.middlewares.frontend-redirect.redirectscheme.permanent=true" \
  --restart unless-stopped \
  stackyn-frontend:latest
```

**Access at:** `https://yourdomain.com`

---

## Important Configuration

### API Base URL

The frontend needs to know where your API server is. Set this **at build time**:

```bash
# For direct access (no domain)
VITE_API_BASE_URL=http://YOUR_VPS_IP:8080

# For production (with domain)
VITE_API_BASE_URL=https://api.yourdomain.com
```

**⚠️ Important:** This must be set when building the Docker image or running `npm run build`. It cannot be changed at runtime.

---

## Firewall Configuration

If using direct access (port 3000), open the port:

```bash
# Allow port 3000
sudo ufw allow 3000/tcp

# Or if using Traefik, allow ports 80 and 443
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
```

---

## Quick Reference

### Find Your VPS IP Address

```bash
# Public IP
curl ifconfig.me

# Or
hostname -I
```

### Test Frontend Access

```bash
# Test direct access
curl http://localhost:3000

# Test from outside (replace with your VPS IP)
curl http://YOUR_VPS_IP:3000
```

### Check Frontend Container Status

```bash
# If using Docker
docker ps | grep frontend
docker logs stackyn-frontend

# If using systemd
sudo systemctl status stackyn-frontend
sudo journalctl -u stackyn-frontend -f
```

---

## Troubleshooting

### Frontend shows "Cannot connect to API"

**Problem:** Frontend can't reach the API server.

**Solutions:**
1. Check API URL in build: Make sure `VITE_API_BASE_URL` was set correctly when building
2. Check API server is running: `curl http://YOUR_VPS_IP:8080/health`
3. Check firewall: Ensure port 8080 is open
4. Rebuild frontend with correct API URL

### Port 3000 Already in Use

```bash
# Find what's using port 3000
sudo lsof -i :3000

# Kill the process or use a different port
# Change port in Docker: -p 3001:3000
# Change port in serve: serve -s dist -l 3001
```

### CORS Errors

If you see CORS errors, make sure:
1. API server allows requests from your frontend domain
2. API server has CORS middleware configured
3. Both frontend and API are using the same protocol (both HTTP or both HTTPS)

---

## Example: Complete Deployment Script

```bash
#!/bin/bash
# Complete frontend deployment script

VPS_IP="YOUR_VPS_IP_HERE"  # Replace with your VPS IP
API_URL="http://${VPS_IP}:8080"

cd /opt/stackyn/frontend

# Build Docker image
docker build \
  --build-arg VITE_API_BASE_URL="${API_URL}" \
  -t stackyn-frontend:latest .

# Stop existing container
docker stop stackyn-frontend 2>/dev/null || true
docker rm stackyn-frontend 2>/dev/null || true

# Run new container
docker run -d \
  --name stackyn-frontend \
  -p 3000:3000 \
  --restart unless-stopped \
  stackyn-frontend:latest

echo "Frontend deployed!"
echo "Access at: http://${VPS_IP}:3000"
```

Save as `deploy-frontend.sh`, make executable (`chmod +x deploy-frontend.sh`), and run it.

---

## Summary

**Quick Access (No Domain):**
- Build: `docker build --build-arg VITE_API_BASE_URL=http://YOUR_VPS_IP:8080 -t stackyn-frontend .`
- Run: `docker run -d -p 3000:3000 --name stackyn-frontend stackyn-frontend`
- Access: `http://YOUR_VPS_IP:3000`

**Production (With Domain):**
- Build: `docker build --build-arg VITE_API_BASE_URL=https://api.yourdomain.com -t stackyn-frontend .`
- Run: Use Traefik labels (see above)
- Access: `https://yourdomain.com`

