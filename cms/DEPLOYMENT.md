# CMS Deployment Guide

## Access URL

The CMS is accessible at: **https://admin.staging.stackyn.com**

## Configuration

### 1. Build Configuration

The CMS is configured to use `/` as the base path (root):
- `vite.config.ts` has `base: '/'`
- `App.tsx` uses `basename="/"` for React Router
- All routes are relative to root

### 2. Docker Compose

The CMS service is already configured in `docker-compose.yml`:

```yaml
cms:
  build:
    context: ./cms
    dockerfile: Dockerfile
    args:
      - VITE_API_BASE_URL=https://api.staging.stackyn.com
  labels:
    - "traefik.http.routers.cms-v2.rule=Host(`admin.staging.stackyn.com`)"
    - "traefik.http.routers.cms-v2.tls=true"
    - "traefik.http.routers.cms-v2.tls.certresolver=letsencrypt"
```

### 3. Traefik Routing

Traefik routes requests to the CMS:
- **Host**: `admin.staging.stackyn.com` → CMS container
- **Path**: `/*` (root)
- **SSL**: Automatic SSL via Let's Encrypt

## Deployment Steps

1. **Build and start services**:
   ```bash
   docker compose up -d --build cms
   ```

2. **Verify CMS is accessible**:
   - Open: https://admin.staging.stackyn.com
   - Should see the login page

3. **Create admin user** (if not already done):
   ```sql
   UPDATE users SET is_admin = true WHERE email = 'admin@example.com';
   ```

## Development

For local development:
```bash
cd cms
npm install
npm run dev
```

Access at: http://localhost:5174

## Troubleshooting

### CMS not loading
1. Check if CMS container is running: `docker ps | grep cms`
2. Check Traefik logs: `docker logs stackyn-traefik`
3. Check CMS logs: `docker logs stackyn-cms`
4. Verify Traefik routing: Check that `/cms` path is configured

### 404 errors on routes
- Ensure `base: '/'` is set in `vite.config.ts`
- Ensure `basename="/"` is set in `App.tsx`
- Rebuild the Docker image after changes

### API errors
- Verify `VITE_API_BASE_URL` is set correctly in Docker build args
- Check that backend API is accessible at the configured URL
- Verify admin user has `is_admin = true` in database

