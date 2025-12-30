#!/bin/bash
# One-command deployment script for Stackyn

set -e

echo "🚀 Stackyn Quick Deployment"
echo "=========================="
echo ""

# Check if .env exists
if [ ! -f .env ]; then
    echo "📝 Creating .env file from example..."
    cp env.example .env
    
    # Generate JWT secret
    if command -v openssl &> /dev/null; then
        JWT_SECRET=$(openssl rand -base64 32)
        # Update .env file with generated JWT secret
        if [[ "$OSTYPE" == "darwin"* ]]; then
            # macOS
            sed -i '' "s/your_jwt_secret_here_generate_with_openssl_rand_base64_32/$JWT_SECRET/" .env
        else
            # Linux
            sed -i "s/your_jwt_secret_here_generate_with_openssl_rand_base64_32/$JWT_SECRET/" .env
        fi
        echo "✅ Generated JWT_SECRET"
    else
        echo "⚠️  openssl not found. Please set JWT_SECRET manually in .env"
    fi
    
    echo ""
    echo "⚠️  IMPORTANT: Please edit .env and set:"
    echo "   - POSTGRES_PASSWORD (set a secure password)"
    echo "   - JWT_SECRET (already generated if openssl is available)"
    echo ""
    read -p "Press Enter after editing .env file..."
fi

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo "❌ Docker is not running. Please start Docker first."
    exit 1
fi

# Check if docker-compose is available
if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
    echo "❌ docker-compose not found. Please install Docker Compose."
    exit 1
fi

echo "🐳 Starting all services..."
echo ""

# Use docker compose (newer) or docker-compose (older)
if docker compose version &> /dev/null; then
    docker compose up -d --build
else
    docker-compose up -d --build
fi

echo ""
echo "✅ Deployment complete!"
echo ""
echo "📋 Services:"
echo "   - Frontend:  http://localhost:3000"
echo "   - API:       http://localhost:8080"
echo "   - Traefik:   http://localhost:8081"
echo ""
echo "📊 View logs: docker-compose logs -f"
echo "🛑 Stop:      docker-compose down"
echo ""

