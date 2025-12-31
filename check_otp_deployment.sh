#!/bin/bash
# Diagnostic script to check OTP signup deployment

echo "🔍 OTP Signup Deployment Diagnostic"
echo "===================================="
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if docker-compose is available
if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
    echo -e "${RED}❌ docker-compose not found${NC}"
    exit 1
fi

echo "1. Checking if services are running..."
if docker compose ps | grep -q "Up"; then
    echo -e "${GREEN}✅ Services are running${NC}"
else
    echo -e "${RED}❌ Some services are not running${NC}"
    docker compose ps
fi
echo ""

echo "2. Checking API service..."
if docker compose ps api | grep -q "Up"; then
    echo -e "${GREEN}✅ API service is running${NC}"
    
    # Check if email service file exists
    if docker compose exec -T api test -f /app/internal/services/email.go 2>/dev/null; then
        echo -e "${GREEN}✅ Email service file exists${NC}"
    else
        echo -e "${RED}❌ Email service file NOT found - code not deployed!${NC}"
    fi
    
    # Check if repositories file exists
    if docker compose exec -T api test -f /app/internal/api/repositories.go 2>/dev/null; then
        echo -e "${GREEN}✅ Repositories file exists${NC}"
    else
        echo -e "${RED}❌ Repositories file NOT found - code not deployed!${NC}"
    fi
else
    echo -e "${RED}❌ API service is not running${NC}"
fi
echo ""

echo "3. Checking environment variables..."
if docker compose exec -T api env | grep -q "EMAIL_RESEND_API_KEY"; then
    API_KEY=$(docker compose exec -T api env | grep "EMAIL_RESEND_API_KEY" | cut -d'=' -f2)
    if [ -n "$API_KEY" ] && [ "$API_KEY" != "" ]; then
        echo -e "${GREEN}✅ EMAIL_RESEND_API_KEY is set${NC}"
        echo "   Key: ${API_KEY:0:10}... (truncated)"
    else
        echo -e "${RED}❌ EMAIL_RESEND_API_KEY is empty${NC}"
    fi
else
    echo -e "${RED}❌ EMAIL_RESEND_API_KEY not found in environment${NC}"
fi

if docker compose exec -T api env | grep -q "EMAIL_FROM_EMAIL"; then
    FROM_EMAIL=$(docker compose exec -T api env | grep "EMAIL_FROM_EMAIL" | cut -d'=' -f2)
    echo -e "${GREEN}✅ EMAIL_FROM_EMAIL is set: $FROM_EMAIL${NC}"
else
    echo -e "${YELLOW}⚠️  EMAIL_FROM_EMAIL not set (using default)${NC}"
fi
echo ""

echo "4. Checking database (OTP table)..."
if docker compose exec -T postgres psql -U stackyn_user -d stackyn -c "\d otps" &>/dev/null; then
    echo -e "${GREEN}✅ OTP table exists${NC}"
    
    # Count OTPs
    COUNT=$(docker compose exec -T postgres psql -U stackyn_user -d stackyn -t -c "SELECT COUNT(*) FROM otps;" 2>/dev/null | tr -d ' ')
    echo "   Total OTPs in database: $COUNT"
else
    echo -e "${RED}❌ OTP table does NOT exist - migrations not run!${NC}"
    echo "   Run: docker compose exec postgres psql -U stackyn_user -d stackyn -f /path/to/migration.sql"
fi
echo ""

echo "5. Checking frontend..."
if docker compose ps frontend | grep -q "Up"; then
    echo -e "${GREEN}✅ Frontend service is running${NC}"
    
    # Check if new SignUp file exists
    if docker compose exec -T frontend test -f /app/src/pages/SignUp.tsx 2>/dev/null || \
       docker compose exec -T frontend test -f /usr/share/nginx/html/assets/*.js 2>/dev/null; then
        echo -e "${GREEN}✅ Frontend files exist${NC}"
    else
        echo -e "${YELLOW}⚠️  Cannot verify frontend files (might be in build)${NC}"
    fi
else
    echo -e "${RED}❌ Frontend service is not running${NC}"
fi
echo ""

echo "6. Testing API endpoint..."
API_URL="${API_URL:-http://localhost:8080}"
if curl -s -X POST "$API_URL/api/auth/send-otp" \
    -H "Content-Type: application/json" \
    -d '{"email":"test@example.com"}' | grep -q "message"; then
    echo -e "${GREEN}✅ API endpoint is responding${NC}"
else
    echo -e "${RED}❌ API endpoint is NOT responding correctly${NC}"
    echo "   Testing: curl -X POST $API_URL/api/auth/send-otp"
    curl -v -X POST "$API_URL/api/auth/send-otp" \
        -H "Content-Type: application/json" \
        -d '{"email":"test@example.com"}' 2>&1 | head -20
fi
echo ""

echo "7. Checking API logs for errors..."
echo "   Recent API logs (last 20 lines):"
docker compose logs api --tail=20 2>&1 | grep -i "error\|fail\|email\|otp" || echo "   No relevant errors found"
echo ""

echo "8. Recommendations:"
echo ""
if ! docker compose exec -T api test -f /app/internal/services/email.go 2>/dev/null; then
    echo -e "${YELLOW}⚠️  Code not deployed - rebuild required:${NC}"
    echo "   docker compose build api --no-cache"
    echo "   docker compose up -d api"
    echo ""
fi

if ! docker compose exec -T postgres psql -U stackyn_user -d stackyn -c "\d otps" &>/dev/null; then
    echo -e "${YELLOW}⚠️  Database migration needed:${NC}"
    echo "   Check migrations in server/internal/db/migrations/"
    echo ""
fi

if ! docker compose exec -T api env | grep -q "EMAIL_RESEND_API_KEY"; then
    echo -e "${YELLOW}⚠️  Environment variable missing:${NC}"
    echo "   Add to .env: EMAIL_RESEND_API_KEY=re_6iU1KmCf_3p6MzQRbsDyerP736x1WWExj"
    echo "   Then: docker compose up -d api"
    echo ""
fi

echo "✅ Diagnostic complete!"
echo ""
echo "Next steps:"
echo "1. If code not deployed: docker compose build --no-cache && docker compose up -d"
echo "2. If env vars missing: Add to .env and restart: docker compose up -d api"
echo "3. If table missing: Run database migrations"
echo "4. Clear browser cache and hard refresh (Ctrl+Shift+R)"

