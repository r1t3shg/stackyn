#!/bin/bash
# Quick script to rebuild frontend with new OTP signup UI

echo "🔄 Rebuilding Frontend with OTP Signup UI"
echo "=========================================="
echo ""

# Check if docker-compose is available
if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
    echo "❌ docker-compose not found"
    exit 1
fi

echo "1. Stopping frontend container..."
docker compose stop frontend

echo ""
echo "2. Removing old frontend container and image..."
docker compose rm -f frontend
docker rmi stackyn-frontend 2>/dev/null || true

echo ""
echo "3. Rebuilding frontend (this may take a few minutes)..."
docker compose build frontend --no-cache

echo ""
echo "4. Starting frontend..."
docker compose up -d frontend

echo ""
echo "5. Checking frontend logs..."
sleep 2
docker compose logs frontend --tail=20

echo ""
echo "✅ Frontend rebuild complete!"
echo ""
echo "📋 Next steps:"
echo "   1. Clear your browser cache (Ctrl+Shift+R or Cmd+Shift+R)"
echo "   2. Visit: https://staging.stackyn.com/signup"
echo "   3. You should see the new OTP signup UI (email input, not password)"
echo ""
echo "🔍 If you still see the old UI:"
echo "   - Try incognito/private mode"
echo "   - Check browser console for errors (F12)"
echo "   - Verify: docker compose logs frontend"

