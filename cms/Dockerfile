# CMS Dockerfile - React/Vite Application
# Multi-stage build for optimized production image
#
# Build Instructions:
#   docker build --build-arg VITE_API_BASE_URL=http://localhost:8080 -t stackyn-cms .
#
# Deployment Notes:
#   - VITE_API_BASE_URL must be set at build time (not runtime)
#   - The build process compiles React/TypeScript to static files
#   - Production build is served using 'serve' static file server
#   - SPA routing is supported via serve's -s flag
#   - Port 3001 is exposed (different from main frontend on 3000)

# Build stage
FROM node:20-alpine AS builder

WORKDIR /app

# Copy package files
COPY package*.json ./

# Install dependencies
RUN npm ci

# Copy source code
COPY . .

# Build the application
# Note: VITE_API_BASE_URL should be set at build time via build arg
# This is baked into the JavaScript bundle and cannot be changed at runtime
ARG VITE_API_BASE_URL=http://localhost:8080
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
EXPOSE 3001

# Start serve with SPA routing support
# Note: Vite builds with base: '/cms/' so assets are at /cms/assets/...
# Traefik strips /cms prefix before forwarding, so we serve from root
# The -s flag enables SPA mode (serve index.html for all routes)
# The --cors flag allows cross-origin requests if needed
CMD ["sh", "-c", "serve -s dist -l 3001 --no-clipboard"]
