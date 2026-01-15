#!/bin/bash

# Debug script for environment variables issue
# Usage: ./debug-env-vars.sh

echo "=========================================="
echo "Environment Variables Debug Script"
echo "=========================================="
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Get database password from environment or use default
DB_PASSWORD=${POSTGRES_PASSWORD:-changeme}
APP_ID="90f6a191-295f-4311-b2d4-3074a1ad417b"

echo "1. Checking if PostgreSQL container is running..."
if docker ps | grep -q stackyn-postgres; then
    echo -e "${GREEN}✓ PostgreSQL container is running${NC}"
else
    echo -e "${RED}✗ PostgreSQL container is not running${NC}"
    echo "   Start it with: docker-compose up -d postgres"
    exit 1
fi
echo ""

echo "2. Checking if API container is running..."
if docker ps | grep -q stackyn-api; then
    echo -e "${GREEN}✓ API container is running${NC}"
else
    echo -e "${RED}✗ API container is not running${NC}"
    echo "   Start it with: docker-compose up -d api"
fi
echo ""

echo "3. Checking if env_vars table exists..."
TABLE_EXISTS=$(docker exec stackyn-postgres psql -U stackyn_user -d stackyn -tAc "SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'env_vars');")
if [ "$TABLE_EXISTS" = "t" ]; then
    echo -e "${GREEN}✓ env_vars table exists${NC}"
else
    echo -e "${RED}✗ env_vars table does NOT exist${NC}"
    echo "   Run migrations or check database initialization"
    exit 1
fi
echo ""

echo "4. Checking table structure..."
echo "   Table schema:"
docker exec stackyn-postgres psql -U stackyn_user -d stackyn -c "\d env_vars"
echo ""

echo "5. Checking if app exists in database..."
APP_EXISTS=$(docker exec stackyn-postgres psql -U stackyn_user -d stackyn -tAc "SELECT EXISTS (SELECT 1 FROM apps WHERE id = '$APP_ID');")
if [ "$APP_EXISTS" = "t" ]; then
    echo -e "${GREEN}✓ App $APP_ID exists in database${NC}"
    echo "   App details:"
    docker exec stackyn-postgres psql -U stackyn_user -d stackyn -c "SELECT id, name, user_id, created_at FROM apps WHERE id = '$APP_ID';"
else
    echo -e "${RED}✗ App $APP_ID does NOT exist in database${NC}"
    echo "   Available apps:"
    docker exec stackyn-postgres psql -U stackyn_user -d stackyn -c "SELECT id, name, user_id FROM apps LIMIT 10;"
fi
echo ""

echo "6. Checking existing environment variables for this app..."
ENV_COUNT=$(docker exec stackyn-postgres psql -U stackyn_user -d stackyn -tAc "SELECT COUNT(*) FROM env_vars WHERE app_id = '$APP_ID';")
echo "   Found $ENV_COUNT environment variable(s) for this app"
if [ "$ENV_COUNT" -gt 0 ]; then
    echo "   Existing variables:"
    docker exec stackyn-postgres psql -U stackyn_user -d stackyn -c "SELECT key, LEFT(value, 50) as value_preview, created_at FROM env_vars WHERE app_id = '$APP_ID';"
fi
echo ""

echo "7. Checking for foreign key constraints..."
echo "   Checking if app_id foreign key constraint exists:"
docker exec stackyn-postgres psql -U stackyn_user -d stackyn -c "SELECT conname, contype FROM pg_constraint WHERE conrelid = 'env_vars'::regclass AND contype = 'f';"
echo ""

echo "8. Testing direct database insert (dry run)..."
echo "   Attempting to insert test env var (will rollback):"
docker exec stackyn-postgres psql -U stackyn_user -d stackyn <<EOF
BEGIN;
INSERT INTO env_vars (app_id, key, value) 
VALUES ('$APP_ID', 'TEST_KEY_DEBUG', 'test_value')
ON CONFLICT (app_id, key) 
DO UPDATE SET value = EXCLUDED.value, updated_at = NOW();
ROLLBACK;
EOF
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Direct database insert test passed${NC}"
else
    echo -e "${RED}✗ Direct database insert test failed${NC}"
    echo "   Check the error message above"
fi
echo ""

echo "9. Checking API container logs for recent errors..."
echo "   Last 20 lines of API logs:"
docker logs stackyn-api --tail 20 2>&1 | grep -i -E "(error|env|envvar|env_var|500)" || echo "   No recent errors found in logs"
echo ""

echo "10. Checking API container health..."
API_HEALTH=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/health 2>/dev/null || echo "000")
if [ "$API_HEALTH" = "200" ]; then
    echo -e "${GREEN}✓ API health check passed (HTTP $API_HEALTH)${NC}"
else
    echo -e "${YELLOW}⚠ API health check returned HTTP $API_HEALTH${NC}"
    echo "   If using production domain, try: curl https://api.dev.stackyn.com/health"
fi
echo ""

echo "11. Testing API endpoint directly (requires auth token)..."
echo "   To test manually, run:"
echo "   curl -X POST https://api.dev.stackyn.com/api/v1/apps/$APP_ID/env \\"
echo "     -H 'Content-Type: application/json' \\"
echo "     -H 'Authorization: Bearer YOUR_TOKEN' \\"
echo "     -d '{\"key\":\"TEST_KEY\",\"value\":\"test_value\"}'"
echo ""

echo "12. Checking database connection from API container..."
docker exec stackyn-api sh -c 'psql -h postgres -U stackyn_user -d stackyn -c "SELECT 1;"' 2>&1 | head -5
echo ""

echo "=========================================="
echo "Debug Summary"
echo "=========================================="
echo "If all checks passed, the issue might be:"
echo "  - Authentication/authorization issue"
echo "  - Request format issue"
echo "  - Network/CORS issue"
echo ""
echo "Next steps:"
echo "  1. Check browser console for detailed error messages"
echo "  2. Check API logs: docker logs -f stackyn-api"
echo "  3. Try the API endpoint with curl (see step 11)"
echo "  4. Verify your auth token is valid"
echo ""

