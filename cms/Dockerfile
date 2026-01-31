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

# Production stage with Nginx
FROM nginx:alpine

# Remove default nginx static assets
RUN rm -rf /usr/share/nginx/html/*

# Copy built files from builder to /cms subdirectory to support prefix
# This ensures physical path matches the URL path /cms/...
COPY --from=builder /app/dist /usr/share/nginx/html/cms

# Copy custom nginx configuration
COPY nginx.conf /etc/nginx/conf.d/default.conf

# Expose port 80
EXPOSE 80

# Start Nginx
CMD ["nginx", "-g", "daemon off;"]

