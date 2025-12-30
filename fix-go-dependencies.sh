#!/bin/bash
# Fix Go dependencies that require non-existent Go 1.24.0

set -e

cd /opt/stackyn/server

echo "🔧 Fixing Go dependencies..."
echo ""

# Backup go.mod
cp go.mod go.mod.backup
echo "✓ Backed up go.mod"

# Downgrade problematic dependencies
echo "Downgrading dependencies..."

# Update golang-migrate
sed -i 's/github.com\/golang-migrate\/migrate\/v4 v4.19.1/github.com\/golang-migrate\/migrate\/v4 v4.18.0/' go.mod

# Update golang.org/x/crypto
sed -i 's/golang.org\/x\/crypto v0.46.0/golang.org\/x\/crypto v0.23.0/' go.mod

echo "✓ Updated go.mod"

# Clean up and download dependencies
echo "Running go mod tidy..."
go mod tidy

echo "Running go mod download..."
go mod download

echo ""
echo "✅ Dependencies fixed!"
echo ""
echo "Now rebuild with:"
echo "  cd /opt/stackyn && docker-compose up -d --build"

