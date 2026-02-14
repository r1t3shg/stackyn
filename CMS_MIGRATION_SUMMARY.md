# CMS Migration: staging.stackyn.com/cms → admin.staging.stackyn.com

## Summary
The CMS has been moved from a path-based route (`/cms`) on the main domain to a dedicated subdomain (`admin.staging.stackyn.com`).

## Changes Made

### 1. Docker Configuration (`docker-compose.yml`)
- **CMS Service**:
  - Updated Traefik router rules to match `Host(admin.staging.stackyn.com)`.
  - Removed `PathPrefix(/cms)` related rules and middlewares.
  - Updated localhost rule to `Host(admin.localhost)` for local development.
  - Exposed port `3001` for direct access.
- **Frontend Service**:
  - Removed `!PathPrefix(/cms)` exclusion since CMS is now on a different domain.

### 2. CMS Application Configuration
- **`cms/vite.config.ts`**:
  - Changed `base` from `/cms/` to `/` (root).
- **`cms/src/App.tsx`**:
  - Changed `BrowserRouter` `basename` from `/cms` to `/`.
- **`cms/src/components/Layout.tsx`**:
  - Updated logout redirect to `/login` (was `window.location.href = '/cms/login'`).
- **`cms/src/lib/api.ts`**:
  - Updated unauthorized redirect to `/login` (was `window.location.href = '/cms/login'`).

### 3. Build Configuration
- **`cms/Dockerfile`**:
  - Removed workaround that created `/cms` directory structure for build artifacts.

### 4. Documentation
- Updated `cms/README.md` and `cms/DEPLOYMENT.md` with new URL and configuration details.

## Next Steps for Deployment

1. **Update DNS Records**:
   - Create a new A record for `admin.staging.stackyn.com` pointing to your server IP.
   - Or Ensure wildcard `*.staging.stackyn.com` points to your server IP.

2. **Rebuild and Restart**:
   ```bash
   # Rebuild CMS to apply base path change
   docker compose build cms

   # Restart services to apply Traefik configuration
   docker compose up -d
   ```

3. **Local Development**:
   - Add `127.0.0.1 admin.localhost` to your hosts file.
   - Access CMS at `http://admin.localhost` (via Traefik) or `http://localhost:3001` (direct).
