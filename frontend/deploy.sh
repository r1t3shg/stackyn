#!/bin/bash

# Deployment script for Stackyn Frontend (Vite)
# Usage: ./deploy.sh

set -e

echo "🚀 Starting deployment..."

# Check if Node.js is installed
if ! command -v node &> /dev/null; then
    echo "❌ Node.js is not installed. Please install Node.js 20+ first."
    exit 1
fi

# Check Node.js version
NODE_VERSION=$(node -v | cut -d'v' -f2 | cut -d'.' -f1)
if [ "$NODE_VERSION" -lt 20 ]; then
    echo "❌ Node.js version 20+ is required. Current version: $(node -v)"
    exit 1
fi

# Set environment variable
export VITE_API_BASE_URL=https://staging.stackyn.com

# Install dependencies
echo "📦 Installing dependencies..."
npm ci

# Build the application
echo "🔨 Building application..."
npm run build

echo "✅ Build completed!"
echo ""
echo "📝 To serve the application:"
echo "   - For development: npm run dev"
echo "   - For production preview: npm run preview"
echo "   - For production: Use nginx or another static file server to serve the 'dist' directory"
