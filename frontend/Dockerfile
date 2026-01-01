# Frontend Dockerfile - React/Vite Application
# Multi-stage build for optimized production image
#
# Build Instructions:
#   docker build --build-arg VITE_API_BASE_URL=https://api.staging.stackyn.com -t stackyn-frontend .
#
# Deployment Notes:
#   - VITE_API_BASE_URL must be set at build time (not runtime)
#   - The build process compiles React/TypeScript to static files
#   - Production build is served using 'serve' static file server
#   - SPA routing is supported via serve's -s flag
#
# Migration Notes:
#   - Migrated from Create React App to Vite for faster builds
#   - Build output is in /app/dist directory
#   - Port 3000 is exposed for Traefik routing

# Build stage
FROM node:20-alpine AS builder

WORKDIR /app

# Copy package files
COPY package*.json ./

# Install dependencies
# Note: Using npm install instead of npm ci because package-lock.json 
# needs to be updated with new dependencies. After updating the lock file
# locally, switch back to 'npm ci' for faster, reproducible builds.
RUN npm install

# Copy source code
COPY . .

# Build the application
# Note: VITE_API_BASE_URL should be set at build time via build arg
# This is baked into the JavaScript bundle and cannot be changed at runtime
ARG VITE_API_BASE_URL=https://api.staging.stackyn.com
ENV VITE_API_BASE_URL=${VITE_API_BASE_URL}
RUN npm run build

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
CMD ["serve", "-s", "dist", "-l", "3000"]
