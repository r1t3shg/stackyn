# Domain Migration: dev.stackyn.com → stackyn.com

## Summary
All references to `dev.stackyn.com` have been updated to `stackyn.com` for production deployment.

## Changes Made

### 1. Backend (Server)
- **`server/internal/api/router.go`**
  - Updated CORS allowed origins from `dev.stackyn.com` to `stackyn.com`
  - Updated CORS allowed origins from `console.dev.stackyn.com` to `console.stackyn.com`
  - Updated subdomain check from `.dev.stackyn.com` to `.stackyn.com`

### 2. Environment Configuration
- **`env.example`**
  - `FRONTEND_API_URL`: `https://api.dev.stackyn.com` → `https://api.stackyn.com`
  - `FRONTEND_DOMAIN`: `dev.stackyn.com` → `stackyn.com`
  - `CONSOLE_DOMAIN`: `console.dev.stackyn.com` → `console.stackyn.com`
  - `API_DOMAIN`: `api.dev.stackyn.com` → `api.stackyn.com`
  - `APP_BASE_DOMAIN`: `dev.stackyn.com` → `stackyn.com`

### 3. Docker Configuration
- **`docker-compose.yml`**
  - Updated all Traefik router rules for:
    - API domain: `api.dev.stackyn.com` → `api.stackyn.com`
    - Frontend domain: `dev.stackyn.com` → `stackyn.com`
    - Console domain: `console.dev.stackyn.com` → `console.stackyn.com`
  - Updated `APP_BASE_DOMAIN` defaults in all workers (build-worker, deploy-worker, cleanup-worker)
  - Updated `VITE_API_BASE_URL` defaults for frontend and CMS services

### 4. Frontend Code
- **`frontend/src/App.tsx`**
  - Updated console subdomain check: `console.dev.stackyn.com` → `console.stackyn.com`

- **`frontend/src/pages/LandingPage.tsx`**
  - Updated all redirect URLs from `console.dev.stackyn.com` to `console.stackyn.com` (8 occurrences)

- **`frontend/src/pages/PrivacyPolicy.tsx`**
  - Updated all redirect URLs from `console.dev.stackyn.com` to `console.stackyn.com` (3 occurrences)

- **`frontend/src/pages/TermsOfService.tsx`**
  - Updated all redirect URLs from `console.dev.stackyn.com` to `console.stackyn.com` (3 occurrences)

### 5. Dockerfile
- **`frontend/Dockerfile`**
  - Updated default `VITE_API_BASE_URL` build arg: `https://api.dev.stackyn.com` → `https://api.stackyn.com`
  - Updated build instruction comment

## Domain Mapping

| Old Domain | New Domain |
|------------|------------|
| `dev.stackyn.com` | `stackyn.com` |
| `api.dev.stackyn.com` | `api.stackyn.com` |
| `console.dev.stackyn.com` | `console.stackyn.com` |
| `*.dev.stackyn.com` (apps) | `*.stackyn.com` (apps) |

## Next Steps for Deployment

1. **Update DNS Records**
   - Point `stackyn.com` → Your VPS IP
   - Point `api.stackyn.com` → Your VPS IP
   - Point `console.stackyn.com` → Your VPS IP
   - Point `*.stackyn.com` → Your VPS IP (for wildcard subdomains)

2. **Update Environment Variables**
   - Copy `env.example` to `.env` on your VPS
   - Verify all domain variables are set correctly:
     ```bash
     FRONTEND_DOMAIN=stackyn.com
     CONSOLE_DOMAIN=console.stackyn.com
     API_DOMAIN=api.stackyn.com
     APP_BASE_DOMAIN=stackyn.com
     FRONTEND_API_URL=https://api.stackyn.com
     ```

3. **Rebuild Frontend**
   - The frontend needs to be rebuilt with the new API URL:
     ```bash
     docker-compose build frontend
     ```

4. **Restart Services**
   - After updating DNS and environment variables:
     ```bash
     docker-compose down
     docker-compose up -d
     ```

5. **Verify SSL Certificates**
   - Traefik will automatically request new Let's Encrypt certificates for the new domains
   - Check Traefik logs to ensure certificates are issued successfully

6. **Test All Endpoints**
   - `https://stackyn.com` - Landing page
   - `https://console.stackyn.com` - Dashboard
   - `https://api.stackyn.com/health` - API health check
   - `https://api.stackyn.com/api/v1/apps` - API endpoints

## Notes

- **Documentation Files**: Some documentation files (like `diagnose-dev-domain.sh`, `debug-env-vars.sh`, etc.) still contain references to `dev.stackyn.com` for historical reference. These don't affect functionality.

- **Local Development**: Local development (localhost) is unaffected and will continue to work as before.

- **CORS**: The backend CORS configuration now allows requests from `stackyn.com` and all its subdomains.

