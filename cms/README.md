# Stackyn CMS - Internal Admin Panel

Internal CMS (admin panel) for Stackyn that uses backend APIs only. No Firebase SDK, no direct database access.

## Features

- **Users Management**: List users, view details, change plans, suspend/activate
- **Apps Management**: List apps, stop/start, trigger redeploy
- **Plans & Quotas**: View plan limits and usage statistics (read-only)

## Development Setup

1. Install dependencies:
```bash
npm install
```

2. Set environment variables (create `.env` file):
```bash
VITE_API_BASE_URL=http://localhost:8080
```

3. Start development server:
```bash
npm run dev
```

**Access URL**: http://localhost:5174

The CMS will run on port **5174** (different from the main frontend on port 3000).

**Note**: In production, the CMS is accessible at `https://admin.staging.stackyn.com`

## Production Build

1. Build for production:
```bash
npm run build
```

2. Preview production build:
```bash
npm run preview
```

## Docker Deployment

### Build Docker Image:
```bash
docker build --build-arg VITE_API_BASE_URL=https://api.staging.stackyn.com -t stackyn-cms .
```

### Run Container:
```bash
docker run -p 3001:3001 stackyn-cms
```

**Access URL**: http://localhost:3001

The CMS service is already configured in `docker-compose.yml` with Traefik routing.

**Access URL**: https://admin.staging.stackyn.com

The CMS is served at the root `/` path on the admin subdomain.

## Authentication

The CMS uses the same authentication system as the main backend. Admin users must have `is_admin = true` in the database.

To create an admin user, update the database:
```sql
UPDATE users SET is_admin = true WHERE email = 'admin@example.com';
```

## API Endpoints

All CMS APIs are under `/admin/*` and require:
1. Valid authentication token (Bearer token)
2. Admin role (`is_admin = true`)

### Users API
- `GET /admin/users` - List users (with pagination and search)
- `GET /admin/users/{id}` - Get user details
- `PATCH /admin/users/{id}/plan` - Update user plan

### Apps API
- `GET /admin/apps` - List apps (with pagination)
- `POST /admin/apps/{id}/stop` - Stop app
- `POST /admin/apps/{id}/start` - Start app
- `POST /admin/apps/{id}/redeploy` - Trigger redeploy

## Architecture

- **CMS → Backend APIs only**: No direct database access
- **Backend → handles all DB access**: All data operations go through backend
- **No Firebase SDK in CMS**: Uses standard REST APIs
- **CMS is internal-use only**: Admin role enforced server-side

## Ports

- **Development**: Port 5174 (Vite dev server)
- **Production**: Port 3001 (Docker container)
- **Main Frontend**: Port 3000 (separate service)
