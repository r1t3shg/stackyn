#!/bin/bash

# Verify Billing Environment Variables
# Run this on your server

echo "=========================================="
echo "Verifying Billing Environment Variables"
echo "=========================================="
echo ""

cd /opt/stackyn || { echo "❌ Not in /opt/stackyn directory"; exit 1; }

echo "1. Checking .env file..."
if [ ! -f .env ]; then
    echo "❌ .env file not found!"
    exit 1
fi

echo "✓ .env file exists"
echo ""

echo "2. Checking if LEMON_API_KEY is in .env file..."
if grep -q "^LEMON_API_KEY=" .env; then
    LEMON_API_KEY_VALUE=$(grep "^LEMON_API_KEY=" .env | cut -d'=' -f2-)
    if [ -z "$LEMON_API_KEY_VALUE" ] || [ "$LEMON_API_KEY_VALUE" = "your_lemon_squeezy_api_key_here" ]; then
        echo "❌ LEMON_API_KEY is empty or placeholder"
    else
        echo "✓ LEMON_API_KEY is set (${#LEMON_API_KEY_VALUE} characters)"
    fi
else
    echo "❌ LEMON_API_KEY not found in .env file"
fi
echo ""

echo "3. Checking if LEMON_STORE_ID is in .env file..."
if grep -q "^LEMON_STORE_ID=" .env; then
    LEMON_STORE_ID_VALUE=$(grep "^LEMON_STORE_ID=" .env | cut -d'=' -f2-)
    if [ -z "$LEMON_STORE_ID_VALUE" ] || [ "$LEMON_STORE_ID_VALUE" = "your_lemon_squeezy_store_id_here" ]; then
        echo "❌ LEMON_STORE_ID is empty or placeholder"
    else
        echo "✓ LEMON_STORE_ID is set: $LEMON_STORE_ID_VALUE"
    fi
else
    echo "❌ LEMON_STORE_ID not found in .env file"
fi
echo ""

echo "4. Checking if FRONTEND_BASE_URL is in .env file..."
if grep -q "^FRONTEND_BASE_URL=" .env; then
    FRONTEND_BASE_URL_VALUE=$(grep "^FRONTEND_BASE_URL=" .env | cut -d'=' -f2-)
    if [ -z "$FRONTEND_BASE_URL_VALUE" ]; then
        echo "❌ FRONTEND_BASE_URL is empty"
    else
        echo "✓ FRONTEND_BASE_URL is set: $FRONTEND_BASE_URL_VALUE"
    fi
else
    echo "❌ FRONTEND_BASE_URL not found in .env file"
fi
echo ""

echo "5. Checking API container environment..."
if docker-compose ps api | grep -q "Up"; then
    echo "API container is running"
    echo ""
    echo "Checking environment variables in API container:"
    
    LEMON_API_KEY_IN_CONTAINER=$(docker-compose exec -T api env 2>/dev/null | grep "^LEMON_API_KEY=" | cut -d'=' -f2- || echo "")
    LEMON_STORE_ID_IN_CONTAINER=$(docker-compose exec -T api env 2>/dev/null | grep "^LEMON_STORE_ID=" | cut -d'=' -f2- || echo "")
    FRONTEND_BASE_URL_IN_CONTAINER=$(docker-compose exec -T api env 2>/dev/null | grep "^FRONTEND_BASE_URL=" | cut -d'=' -f2- || echo "")
    
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
echo "=========================================="
echo "Next Steps:"
echo "=========================================="
echo ""
echo "If variables are missing in .env:"
echo "1. Edit /opt/stackyn/.env"
echo "2. Add the required variables"
echo ""
echo "If variables are in .env but not in container:"
echo "1. Restart API container: docker-compose restart api"
echo "2. Or rebuild: docker-compose up -d --force-recreate api"
echo ""

