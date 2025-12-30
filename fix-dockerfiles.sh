#!/bin/bash
# Fix Dockerfiles to handle Go 1.24.0 requirement

cd /opt/stackyn/server

echo "Fixing Dockerfiles to use GOTOOLCHAIN=auto..."

# Update Dockerfile.api
if [ -f Dockerfile.api ]; then
    sed -i '/RUN CGO_ENABLED=0 GOOS=linux go build/i ENV GOTOOLCHAIN=auto' Dockerfile.api
    echo "✓ Updated Dockerfile.api"
fi

# Update Dockerfile.build-worker
if [ -f Dockerfile.build-worker ]; then
    sed -i '/RUN CGO_ENABLED=0 GOOS=linux go build/i ENV GOTOOLCHAIN=auto' Dockerfile.build-worker
    echo "✓ Updated Dockerfile.build-worker"
fi

# Update Dockerfile.deploy-worker
if [ -f Dockerfile.deploy-worker ]; then
    sed -i '/RUN CGO_ENABLED=0 GOOS=linux go build/i ENV GOTOOLCHAIN=auto' Dockerfile.deploy-worker
    echo "✓ Updated Dockerfile.deploy-worker"
fi

# Update Dockerfile.cleanup-worker
if [ -f Dockerfile.cleanup-worker ]; then
    sed -i '/RUN CGO_ENABLED=0 GOOS=linux go build/i ENV GOTOOLCHAIN=auto' Dockerfile.cleanup-worker
    echo "✓ Updated Dockerfile.cleanup-worker"
fi

echo ""
echo "✅ All Dockerfiles updated!"
echo ""
echo "Now rebuild with:"
echo "  cd /opt/stackyn && docker-compose up -d --build"

