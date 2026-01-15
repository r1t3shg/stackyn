# OTP Signup Deployment Checklist

## Issues to Check

### 1. **Frontend Not Rebuilt**
The frontend needs to be rebuilt to include the new SignUp component.

**Solution:**
```bash
# On your VPS, rebuild the frontend
cd /opt/stackyn  # or wherever your project is
docker compose build frontend --no-cache
docker compose up -d frontend
```

### 2. **Backend Not Restarted**
The backend needs to be restarted to load the new code.

**Solution:**
```bash
# Restart the API service
docker compose restart api

# Or rebuild and restart
docker compose build api --no-cache
docker compose up -d api
```

### 3. **Environment Variables Not Set**
The Resend API key needs to be in your `.env` file or environment.

**Solution:**
```bash
# Check if RESEND_API_KEY is set
docker compose exec api env | grep RESEND

# Add to .env file (if not present)
echo "RESEND_API_KEY=re_6iU1KmCf_3p6MzQRbsDyerP736x1WWExj" >> .env
echo "EMAIL_FROM_EMAIL=noreply@stackyn.com" >> .env

# Restart API after adding env vars
docker compose up -d api
```

### 4. **Database Migrations Not Run**
The OTP table might not exist in the database.

**Solution:**
```bash
# Check if OTP table exists
docker compose exec postgres psql -U stackyn_user -d stackyn -c "\d otps"

# If table doesn't exist, run migrations
# The migration should run automatically, but you can check:
docker compose exec api ls -la /app/migrations  # or wherever migrations are

# Or manually create the table:
docker compose exec postgres psql -U stackyn_user -d stackyn << EOF
CREATE TABLE IF NOT EXISTS otps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL,
    otp_hash VARCHAR(255) NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    used BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_otps_email ON otps(email);
CREATE INDEX IF NOT EXISTS idx_otps_expires_at ON otps(expires_at);
CREATE INDEX IF NOT EXISTS idx_otps_used ON otps(used);
EOF
```

### 5. **Browser Cache**
Your browser might be showing the old signup page.

**Solution:**
- Hard refresh: `Ctrl+Shift+R` (Windows/Linux) or `Cmd+Shift+R` (Mac)
- Clear browser cache
- Try incognito/private mode

### 6. **Check API Endpoints**
Verify the backend endpoints are working.

**Solution:**
```bash
# Test send-otp endpoint
curl -X POST https://api.staging.stackyn.com/api/auth/send-otp \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com"}'

# Should return: {"message":"OTP sent to email"}
```

### 7. **Check Logs**
Check for errors in the logs.

**Solution:**
```bash
# Check API logs
docker compose logs api --tail=100

# Check frontend logs
docker compose logs frontend --tail=100

# Look for errors related to:
# - Email service initialization
# - OTP service
# - Database connection
# - Missing environment variables
```

### 8. **Verify Code is Deployed**
Make sure the new code is actually on the VPS.

**Solution:**
```bash
# Check if new files exist
docker compose exec api ls -la /app/internal/services/email.go
docker compose exec api ls -la /app/internal/api/repositories.go

# Check frontend
docker compose exec frontend ls -la /app/src/pages/SignUp.tsx
```

## Quick Fix Commands

Run these in order:

```bash
# 1. Pull latest code
cd /opt/stackyn  # or your project directory
git pull

# 2. Rebuild everything
docker compose build --no-cache

# 3. Ensure environment variables are set
# Edit .env file and add:
# RESEND_API_KEY=re_6iU1KmCf_3p6MzQRbsDyerP736x1WWExj
# EMAIL_FROM_EMAIL=noreply@stackyn.com

# 4. Restart all services
docker compose down
docker compose up -d

# 5. Check logs
docker compose logs -f api
```

## Verify Deployment

1. **Check signup page loads:**
   - Visit: `https://staging.stackyn.com/signup`
   - Should see "Create account" with email input (not password fields)

2. **Test OTP flow:**
   - Enter email
   - Click "Continue"
   - Should see OTP input screen
   - Check email for OTP code

3. **Check browser console:**
   - Open DevTools (F12)
   - Check Console tab for errors
   - Check Network tab for API calls

## Common Error Messages

- **"Email service not configured"** → RESEND_API_KEY not set
- **"Failed to send OTP"** → Check Resend API key validity
- **"OTP table doesn't exist"** → Run database migrations
- **"Cannot connect to backend"** → API service not running or wrong URL

