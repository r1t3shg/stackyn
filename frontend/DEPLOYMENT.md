# Frontend Deployment Guide for VPS

This guide explains how to deploy the React (Vite) frontend application to a VPS.

## Prerequisites

- A VPS with Ubuntu/Debian (or similar Linux distribution)
- Node.js 20+ installed (for building)
- Nginx (for serving static files)
- Domain name pointing to your VPS (optional but recommended)
- SSL certificate (Let's Encrypt recommended)

## Option 1: Static File Deployment with Nginx

### Step 1: Prepare Your VPS

```bash
# Update system packages
sudo apt update && sudo apt upgrade -y

# Install Node.js 20
curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
sudo apt install -y nodejs

# Install PM2 for process management
sudo npm install -g pm2

# Install Nginx
sudo apt install -y nginx
```

### Step 2: Clone and Build the Application

```bash
# Navigate to your deployment directory
cd /var/www

# Clone your repository (or upload files)
git clone <your-repo-url> stackyn-frontend
cd stackyn-frontend/frontend

# Install dependencies
npm ci

# Set environment variable
export VITE_API_BASE_URL=https://staging.stackyn.com

# Build the application (creates dist/ directory)
npm run build
```

### Step 3: Create Environment File

Create a `.env.production` file in the frontend directory:

```bash
echo "VITE_API_BASE_URL=https://staging.stackyn.com" > .env.production
```

### Step 4: Configure Nginx to Serve Static Files

Create an Nginx configuration file:

```bash
sudo nano /etc/nginx/sites-available/stackyn-frontend
```

Add the following configuration (replace `your-domain.com` with your domain and `/var/www/stackyn-frontend/frontend/dist` with your actual build path):

```nginx
server {
    listen 80;
    server_name your-domain.com;

    # Root directory (where Vite builds to)
    root /var/www/stackyn-frontend/frontend/dist;
    index index.html;

    # Gzip compression
    gzip on;
    gzip_vary on;
    gzip_min_length 1024;
    gzip_types text/plain text/css text/xml text/javascript application/x-javascript application/xml+rss application/json application/javascript;

    # Handle React Router (client-side routing)
    location / {
        try_files $uri $uri/ /index.html;
    }

    # Cache static assets
    location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2|ttf|eot)$ {
        expires 1y;
        add_header Cache-Control "public, immutable";
    }

    # Don't cache index.html
    location = /index.html {
        add_header Cache-Control "no-cache, no-store, must-revalidate";
        add_header Pragma "no-cache";
        add_header Expires "0";
    }
}

# If using SSL (recommended)
# server {
#     listen 443 ssl http2;
#     server_name your-domain.com;
#
#     ssl_certificate /etc/letsencrypt/live/your-domain.com/fullchain.pem;
#     ssl_certificate_key /etc/letsencrypt/live/your-domain.com/privkey.pem;
#
#     location / {
#         proxy_pass http://localhost:3000;
#         proxy_http_version 1.1;
#         proxy_set_header Upgrade $http_upgrade;
#         proxy_set_header Connection 'upgrade';
#         proxy_set_header Host $host;
#         proxy_set_header X-Real-IP $remote_addr;
#         proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
#         proxy_set_header X-Forwarded-Proto $scheme;
#         proxy_cache_bypass $http_upgrade;
#     }
# }
```

Enable the site:

```bash
sudo ln -s /etc/nginx/sites-available/stackyn-frontend /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

### Step 6: Setup SSL (Optional but Recommended)

```bash
# Install Certbot
sudo apt install -y certbot python3-certbot-nginx

# Obtain SSL certificate
sudo certbot --nginx -d your-domain.com

# Certbot will automatically configure Nginx and set up auto-renewal
```

## Option 2: Docker Deployment

### Step 1: Install Docker

```bash
# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh

# Install Docker Compose
sudo apt install -y docker-compose
```

### Step 2: Build and Run Docker Container

```bash
cd /var/www/stackyn-frontend/frontend

# Build the Docker image
docker build -t stackyn-frontend .

# Run the container
docker run -d \
  --name stackyn-frontend \
  -p 3000:3000 \
  -e NEXT_PUBLIC_API_BASE_URL=https://staging.stackyn.com \
  --restart unless-stopped \
  stackyn-frontend
```

### Step 3: Configure Nginx

Follow Step 5 from Option 1 to configure Nginx as a reverse proxy.

## Environment Variables

The application uses the following environment variable:

- `VITE_API_BASE_URL`: The base URL for the API (default: `http://localhost:8080`)
  - For production: `https://staging.stackyn.com`

**Important**: In Vite, environment variables prefixed with `VITE_` are embedded into the JavaScript bundle at build time. Make sure to set this variable before running `npm run build`.

## Updating the Application

### For Static File Deployment:

**Using the deployment script:**

```bash
cd /var/www/stackyn-frontend/frontend
git pull
chmod +x deploy.sh
./deploy.sh
```

**Manual update:**

```bash
cd /var/www/stackyn-frontend/frontend
git pull
npm ci
export VITE_API_BASE_URL=https://staging.stackyn.com
npm run build
# Nginx will automatically serve the new files from dist/
```

### For Docker Deployment:

```bash
cd /var/www/stackyn-frontend/frontend
git pull
docker build -t stackyn-frontend .
docker stop stackyn-frontend
docker rm stackyn-frontend
docker run -d \
  --name stackyn-frontend \
  -p 3000:80 \
  --restart unless-stopped \
  stackyn-frontend
```

Note: The Docker image uses nginx to serve the static files. The build process embeds the API URL at build time.

## Troubleshooting

### Check Application Logs

```bash
# Nginx logs
sudo tail -f /var/log/nginx/access.log
sudo tail -f /var/log/nginx/error.log

# Docker logs
docker logs stackyn-frontend
```

### Check Nginx Status

```bash
sudo systemctl status nginx
sudo nginx -t
```

### Verify Environment Variable

```bash
# Check if the environment variable is set correctly
echo $VITE_API_BASE_URL

# Check built files (API URL should be embedded in the JS bundle)
grep -r "staging.stackyn.com" dist/
```

### Test API Connection

```bash
curl https://staging.stackyn.com/health
```

## Security Considerations

1. **Firewall**: Configure UFW or iptables to only allow necessary ports (80, 443, 22)
2. **SSL**: Always use HTTPS in production
3. **Environment Variables**: Never commit `.env.production` to version control
4. **Updates**: Keep Node.js, Nginx, and system packages updated

## Monitoring

Consider setting up monitoring with:
- Nginx access/error logs
- Application health checks
- Uptime monitoring services
- File system monitoring for the dist/ directory

