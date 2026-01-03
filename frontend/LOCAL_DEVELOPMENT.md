# Local Development Setup

This guide explains how to set up local development for the Stackyn frontend.

## Quick Start

1. **Start the backend services** (PostgreSQL, Redis, API):
   ```bash
   docker compose up -d
   ```

2. **Create a local environment file** (optional):
   ```bash
   # In the frontend directory
   touch .env.local
   ```
   
   Add this to `.env.local`:
   ```
   # Leave empty to use Vite proxy (recommended)
   # Or set to http://localhost:8080 if you want to bypass the proxy
   VITE_API_BASE_URL=
   ```

3. **Install dependencies**:
   ```bash
   cd frontend
   npm install
   ```

4. **Start the development server**:
   ```bash
   npm run dev
   ```

5. **Access the application**:
   - Frontend: http://localhost:3000
   - API: http://localhost:8080

## How It Works

### Vite Proxy (Recommended)

When running `npm run dev`, Vite automatically proxies all `/api/*` and `/health` requests to `http://localhost:8080`. This means:

- ✅ No CORS issues
- ✅ No need to configure API URLs
- ✅ Works seamlessly with the local backend

The frontend will make requests like:
- `GET /api/apps` → proxied to → `http://localhost:8080/api/apps`
- `POST /api/auth/send-otp` → proxied to → `http://localhost:8080/api/auth/send-otp`

### Manual API URL (Alternative)

If you prefer to set an explicit API URL, create `.env.local` with:
```
VITE_API_BASE_URL=http://localhost:8080
```

This will bypass the proxy and make direct requests to the API.

## Backend Requirements

Make sure the following services are running:

1. **PostgreSQL** - Database
2. **Redis** - Cache/session store
3. **API Server** - Backend API (exposed on port 8080)

Check service status:
```bash
docker compose ps
```

Check API logs:
```bash
docker compose logs -f api
```

## Troubleshooting

### API Connection Issues

1. **Verify API is running**:
   ```bash
   curl http://localhost:8080/health
   ```

2. **Check API logs**:
   ```bash
   docker compose logs api
   ```

3. **Verify port 8080 is exposed**:
   ```bash
   docker compose ps
   # Should show: 0.0.0.0:8080->8080/tcp for api service
   ```

### CORS Errors

If you see CORS errors, make sure you're using the Vite proxy (leave `VITE_API_BASE_URL` empty in `.env.local`). The proxy handles CORS automatically.

### Environment Variables

- `.env.local` - Local development (not committed to git)
- `.env.production` - Production build (not committed to git)
- Environment variables prefixed with `VITE_` are available in the frontend code via `import.meta.env.VITE_*`

## Switching Between Local and Remote

### Local Development
- Use Vite proxy (default) or set `VITE_API_BASE_URL=http://localhost:8080` in `.env.local`
- Run `npm run dev`

### Production/Staging
- Set `VITE_API_BASE_URL=https://api.staging.stackyn.com` at build time
- Run `npm run build`

The API URL is determined at build time for production builds, but can be overridden at runtime in development mode.

