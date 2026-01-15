# CMS Deployment Guide

## Access URL

The CMS is accessible at: **https://staging.stackyn.com/cms**

## Configuration

### 1. Build Configuration

The CMS is configured to use `/cms/` as the base path:
- `vite.config.ts` has `base: '/cms/'`
- `App.tsx` uses `basename="/cms"` for React Router
- All routes are relative to `/cms`

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
    - "traefik.http.routers.cms-https.rule=Host(`staging.stackyn.com`) && PathPrefix(`/cms`)"
    - "traefik.http.routers.cms-https.middlewares=cms-stripprefix"
    - "traefik.http.middlewares.cms-stripprefix.stripprefix.prefixes=/cms"
```

### 3. Traefik Routing

Traefik routes requests to the CMS:
- **Path**: `/cms/*` → CMS container
- **Prefix stripping**: `/cms` is stripped before forwarding to the container
- **SSL**: Automatic SSL via Let's Encrypt

## Deployment Steps

1. **Build and start services**:
   ```bash
   docker compose up -d --build cms
   ```

2. **Verify CMS is accessible**:
   - Open: https://staging.stackyn.com/cms
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
- Ensure `base: '/cms/'` is set in `vite.config.ts`
- Ensure `basename="/cms"` is set in `App.tsx`
- Rebuild the Docker image after changes

### API errors
- Verify `VITE_API_BASE_URL` is set correctly in Docker build args
- Check that backend API is accessible at the configured URL
- Verify admin user has `is_admin = true` in database

