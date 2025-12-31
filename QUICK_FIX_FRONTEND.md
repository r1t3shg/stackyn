# Quick Fix: Rebuild Frontend for OTP Signup UI

## The Problem
You're seeing the old signup UI because the frontend container is using a cached build that doesn't include the new SignUp component.

## Solution

### On Your VPS, run:

```bash
cd /opt/stackyn  # or wherever your project is

# Option 1: Quick rebuild (recommended)
docker compose build frontend --no-cache
docker compose up -d frontend

# Option 2: Full rebuild (if quick doesn't work)
docker compose stop frontend
docker compose rm -f frontend
docker compose build frontend --no-cache
docker compose up -d frontend

# Option 3: Use the script
chmod +x rebuild_frontend.sh
./rebuild_frontend.sh
```

### After Rebuilding:

1. **Clear Browser Cache:**
   - Hard refresh: `Ctrl+Shift+R` (Windows/Linux) or `Cmd+Shift+R` (Mac)
   - Or use incognito/private mode

2. **Verify the New UI:**
   - Visit: `https://staging.stackyn.com/signup`
   - You should see:
     - "Create account" heading
     - Email input field (NOT password fields)
     - "Continue" button

3. **Check if it's working:**
   ```bash
   # Check frontend logs
   docker compose logs frontend --tail=50
   
   # Check if container is running
   docker compose ps frontend
   ```

## Why This Happens

Docker caches build layers. When you update source code:
- Docker sees the same `package.json` and thinks nothing changed
- It uses the cached build from before
- Your new `SignUp.tsx` file isn't included

Using `--no-cache` forces Docker to:
- Re-read all source files
- Rebuild the entire frontend
- Include your new code

## Verify New Code is Deployed

```bash
# Check if new SignUp file exists in container
docker compose exec frontend ls -la /app/dist/assets/*.js | head -5

# Or check build logs for the new component
docker compose logs frontend | grep -i "signup\|otp"
```

## Still Not Working?

1. **Check browser console (F12):**
   - Look for JavaScript errors
   - Check Network tab for failed requests

2. **Verify the route:**
   - Make sure you're visiting `/signup` (not `/sign-up` or `/register`)

3. **Check frontend container:**
   ```bash
   docker compose exec frontend ls -la /app/dist
   # Should show index.html and assets folder
   ```

4. **Full restart:**
   ```bash
   docker compose down
   docker compose build --no-cache
   docker compose up -d
   ```

