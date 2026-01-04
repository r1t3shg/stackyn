# Frontend Dockerfile - React/Vite Application
# Optimized multi-stage build for fast production images
#
# Build Instructions (with BuildKit for faster builds):
#   DOCKER_BUILDKIT=1 docker build --build-arg VITE_API_BASE_URL=https://api.staging.stackyn.com -t stackyn-frontend .
#
# Deployment Notes:
#   - VITE_API_BASE_URL must be set at build time (not runtime)
#   - The build process compiles React/TypeScript to static files
#   - Production build is served using 'serve' static file server
#   - SPA routing is supported via serve's -s flag
#
# Performance Optimizations:
#   - Uses npm ci for faster, reproducible dependency installation
#   - BuildKit cache mounts for node_modules (reuses cache between builds)
#   - Separate layers for dependencies and source code
#   - Minimal production image

# Build stage
FROM node:20-alpine AS builder

WORKDIR /app

# Copy package files first (for better layer caching)
COPY package*.json ./

# Install dependencies with cache mount (BuildKit feature)
# This caches node_modules between builds, making rebuilds much faster
RUN --mount=type=cache,target=/root/.npm \
    npm ci --only=production=false

# Copy source code (this layer only rebuilds when source changes)
COPY . .

# Build the application
# Note: VITE_API_BASE_URL should be set at build time via build arg
# This is baked into the JavaScript bundle and cannot be changed at runtime
ARG VITE_API_BASE_URL=https://api.staging.stackyn.com
ENV VITE_API_BASE_URL=${VITE_API_BASE_URL}

# Build with Vite
# Using build:fast to skip TypeScript type checking for faster builds
# Type checking is done in CI/CD, so we can skip it here for speed
# For production with full type checking, use: npm run build
RUN npm run build:fast

# Production stage with simple static server
FROM node:20-alpine

WORKDIR /app

# Install serve globally for serving static files
RUN npm install -g serve

# Copy built files from builder
COPY --from=builder /app/dist ./dist

# Expose port
EXPOSE 3000

# Start serve with SPA routing support
# Listen on 0.0.0.0 to allow Traefik to connect
CMD ["serve", "-s", "dist", "-l", "3000"]
