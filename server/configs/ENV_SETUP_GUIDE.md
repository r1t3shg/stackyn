# Environment Variables Setup Guide

This guide explains what values you need to set in your `.env` file.

## 🔴 REQUIRED - Must Change These

### 1. JWT_SECRET (CRITICAL - Security)
**You MUST generate a secure random string for this!**

```bash
# Generate a secure JWT secret (run this command):
openssl rand -base64 32

# Or use this alternative:
openssl rand -hex 32
```

**Example:**
```env
JWT_SECRET=K8j2mN9pQ4rT7vW0xY3zA6bC9dE2fG5hI8jK1lM4nO7pQ0sT3uV6wX9yZ2aB5cD8eF
```

**Why:** This is used to sign and verify JWT tokens. If compromised, attackers can create valid tokens.

---

### 2. POSTGRES_PASSWORD
**Set this to the password you created for your PostgreSQL user.**

```env
POSTGRES_PASSWORD=your_actual_secure_password_here
```

**How to set it:**
```bash
# If you haven't created the user yet:
sudo -u postgres psql
CREATE USER stackyn_user WITH PASSWORD 'your_secure_password_here';
CREATE DATABASE stackyn;
GRANT ALL PRIVILEGES ON DATABASE stackyn TO stackyn_user;
\q
```

**Why:** Required to connect to your PostgreSQL database.

---

## 🟡 RECOMMENDED - Should Change These

### 3. REDIS_PASSWORD (Optional but Recommended)
**Set a password for Redis if you want security.**

```env
REDIS_PASSWORD=your_redis_password_here
```

**How to set it:**
```bash
# Edit Redis config
sudo nano /etc/redis/redis.conf

# Find and uncomment this line, set your password:
requirepass your_redis_password_here

# Restart Redis
sudo systemctl restart redis-server
```

**Why:** Prevents unauthorized access to your Redis instance.

---

### 4. POSTGRES_USER (If Different)
**If you created a different PostgreSQL user:**

```env
POSTGRES_USER=stackyn_user  # or whatever you named it
```

**Default:** `postgres` (works for local development)

---

## 🟢 OPTIONAL - Only Change If Needed

### 5. SERVER_PORT
**Only change if port 8080 is already in use:**

```env
SERVER_PORT=8080  # Default is fine
# Or use: 3000, 5000, 8000, etc.
```

---

### 6. POSTGRES_HOST
**Only change if PostgreSQL is on a different server:**

```env
POSTGRES_HOST=localhost  # Default is fine for single VPS
# Or: 192.168.1.100, db.example.com, etc.
```

---

### 7. REDIS_HOST
**Only change if Redis is on a different server:**

```env
REDIS_HOST=localhost  # Default is fine for single VPS
# Or: 192.168.1.100, redis.example.com, etc.
```

---

### 8. LOG_LEVEL
**Adjust based on your needs:**

```env
LOG_LEVEL=info     # Production (default)
LOG_LEVEL=debug    # Development (more verbose)
LOG_LEVEL=warn     # Only warnings and errors
LOG_LEVEL=error    # Only errors
```

---

### 9. WORKER_CONCURRENCY
**Adjust based on your VPS resources:**

```env
WORKER_CONCURRENCY=10  # Default (good for 2-4 CPU cores)
# Lower for less CPU: 5
# Higher for more CPU: 20
```

---

## ✅ Can Leave As-Is (Defaults Work)

These don't need to be changed for a standard single-VPS deployment:

- `SERVER_ADDR=0.0.0.0` ✅
- `POSTGRES_PORT=5432` ✅
- `POSTGRES_DATABASE=stackyn` ✅
- `POSTGRES_SSLMODE=disable` ✅ (for local connections)
- `REDIS_PORT=6379` ✅
- `REDIS_DB=0` ✅
- `DOCKER_HOST=unix:///var/run/docker.sock` ✅
- `DOCKER_API_VERSION=1.43` ✅
- `DOCKER_TLS_ENABLED=false` ✅
- `TRAEFIK_API_URL=http://localhost:8080` ✅
- `TRAEFIK_ENTRY_POINT=web` ✅
- `TRAEFIK_NETWORK_NAME=traefik` ✅
- `JWT_EXPIRATION=3600` ✅ (1 hour)

---

## 📝 Complete Example .env File

Here's a complete example with all required values filled in:

```env
# Server Configuration
SERVER_ADDR=0.0.0.0
SERVER_PORT=8080

# Postgres Configuration
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=stackyn_user
POSTGRES_PASSWORD=MySecurePassword123!
POSTGRES_DATABASE=stackyn
POSTGRES_SSLMODE=disable

# Redis Configuration
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=MyRedisPassword456!
REDIS_DB=0

# Docker Configuration
DOCKER_HOST=unix:///var/run/docker.sock
DOCKER_API_VERSION=1.43
DOCKER_TLS_ENABLED=false
DOCKER_CERT_PATH=
DOCKER_KEY_PATH=
DOCKER_CA_PATH=

# Traefik Configuration
TRAEFIK_API_URL=http://localhost:8080
TRAEFIK_ENTRY_POINT=web
TRAEFIK_NETWORK_NAME=traefik

# JWT Configuration (REQUIRED - generate a secure random string)
JWT_SECRET=K8j2mN9pQ4rT7vW0xY3zA6bC9dE2fG5hI8jK1lM4nO7pQ0sT3uV6wX9yZ2aB5cD8eF
JWT_EXPIRATION=3600

# Logging Configuration
LOG_LEVEL=info

# Worker Configuration
WORKER_CONCURRENCY=10
```

---

## 🔒 Security Checklist

Before deploying to production:

- [ ] `JWT_SECRET` is a long, random string (32+ characters)
- [ ] `POSTGRES_PASSWORD` is strong (12+ characters, mixed case, numbers, symbols)
- [ ] `REDIS_PASSWORD` is set (if using Redis authentication)
- [ ] `.env` file has restricted permissions: `chmod 600 .env`
- [ ] `.env` file is in `.gitignore` (never commit it!)

---

## 🚀 Quick Setup Commands

```bash
# 1. Generate JWT secret
openssl rand -base64 32

# 2. Create .env file
cd server
cp configs/env.example .env

# 3. Edit .env file
nano .env
# Paste the generated JWT_SECRET
# Set POSTGRES_PASSWORD
# Set REDIS_PASSWORD (optional)

# 4. Secure the file
chmod 600 .env

# 5. Verify it works
go run ./cmd/api
# Should start without errors
```

---

## ❓ Troubleshooting

**Error: "missing required configuration: JWT_SECRET"**
- Solution: Generate and set `JWT_SECRET` in `.env`

**Error: "missing required configuration: POSTGRES_PASSWORD"**
- Solution: Set `POSTGRES_PASSWORD` to match your PostgreSQL user password

**Error: "connection refused" (PostgreSQL)**
- Solution: Check `POSTGRES_HOST` and `POSTGRES_PORT`, ensure PostgreSQL is running

**Error: "connection refused" (Redis)**
- Solution: Check `REDIS_HOST` and `REDIS_PORT`, ensure Redis is running

**Error: "Cannot connect to Docker daemon"**
- Solution: Ensure Docker is running and `DOCKER_HOST` is correct. Add user to docker group: `sudo usermod -aG docker $USER`

