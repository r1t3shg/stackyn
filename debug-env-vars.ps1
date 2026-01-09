# Debug script for environment variables issue
# Usage: .\debug-env-vars.ps1

Write-Host "==========================================" -ForegroundColor Cyan
Write-Host "Environment Variables Debug Script" -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan
Write-Host ""

$APP_ID = "90f6a191-295f-4311-b2d4-3074a1ad417b"
$DB_PASSWORD = if ($env:POSTGRES_PASSWORD) { $env:POSTGRES_PASSWORD } else { "changeme" }

Write-Host "1. Checking if PostgreSQL container is running..." -ForegroundColor Yellow
$postgresRunning = docker ps --filter "name=stackyn-postgres" --format "{{.Names}}" | Select-String "stackyn-postgres"
if ($postgresRunning) {
    Write-Host "✓ PostgreSQL container is running" -ForegroundColor Green
} else {
    Write-Host "✗ PostgreSQL container is not running" -ForegroundColor Red
    Write-Host "   Start it with: docker-compose up -d postgres" -ForegroundColor Yellow
    exit 1
}
Write-Host ""

Write-Host "2. Checking if API container is running..." -ForegroundColor Yellow
$apiRunning = docker ps --filter "name=stackyn-api" --format "{{.Names}}" | Select-String "stackyn-api"
if ($apiRunning) {
    Write-Host "✓ API container is running" -ForegroundColor Green
} else {
    Write-Host "✗ API container is not running" -ForegroundColor Red
    Write-Host "   Start it with: docker-compose up -d api" -ForegroundColor Yellow
}
Write-Host ""

Write-Host "3. Checking if env_vars table exists..." -ForegroundColor Yellow
$tableCheck = docker exec stackyn-postgres psql -U stackyn_user -d stackyn -tAc "SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'env_vars');"
if ($tableCheck -eq "t") {
    Write-Host "✓ env_vars table exists" -ForegroundColor Green
} else {
    Write-Host "✗ env_vars table does NOT exist" -ForegroundColor Red
    Write-Host "   Run migrations or check database initialization" -ForegroundColor Yellow
    exit 1
}
Write-Host ""

Write-Host "4. Checking table structure..." -ForegroundColor Yellow
Write-Host "   Table schema:" -ForegroundColor Gray
docker exec stackyn-postgres psql -U stackyn_user -d stackyn -c "\d env_vars"
Write-Host ""

Write-Host "5. Checking if app exists in database..." -ForegroundColor Yellow
$appCheck = docker exec stackyn-postgres psql -U stackyn_user -d stackyn -tAc "SELECT EXISTS (SELECT 1 FROM apps WHERE id = '$APP_ID');"
if ($appCheck -eq "t") {
    Write-Host "✓ App $APP_ID exists in database" -ForegroundColor Green
    Write-Host "   App details:" -ForegroundColor Gray
    docker exec stackyn-postgres psql -U stackyn_user -d stackyn -c "SELECT id, name, user_id, created_at FROM apps WHERE id = '$APP_ID';"
} else {
    Write-Host "✗ App $APP_ID does NOT exist in database" -ForegroundColor Red
    Write-Host "   Available apps:" -ForegroundColor Gray
    docker exec stackyn-postgres psql -U stackyn_user -d stackyn -c "SELECT id, name, user_id FROM apps LIMIT 10;"
}
Write-Host ""

Write-Host "6. Checking existing environment variables for this app..." -ForegroundColor Yellow
$envCount = docker exec stackyn-postgres psql -U stackyn_user -d stackyn -tAc "SELECT COUNT(*) FROM env_vars WHERE app_id = '$APP_ID';"
Write-Host "   Found $envCount environment variable(s) for this app" -ForegroundColor Gray
if ([int]$envCount -gt 0) {
    Write-Host "   Existing variables:" -ForegroundColor Gray
    docker exec stackyn-postgres psql -U stackyn_user -d stackyn -c "SELECT key, LEFT(value, 50) as value_preview, created_at FROM env_vars WHERE app_id = '$APP_ID';"
}
Write-Host ""

Write-Host "7. Checking API container logs for recent errors..." -ForegroundColor Yellow
Write-Host "   Last 30 lines of API logs:" -ForegroundColor Gray
docker logs stackyn-api --tail 30 2>&1 | Select-String -Pattern "error|env|envvar|env_var|500" -CaseSensitive:$false
Write-Host ""

Write-Host "8. Checking API container health..." -ForegroundColor Yellow
try {
    $response = Invoke-WebRequest -Uri "http://localhost:8080/health" -Method GET -TimeoutSec 5 -UseBasicParsing
    if ($response.StatusCode -eq 200) {
        Write-Host "✓ API health check passed (HTTP $($response.StatusCode))" -ForegroundColor Green
    }
} catch {
    Write-Host "⚠ API health check failed or API not accessible on localhost:8080" -ForegroundColor Yellow
    Write-Host "   If using production domain, try: curl https://api.dev.stackyn.com/health" -ForegroundColor Gray
}
Write-Host ""

Write-Host "==========================================" -ForegroundColor Cyan
Write-Host "Debug Summary" -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan
Write-Host "If all checks passed, the issue might be:" -ForegroundColor Yellow
Write-Host "  - Authentication/authorization issue" -ForegroundColor Gray
Write-Host "  - Request format issue" -ForegroundColor Gray
Write-Host "  - Network/CORS issue" -ForegroundColor Gray
Write-Host ""
Write-Host "Next steps:" -ForegroundColor Yellow
Write-Host "  1. Check browser console for detailed error messages" -ForegroundColor Gray
Write-Host "  2. Check API logs: docker logs -f stackyn-api" -ForegroundColor Gray
Write-Host "  3. Try the API endpoint with curl (see commands below)" -ForegroundColor Gray
Write-Host "  4. Verify your auth token is valid" -ForegroundColor Gray
Write-Host ""

