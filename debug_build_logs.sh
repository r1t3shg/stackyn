#!/bin/bash
# Debug script for build logs issue
# Run this on the VPS to diagnose why build logs aren't showing

DEPLOYMENT_ID="7a94a847-b443-4f30-8264-db7019c4dc10"
APP_ID="723e17b2-fa56-4cf0-8fc4-95b39f489aa2"

echo "=========================================="
echo "Build Logs Debugging Script"
echo "=========================================="
echo ""

echo "1. Checking deployment record in database..."
docker exec stackyn-postgres psql -U stackyn_user -d stackyn -c "
SELECT 
    id,
    app_id,
    build_job_id,
    status,
    container_id,
    created_at
FROM deployments 
WHERE id = '$DEPLOYMENT_ID';
"

echo ""
echo "2. Checking if build_job_id exists in build_jobs table..."
docker exec stackyn-postgres psql -U stackyn_user -d stackyn -c "
SELECT 
    d.id as deployment_id,
    d.build_job_id,
    CASE 
        WHEN d.build_job_id IS NULL THEN 'NULL in deployment'
        WHEN bj.id IS NULL THEN 'build_job_id exists but build_job record NOT FOUND'
        ELSE 'build_job record EXISTS'
    END as build_job_status
FROM deployments d
LEFT JOIN build_jobs bj ON d.build_job_id = bj.id
WHERE d.id = '$DEPLOYMENT_ID';
"

echo ""
echo "3. Checking build logs in filesystem..."
echo "Listing build log directory:"
docker exec stackyn-build-worker ls -la /app/logs/$APP_ID/build/ 2>/dev/null || echo "Build log directory does not exist"

echo ""
echo "4. Checking if build log file exists (if build_job_id is known)..."
BUILD_JOB_ID=$(docker exec stackyn-postgres psql -U stackyn_user -d stackyn -t -c "SELECT build_job_id FROM deployments WHERE id = '$DEPLOYMENT_ID';" | tr -d '[:space:]')
if [ -n "$BUILD_JOB_ID" ] && [ "$BUILD_JOB_ID" != "" ]; then
    echo "Build job ID: $BUILD_JOB_ID"
    echo "Checking for log file: /app/logs/$APP_ID/build/$BUILD_JOB_ID.log"
    docker exec stackyn-build-worker ls -lh /app/logs/$APP_ID/build/$BUILD_JOB_ID.log 2>/dev/null || echo "Build log file does not exist"
    
    if docker exec stackyn-build-worker test -f /app/logs/$APP_ID/build/$BUILD_JOB_ID.log; then
        echo ""
        echo "Build log file exists! Showing first 20 lines:"
        docker exec stackyn-build-worker head -n 20 /app/logs/$APP_ID/build/$BUILD_JOB_ID.log
        echo ""
        echo "File size:"
        docker exec stackyn-build-worker stat -c "%s bytes" /app/logs/$APP_ID/build/$BUILD_JOB_ID.log
    fi
else
    echo "WARNING: build_job_id is NULL or empty in database!"
fi

echo ""
echo "5. Checking API logs for GetDeploymentLogs calls..."
echo "Recent GetDeploymentLogs calls:"
docker logs stackyn-api --tail 100 2>&1 | grep -E "GetDeploymentLogs|build_job_id|build.*log|Checking for build_job_id" | tail -20

echo ""
echo "6. Checking build-worker logs for build log persistence..."
echo "Recent build log persistence:"
docker logs stackyn-build-worker --tail 200 2>&1 | grep -E "PersistLog|build.*log|build_job_id.*$BUILD_JOB_ID" | tail -20

echo ""
echo "7. Checking all build log files for this app..."
echo "All build log files in /app/logs/$APP_ID/build/:"
docker exec stackyn-build-worker find /app/logs/$APP_ID/build -name "*.log" -type f -exec ls -lh {} \; 2>/dev/null || echo "No build log files found"

echo ""
echo "8. Testing API endpoint directly..."
echo "Making API call to get deployment logs:"
docker exec stackyn-api curl -s -H "Authorization: Bearer $(docker exec stackyn-api printenv JWT_SECRET 2>/dev/null || echo 'test')" \
    http://localhost:8080/api/v1/deployments/$DEPLOYMENT_ID/logs 2>/dev/null | head -c 500 || echo "API call failed"

echo ""
echo "=========================================="
echo "Debugging complete!"
echo "=========================================="

