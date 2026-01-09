# Debug Commands for Environment Variables Issue

## Quick Debug Script

Run the PowerShell script:
```powershell
.\debug-env-vars.ps1
```

Or run individual commands below.

## Individual Debug Commands

### 1. Check if containers are running
```powershell
docker ps --filter "name=stackyn"
```

### 2. Check if env_vars table exists
```powershell
docker exec stackyn-postgres psql -U stackyn_user -d stackyn -c "\d env_vars"
```

### 3. Check if the app exists
```powershell
docker exec stackyn-postgres psql -U stackyn_user -d stackyn -c "SELECT id, name, user_id FROM apps WHERE id = '90f6a191-295f-4311-b2d4-3074a1ad417b';"
```

### 4. List all apps (if the app ID doesn't exist)
```powershell
docker exec stackyn-postgres psql -U stackyn_user -d stackyn -c "SELECT id, name, user_id FROM apps LIMIT 10;"
```

### 5. Check existing environment variables for the app
```powershell
docker exec stackyn-postgres psql -U stackyn_user -d stackyn -c "SELECT key, value, created_at FROM env_vars WHERE app_id = '90f6a191-295f-4311-b2d4-3074a1ad417b';"
```

### 6. Test direct database insert (will rollback)
```powershell
docker exec stackyn-postgres psql -U stackyn_user -d stackyn -c "BEGIN; INSERT INTO env_vars (app_id, key, value) VALUES ('90f6a191-295f-4311-b2d4-3074a1ad417b', 'TEST_KEY', 'test_value') ON CONFLICT (app_id, key) DO UPDATE SET value = EXCLUDED.value; ROLLBACK;"
```

### 7. Check API logs for errors
```powershell
docker logs stackyn-api --tail 50 | Select-String -Pattern "error|env|500" -CaseSensitive:$false
```

### 8. Follow API logs in real-time
```powershell
docker logs -f stackyn-api
```

### 9. Check database constraints on env_vars table
```powershell
docker exec stackyn-postgres psql -U stackyn_user -d stackyn -c "SELECT conname, contype, pg_get_constraintdef(oid) FROM pg_constraint WHERE conrelid = 'env_vars'::regclass;"
```

### 10. Check if foreign key constraint is working
```powershell
docker exec stackyn-postgres psql -U stackyn_user -d stackyn -c "SELECT conname, confrelid::regclass, confkey FROM pg_constraint WHERE conrelid = 'env_vars'::regclass AND contype = 'f';"
```

### 11. Test API endpoint with curl (replace YOUR_TOKEN)
```powershell
$token = "YOUR_AUTH_TOKEN"
$appId = "90f6a191-295f-4311-b2d4-3074a1ad417b"
$body = @{
    key = "TEST_KEY"
    value = "test_value"
} | ConvertTo-Json

Invoke-RestMethod -Uri "https://api.dev.stackyn.com/api/v1/apps/$appId/env" `
    -Method POST `
    -Headers @{
        "Content-Type" = "application/json"
        "Authorization" = "Bearer $token"
    } `
    -Body $body
```

### 12. Check API health endpoint
```powershell
Invoke-WebRequest -Uri "https://api.dev.stackyn.com/health" -Method GET
```

### 13. Check database connection from API container
```powershell
docker exec stackyn-api sh -c 'psql -h postgres -U stackyn_user -d stackyn -c "SELECT 1;"'
```

### 14. Check table row count
```powershell
docker exec stackyn-postgres psql -U stackyn_user -d stackyn -c "SELECT COUNT(*) FROM env_vars;"
```

### 15. Check for any database errors in PostgreSQL logs
```powershell
docker logs stackyn-postgres --tail 50 | Select-String -Pattern "error|ERROR" -CaseSensitive:$false
```

## Common Issues and Solutions

### Issue: Table doesn't exist
**Solution:** Run database migrations
```powershell
# Check if migrations need to be run
docker exec stackyn-api sh -c 'cd /app && ./api migrate'
```

### Issue: App ID doesn't exist
**Solution:** Use a valid app ID from your database
```powershell
docker exec stackyn-postgres psql -U stackyn_user -d stackyn -c "SELECT id, name FROM apps;"
```

### Issue: Foreign key constraint violation
**Solution:** Ensure the app exists before creating env vars
```powershell
# Verify app exists
docker exec stackyn-postgres psql -U stackyn_user -d stackyn -c "SELECT id FROM apps WHERE id = 'YOUR_APP_ID';"
```

### Issue: Authentication error
**Solution:** Check your auth token
```powershell
# Get token from browser localStorage or check if it's expired
# In browser console: localStorage.getItem('auth_token')
```

## View Real-time Logs

To see errors as they happen:
```powershell
# API logs
docker logs -f stackyn-api

# PostgreSQL logs  
docker logs -f stackyn-postgres
```

## Database Connection Details

- **Host:** postgres (from within Docker network) or localhost (from host)
- **Port:** 5432
- **Database:** stackyn
- **User:** stackyn_user
- **Password:** Check your `.env` file or use default `changeme`

## Next Steps After Debugging

1. If table doesn't exist → Run migrations
2. If app doesn't exist → Use correct app ID
3. If foreign key error → Verify app exists
4. If auth error → Check token validity
5. If 500 error persists → Check API logs for detailed error message

