# Fix Go 1.24.0 Dependency Issue

The dependencies require Go 1.24.0 which doesn't exist. Here's how to fix it:

## Quick Fix (On Your VPS)

Run this command on your VPS:

```bash
cd /opt/stackyn/server

# Fix all Dockerfiles at once
for file in Dockerfile.*; do
    sed -i '/RUN CGO_ENABLED=0 GOOS=linux go build/i ENV GOTOOLCHAIN=auto' "$file"
done

# Rebuild
cd /opt/stackyn
docker-compose up -d --build
```

## Alternative: Downgrade Dependencies

If the above doesn't work, downgrade the problematic dependencies:

```bash
cd /opt/stackyn/server

# Edit go.mod
nano go.mod

# Change these lines:
# FROM: github.com/golang-migrate/migrate/v4 v4.19.1
# TO:   github.com/golang-migrate/migrate/v4 v4.18.0
#
# FROM: golang.org/x/crypto v0.46.0
# TO:   golang.org/x/crypto v0.23.0

# Then run:
go mod tidy
go mod download

# Rebuild
cd /opt/stackyn
docker-compose up -d --build
```

## Manual Fix

Edit each Dockerfile and add `ENV GOTOOLCHAIN=auto` before the build command:

**Before:**
```dockerfile
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/api ./cmd/api
```

**After:**
```dockerfile
ENV GOTOOLCHAIN=auto
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/api ./cmd/api
```

Apply this to all 4 Dockerfiles:
- `Dockerfile.api`
- `Dockerfile.build-worker`
- `Dockerfile.deploy-worker`
- `Dockerfile.cleanup-worker`

