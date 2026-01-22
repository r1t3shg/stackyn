#!/bin/bash

# Check Billing Configuration
# Run this on your server to verify billing environment variables are set

echo "=========================================="
echo "Checking Billing Configuration"
echo "=========================================="
echo ""

cd /opt/stackyn || { echo "❌ Not in /opt/stackyn directory"; exit 1; }

# Load environment variables
if [ ! -f .env ]; then
    echo "❌ .env file not found!"
    exit 1
fi

source .env

echo "1. Checking required environment variables..."
echo ""

# Check LEMON_API_KEY
if [ -z "$LEMON_API_KEY" ] || [ "$LEMON_API_KEY" = "your_lemon_squeezy_api_key_here" ]; then
    echo "❌ LEMON_API_KEY is not set or is placeholder"
else
    echo "✓ LEMON_API_KEY is set (${#LEMON_API_KEY} characters)"
fi

# Check LEMON_STORE_ID
if [ -z "$LEMON_STORE_ID" ] || [ "$LEMON_STORE_ID" = "your_lemon_squeezy_store_id_here" ]; then
    echo "❌ LEMON_STORE_ID is not set or is placeholder"
else
    echo "✓ LEMON_STORE_ID is set: $LEMON_STORE_ID"
fi

# Check FRONTEND_BASE_URL
if [ -z "$FRONTEND_BASE_URL" ]; then
    echo "❌ FRONTEND_BASE_URL is not set"
else
    echo "✓ FRONTEND_BASE_URL is set: $FRONTEND_BASE_URL"
fi

echo ""
echo "2. Checking API container environment..."
echo ""

# Check if API container has the variables
if docker-compose ps api | grep -q "Up"; then
    echo "Checking environment variables in API container..."
    
    LEMON_API_KEY_IN_CONTAINER=$(docker-compose exec -T api env | grep LEMON_API_KEY | cut -d'=' -f2 || echo "")
    LEMON_STORE_ID_IN_CONTAINER=$(docker-compose exec -T api env | grep LEMON_STORE_ID | cut -d'=' -f2 || echo "")
    FRONTEND_BASE_URL_IN_CONTAINER=$(docker-compose exec -T api env | grep FRONTEND_BASE_URL | cut -d'=' -f2 || echo "")
    
    if [ -z "$LEMON_API_KEY_IN_CONTAINER" ]; then
        echo "❌ LEMON_API_KEY not found in API container"
    else
        echo "✓ LEMON_API_KEY found in API container (${#LEMON_API_KEY_IN_CONTAINER} characters)"
    fi
    
    if [ -z "$LEMON_STORE_ID_IN_CONTAINER" ]; then
        echo "❌ LEMON_STORE_ID not found in API container"
    else
        echo "✓ LEMON_STORE_ID found in API container: $LEMON_STORE_ID_IN_CONTAINER"
    fi
    
    if [ -z "$FRONTEND_BASE_URL_IN_CONTAINER" ]; then
        echo "❌ FRONTEND_BASE_URL not found in API container"
    else
        echo "✓ FRONTEND_BASE_URL found in API container: $FRONTEND_BASE_URL_IN_CONTAINER"
    fi
else
    echo "⚠️  API container is not running"
fi

echo ""
echo "3. Checking API logs for billing errors..."
echo ""

docker-compose logs api --tail=50 | grep -i -E "(billing|lemon|checkout|not configured)" | tail -10 || echo "No billing-related errors found in recent logs"

echo ""
echo "=========================================="
echo "Next Steps:"
echo "=========================================="
echo ""
echo "If variables are missing in .env file:"
echo "1. Edit /opt/stackyn/.env"
echo "2. Add the required variables:"
echo "   LEMON_API_KEY=your_key_here"
echo "   LEMON_STORE_ID=your_store_id_here"
echo "   FRONTEND_BASE_URL=https://console.staging.stackyn.com"
echo ""
echo "If variables are in .env but not in container:"
echo "1. Restart API container: docker-compose restart api"
echo "2. Wait 10 seconds, then test again"
echo ""

