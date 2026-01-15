# Quick Start - Deploy Everything with Docker Compose

Deploy the entire Stackyn platform with one command!

## Prerequisites

- Docker and Docker Compose installed
- At least 4GB RAM, 2 CPU cores, 20GB disk space

## Step 1: Setup Environment Variables

```bash
# Copy example env file
cp env.example .env

# Edit .env file
nano .env
```

**Required:**
- `JWT_SECRET` - Generate with: `openssl rand -base64 32`
- `POSTGRES_PASSWORD` - Set a secure password

**Optional:**
- `FRONTEND_API_URL` - API URL for frontend (default: `http://localhost:8080`)
- `FRONTEND_DOMAIN` - Domain for frontend (default: `localhost`)
- `API_DOMAIN` - Domain for API (default: `api.localhost`)

## Step 2: Deploy Everything

```bash
docker-compose up -d
```

That's it! 🎉

## Step 3: Access Services

- **Frontend:** http://localhost:3000
- **API:** http://localhost:8080
- **Traefik Dashboard:** http://localhost:8081 (Traefik admin)

## Step 4: Run Database Migrations

```bash
# Wait for services to be ready
docker-compose ps

# Run migrations
docker-compose exec api /app/api migrate
# OR manually if you have migrate command
docker-compose exec postgres psql -U stackyn_user -d stackyn -f /path/to/migrations
```

## Useful Commands

```bash
# View logs
docker-compose logs -f

# View specific service logs
docker-compose logs -f api
docker-compose logs -f frontend

# Stop everything
docker-compose down

# Stop and remove volumes (⚠️ deletes data)
docker-compose down -v

# Restart a service
docker-compose restart api

# Rebuild after code changes
docker-compose up -d --build
```

## Production Deployment

For production, update `.env`:

```env
# Use your actual domain
FRONTEND_DOMAIN=yourdomain.com
API_DOMAIN=api.yourdomain.com
FRONTEND_API_URL=https://api.yourdomain.com

# Strong passwords
POSTGRES_PASSWORD=your_secure_password
REDIS_PASSWORD=your_redis_password
JWT_SECRET=your_generated_jwt_secret
```

Then deploy:
```bash
docker-compose up -d
```

## Troubleshooting

### Services won't start
```bash
# Check logs
docker-compose logs

# Check if ports are in use
sudo lsof -i :3000
sudo lsof -i :8080
sudo lsof -i :5432
```

### Database connection errors
```bash
# Wait for postgres to be healthy
docker-compose ps postgres

# Check postgres logs
docker-compose logs postgres
```

### Frontend can't connect to API
- Make sure `FRONTEND_API_URL` in `.env` matches your API URL
- Rebuild frontend: `docker-compose up -d --build frontend`

