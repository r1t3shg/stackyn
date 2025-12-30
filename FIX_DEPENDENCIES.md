# Fix Go 1.24.0 Dependency Issue

The dependencies `golang-migrate` and `golang.org/x/crypto` require Go 1.24.0 which doesn't exist. Here's how to fix it:

## Quick Fix (Run on VPS)

```bash
cd /opt/stackyn/server

# Backup go.mod
cp go.mod go.mod.backup

# Downgrade problematic dependencies
sed -i 's/github.com\/golang-migrate\/migrate\/v4 v4.19.1/github.com\/golang-migrate\/migrate\/v4 v4.18.0/' go.mod
sed -i 's/golang.org\/x\/crypto v0.46.0/golang.org\/x\/crypto v0.23.0/' go.mod

# Update dependencies
go mod tidy
go mod download

# Rebuild
cd /opt/stackyn
docker-compose up -d --build
```

## Manual Fix

Edit `go.mod` manually:

```bash
cd /opt/stackyn/server
nano go.mod
```

Find and change these lines:

**Change:**
```
github.com/golang-migrate/migrate/v4 v4.19.1
```
**To:**
```
github.com/golang-migrate/migrate/v4 v4.18.0
```

**Change:**
```
golang.org/x/crypto v0.46.0
```
**To:**
```
golang.org/x/crypto v0.23.0
```

Then run:
```bash
go mod tidy
go mod download
cd /opt/stackyn
docker-compose up -d --build
```

## Verify Fix

After running the fix, verify the changes:

```bash
cd /opt/stackyn/server
grep -E "(golang-migrate|crypto)" go.mod
```

You should see:
- `github.com/golang-migrate/migrate/v4 v4.18.0`
- `golang.org/x/crypto v0.23.0`

