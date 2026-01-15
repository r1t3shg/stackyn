# Docker Compose .env File Verification

## ✅ Yes, `cp env.example .env` will work on VPS!

Docker Compose automatically loads environment variables from a `.env` file in the same directory as `docker-compose.yml`.

## How Docker Compose Loads .env

1. **Automatic Loading**: Docker Compose automatically reads `.env` file from the same directory
2. **Variable Substitution**: Uses `${VARIABLE_NAME}` syntax in docker-compose.yml
3. **Default Values**: Supports `${VARIABLE:-default}` syntax for fallbacks

## Variables Used in docker-compose.yml

All these variables are correctly referenced and will be loaded from `.env`:

### Database & Redis
- ✅ `POSTGRES_PASSWORD` - Used in postgres, api, build-worker, deploy-worker
- ✅ `REDIS_PASSWORD` - Used in redis, api, build-worker, deploy-worker

### Authentication
- ✅ `JWT_SECRET` - Used in api, build-worker, deploy-worker, cleanup-worker

### Domain Configuration
- ✅ `API_DOMAIN` - Used in Traefik labels for API routing
- ✅ `FRONTEND_DOMAIN` - Used in Traefik labels for frontend routing
- ✅ `CONSOLE_DOMAIN` - Used in Traefik labels for console routing
- ✅ `FRONTEND_API_URL` - Used in frontend build args
- ✅ `APP_BASE_DOMAIN` - **NOW ADDED** to workers (deploy-worker, build-worker, cleanup-worker)

### SSL & Email
- ✅ `ACME_EMAIL` - Used in Traefik for Let's Encrypt
- ✅ `RESEND_API_KEY` - Used in API service
- ✅ `EMAIL_FROM_EMAIL` - Used in API service

## Verification Steps

On your VPS, after running `cp env.example .env`:

```bash
# 1. Verify .env file exists
ls -la .env

# 2. Check if Docker Compose can read variables
docker compose config | grep -E "POSTGRES_PASSWORD|JWT_SECRET|APP_BASE_DOMAIN"

# 3. Verify specific variable substitution
docker compose config | grep "APP_BASE_DOMAIN"
```

## Important Notes

1. **File Location**: `.env` must be in the same directory as `docker-compose.yml`
   - ✅ Correct: `/opt/stackyn/.env` (same dir as docker-compose.yml)
   - ❌ Wrong: `/opt/stackyn/server/.env` (different directory)

2. **File Permissions**: Secure the file
   ```bash
   chmod 600 .env
   ```

3. **Variable Format**: No spaces around `=`
   - ✅ Correct: `POSTGRES_PASSWORD=value`
   - ❌ Wrong: `POSTGRES_PASSWORD = value`

4. **Comments**: Lines starting with `#` are ignored

## Testing on VPS

```bash
cd /opt/stackyn

# Copy template
cp env.example .env

# Edit and set values
nano .env

# Verify Docker Compose can read it
docker compose config > /tmp/compose-config.txt
grep -i "password\|secret\|domain" /tmp/compose-config.txt

# If variables show correctly, you're good to go!
```

## What Was Fixed

I've added `APP_BASE_DOMAIN` to all workers that need it:
- ✅ `build-worker` - Needs it for docker-compose detection
- ✅ `deploy-worker` - **REQUIRED** for subdomain generation
- ✅ `cleanup-worker` - Added for consistency

This ensures the Go code can read `APP_BASE_DOMAIN` via `os.Getenv()`.

## Summary

✅ **Yes, `cp env.example .env` will work perfectly on VPS!**

Docker Compose will automatically:
1. Load all variables from `.env`
2. Substitute them in `${VARIABLE}` references
3. Pass them to containers as environment variables
4. Use default values if variables are missing (where specified)

Just make sure:
- `.env` is in the same directory as `docker-compose.yml`
- All required variables are set (no empty values for required ones)
- File permissions are secure (`chmod 600 .env`)

