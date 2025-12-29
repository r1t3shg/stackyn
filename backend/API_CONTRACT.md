# API Contract Document

This document defines the exact API contracts that the backend must implement based on the frontend codebase. All endpoints, request/response formats, and error codes are extracted directly from the frontend code.

**Base URL**: `http://localhost:8080` (configurable via `VITE_API_BASE_URL` or `NEXT_PUBLIC_API_BASE_URL`)

**Authentication**: 
- Most endpoints require Bearer token authentication via `Authorization: Bearer <token>` header
- Token is stored in `localStorage.getItem('auth_token')` on the frontend
- Endpoints marked as "No Auth Required" do not require authentication

**Error Response Format**:
- All errors return JSON with format: `{ "error": "error message" }`
- HTTP status codes: 400 (Bad Request), 401 (Unauthorized), 403 (Forbidden), 404 (Not Found), 500 (Internal Server Error)
- When `response.ok` is false, frontend expects JSON with `error` field

---

## Table of Contents

1. [Health Check](#health-check)
2. [Authentication Endpoints](#authentication-endpoints)
3. [User Endpoints](#user-endpoints)
4. [Apps Endpoints](#apps-endpoints)
5. [Deployments Endpoints](#deployments-endpoints)
6. [Environment Variables Endpoints](#environment-variables-endpoints)
7. [Admin Endpoints](#admin-endpoints)

---

## Health Check

### GET /health
**Auth Required**: No

**Response**: `200 OK`
```json
{
  "status": "string"
}
```

**Error Codes**: None expected (health check should always succeed or fail with network error)

---

## Authentication Endpoints

### POST /api/auth/verify-token
**Auth Required**: No

**Request Headers**:
- `Content-Type: application/json`

**Request Body**:
```json
{
  "id_token": "string"  // Firebase ID token
}
```

**Response**: `200 OK`
```json
{
  "uid": "string",
  "email": "string",
  "email_verified": boolean
}
```

**Error Codes**:
- `400`: Invalid token
- `401`: Token verification failed

---

### POST /api/auth/login
**Auth Required**: No

**Request Headers**:
- `Content-Type: application/json`

**Request Body**:
```json
{
  "email": "string",
  "password": "string"
}
```

**Response**: `200 OK`
```json
{
  "user": {
    "id": "string",
    "email": "string"
  },
  "token": "string"  // Bearer token for subsequent requests
}
```

**Error Codes**:
- `401`: Invalid credentials
- `400`: Missing email or password

**Note**: Used by CMS admin interface. Frontend expects this exact format and stores token in localStorage.

---

## User Endpoints

### GET /api/user/me
**Auth Required**: Yes (Bearer token)

**Request Headers**:
- `Authorization: Bearer <token>`

**Response**: `200 OK`
```json
{
  "id": "string",
  "email": "string",
  "full_name": "string | null | undefined",
  "company_name": "string | null | undefined",
  "email_verified": boolean,
  "plan": "string",
  "created_at": "string",  // ISO 8601 datetime
  "updated_at": "string",  // ISO 8601 datetime
  "quota": {
    "plan_name": "string",
    "plan": {
      "name": "string",
      "display_name": "string",
      "price": number,
      "max_ram_mb": number,
      "max_disk_mb": number,
      "max_apps": number,
      "always_on": boolean,
      "auto_deploy": boolean,
      "health_checks": boolean,
      "logs": boolean,
      "zero_downtime": boolean,
      "workers": boolean,
      "priority_builds": boolean,
      "manual_deploy_only": boolean
    },
    "app_count": number,
    "total_ram_mb": number,
    "total_disk_mb": number
  } | null | undefined
}
```

**Error Codes**:
- `401`: Unauthorized (missing or invalid token)
- `404`: User not found

---

## Apps Endpoints

### GET /api/apps
**Auth Required**: Yes (Bearer token)

**Request Headers**:
- `Authorization: Bearer <token>`

**Response**: `200 OK`
```json
[
  {
    "id": "string",
    "name": "string",
    "slug": "string",
    "status": "string",
    "url": "string",
    "repo_url": "string",
    "branch": "string",
    "created_at": "string",  // ISO 8601 datetime
    "updated_at": "string",  // ISO 8601 datetime
    "deployment": {
      "active_deployment_id": "string | null",
      "last_deployed_at": "string | null",  // ISO 8601 datetime
      "state": "string",
      "resource_limits": {
        "memory_mb": number,
        "cpu": number,
        "disk_gb": number
      } | null | undefined,
      "usage_stats": {
        "memory_usage_mb": number,
        "memory_usage_percent": number,
        "disk_usage_gb": number,
        "disk_usage_percent": number,
        "restart_count": number
      } | null | undefined
    } | null | undefined
  }
]
```

**Note**: Frontend expects an array. If null/undefined, frontend converts to empty array `[]`.

**Error Codes**:
- `401`: Unauthorized
- `500`: Internal server error

---

### GET /api/v1/apps/{id}
**Auth Required**: Yes (Bearer token)

**Path Parameters**:
- `id`: string | number (app ID)

**Request Headers**:
- `Authorization: Bearer <token>`

**Response**: `200 OK`
```json
{
  "id": "string",
  "name": "string",
  "slug": "string",
  "status": "string",
  "url": "string",
  "repo_url": "string",
  "branch": "string",
  "created_at": "string",  // ISO 8601 datetime
  "updated_at": "string",  // ISO 8601 datetime
  "deployment": {
    "active_deployment_id": "string | null",
    "last_deployed_at": "string | null",  // ISO 8601 datetime
    "state": "string",
    "resource_limits": {
      "memory_mb": number,
      "cpu": number,
      "disk_gb": number
    } | null | undefined,
    "usage_stats": {
      "memory_usage_mb": number,
      "memory_usage_percent": number,
      "disk_usage_gb": number,
      "disk_usage_percent": number,
      "restart_count": number
    } | null | undefined
  } | null | undefined
}
```

**Error Codes**:
- `401`: Unauthorized
- `404`: App not found
- `403`: Forbidden (user doesn't own this app)

---

### POST /api/v1/apps
**Auth Required**: Yes (Bearer token)

**Request Headers**:
- `Authorization: Bearer <token>`
- `Content-Type: application/json`

**Request Body**:
```json
{
  "name": "string",
  "repo_url": "string",
  "branch": "string"
}
```

**Response**: `200 OK` or `201 Created`
```json
{
  "app": {
    "id": "string",
    "name": "string",
    "slug": "string",
    "status": "string",
    "url": "string",
    "repo_url": "string",
    "branch": "string",
    "created_at": "string",  // ISO 8601 datetime
    "updated_at": "string",  // ISO 8601 datetime
    "deployment": {
      "active_deployment_id": "string | null",
      "last_deployed_at": "string | null",  // ISO 8601 datetime
      "state": "string"
    } | null | undefined
  },
  "deployment": {
    "id": number,
    "app_id": number,
    "status": "pending" | "building" | "running" | "failed" | "stopped",
    "image_name": "string | null" | { "String": "string", "Valid": boolean } | null | undefined,
    "container_id": "string | null" | { "String": "string", "Valid": boolean } | null | undefined,
    "subdomain": "string | null" | { "String": "string", "Valid": boolean } | null | undefined,
    "build_log": "string | null" | { "String": "string", "Valid": boolean } | null | undefined,
    "runtime_log": "string | null" | { "String": "string", "Valid": boolean } | null | undefined,
    "error_message": "string | null" | { "String": "string", "Valid": boolean } | null | undefined,
    "created_at": "string",  // ISO 8601 datetime
    "updated_at": "string"   // ISO 8601 datetime
  },
  "error": "string | null | undefined"  // Optional error message
}
```

**Note**: Deployment fields may be serialized as Go's `sql.NullString` format: `{ "String": "...", "Valid": true }` or as regular strings. Frontend handles both formats.

**Error Codes**:
- `400`: Invalid request (missing required fields, invalid repo_url, etc.)
- `401`: Unauthorized
- `409`: App name already exists
- `500`: Internal server error

---

### DELETE /api/v1/apps/{id}
**Auth Required**: Yes (Bearer token)

**Path Parameters**:
- `id`: string | number (app ID)

**Request Headers**:
- `Authorization: Bearer <token>`

**Response**: `200 OK` or `204 No Content`
- No response body expected (empty response)

**Timeout**: Frontend uses 120 second (2 minute) timeout for this operation due to Docker cleanup operations.

**Error Codes**:
- `401`: Unauthorized
- `404`: App not found
- `403`: Forbidden (user doesn't own this app)
- `500`: Internal server error

**Error Response**: `{ "error": "error message" }`

---

### POST /api/v1/apps/{id}/redeploy
**Auth Required**: Yes (Bearer token)

**Path Parameters**:
- `id`: string | number (app ID)

**Request Headers**:
- `Authorization: Bearer <token>`

**Request Body**: None

**Response**: `200 OK`
```json
{
  "app": {
    "id": "string",
    "name": "string",
    "slug": "string",
    "status": "string",
    "url": "string",
    "repo_url": "string",
    "branch": "string",
    "created_at": "string",  // ISO 8601 datetime
    "updated_at": "string",  // ISO 8601 datetime
    "deployment": {
      "active_deployment_id": "string | null",
      "last_deployed_at": "string | null",  // ISO 8601 datetime
      "state": "string"
    } | null | undefined
  },
  "deployment": {
    "id": number,
    "app_id": number,
    "status": "pending" | "building" | "running" | "failed" | "stopped",
    "image_name": "string | null" | { "String": "string", "Valid": boolean } | null | undefined,
    "container_id": "string | null" | { "String": "string", "Valid": boolean } | null | undefined,
    "subdomain": "string | null" | { "String": "string", "Valid": boolean } | null | undefined,
    "build_log": "string | null" | { "String": "string", "Valid": boolean } | null | undefined,
    "runtime_log": "string | null" | { "String": "string", "Valid": boolean } | null | undefined,
    "error_message": "string | null" | { "String": "string", "Valid": boolean } | null | undefined,
    "created_at": "string",  // ISO 8601 datetime
    "updated_at": "string"   // ISO 8601 datetime
  },
  "error": "string | null | undefined"
}
```

**Error Codes**:
- `401`: Unauthorized
- `404`: App not found
- `403`: Forbidden (user doesn't own this app)
- `500`: Internal server error

---

### GET /api/v1/apps/{id}/deployments
**Auth Required**: Yes (Bearer token)

**Path Parameters**:
- `id`: string | number (app ID)

**Request Headers**:
- `Authorization: Bearer <token>`

**Response**: `200 OK`
```json
[
  {
    "id": number,
    "app_id": number,
    "status": "pending" | "building" | "running" | "failed" | "stopped",
    "image_name": "string | null" | { "String": "string", "Valid": boolean } | null | undefined,
    "container_id": "string | null" | { "String": "string", "Valid": boolean } | null | undefined,
    "subdomain": "string | null" | { "String": "string", "Valid": boolean } | null | undefined,
    "build_log": "string | null" | { "String": "string", "Valid": boolean } | null | undefined,
    "runtime_log": "string | null" | { "String": "string", "Valid": boolean } | null | undefined,
    "error_message": "string | null" | { "String": "string", "Valid": boolean } | null | undefined,
    "created_at": "string",  // ISO 8601 datetime
    "updated_at": "string"   // ISO 8601 datetime
  }
]
```

**Note**: Frontend expects an array. If null/undefined, frontend converts to empty array `[]`.

**Error Codes**:
- `401`: Unauthorized
- `404`: App not found
- `403`: Forbidden (user doesn't own this app)

---

## Deployments Endpoints

### GET /api/v1/deployments/{id}
**Auth Required**: Yes (Bearer token)

**Path Parameters**:
- `id`: string | number (deployment ID)

**Request Headers**:
- `Authorization: Bearer <token>`

**Response**: `200 OK`
```json
{
  "id": number,
  "app_id": number,
  "status": "pending" | "building" | "running" | "failed" | "stopped",
  "image_name": "string | null" | { "String": "string", "Valid": boolean } | null | undefined,
  "container_id": "string | null" | { "String": "string", "Valid": boolean } | null | undefined,
  "subdomain": "string | null" | { "String": "string", "Valid": boolean } | null | undefined,
  "build_log": "string | null" | { "String": "string", "Valid": boolean } | null | undefined,
  "runtime_log": "string | null" | { "String": "string", "Valid": boolean } | null | undefined,
  "error_message": "string | null" | { "String": "string", "Valid": boolean } | null | undefined,
  "created_at": "string",  // ISO 8601 datetime
  "updated_at": "string"   // ISO 8601 datetime
}
```

**Error Codes**:
- `401`: Unauthorized
- `404`: Deployment not found
- `403`: Forbidden (user doesn't have access to this deployment's app)

---

### GET /api/v1/deployments/{id}/logs
**Auth Required**: Yes (Bearer token)

**Path Parameters**:
- `id`: string | number (deployment ID)

**Request Headers**:
- `Authorization: Bearer <token>`

**Response**: `200 OK`
```json
{
  "deployment_id": number,
  "status": "string",
  "build_log": "string | null | undefined",
  "runtime_log": "string | null | undefined",
  "error_message": "string | null | undefined"
}
```

**Error Codes**:
- `401`: Unauthorized
- `404`: Deployment not found
- `403`: Forbidden (user doesn't have access to this deployment's app)

---

## Environment Variables Endpoints

### GET /api/v1/apps/{id}/env
**Auth Required**: Yes (Bearer token)

**Path Parameters**:
- `id`: string | number (app ID)

**Request Headers**:
- `Authorization: Bearer <token>`

**Response**: `200 OK`
```json
[
  {
    "id": number,
    "app_id": number,
    "key": "string",
    "value": "string",
    "created_at": "string",  // ISO 8601 datetime
    "updated_at": "string"   // ISO 8601 datetime
  }
]
```

**Note**: Frontend expects an array. If null/undefined, frontend converts to empty array `[]`.

**Error Codes**:
- `401`: Unauthorized
- `404`: App not found
- `403`: Forbidden (user doesn't own this app)

---

### POST /api/v1/apps/{id}/env
**Auth Required**: Yes (Bearer token)

**Path Parameters**:
- `id`: string | number (app ID)

**Request Headers**:
- `Authorization: Bearer <token>`
- `Content-Type: application/json`

**Request Body**:
```json
{
  "key": "string",
  "value": "string"
}
```

**Response**: `200 OK` or `201 Created`
```json
{
  "id": number,
  "app_id": number,
  "key": "string",
  "value": "string",
  "created_at": "string",  // ISO 8601 datetime
  "updated_at": "string"   // ISO 8601 datetime
}
```

**Note**: This endpoint creates or updates an environment variable. If the key already exists, it updates the value.

**Error Codes**:
- `400`: Invalid request (missing key or value)
- `401`: Unauthorized
- `404`: App not found
- `403`: Forbidden (user doesn't own this app)
- `500`: Internal server error

---

### DELETE /api/v1/apps/{id}/env/{key}
**Auth Required**: Yes (Bearer token)

**Path Parameters**:
- `id`: string | number (app ID)
- `key`: string (environment variable key, URL encoded)

**Request Headers**:
- `Authorization: Bearer <token>`

**Response**: `200 OK` or `204 No Content`
- No response body expected (empty response)

**Error Codes**:
- `401`: Unauthorized
- `404`: App or environment variable not found
- `403`: Forbidden (user doesn't own this app)
- `500`: Internal server error

**Error Response**: `{ "error": "error message" }`

---

## Admin Endpoints

### GET /admin/users
**Auth Required**: Yes (Bearer token, admin role)

**Query Parameters**:
- `limit`: number (default: 50)
- `offset`: number (default: 0)
- `search`: string (optional, for searching by email)

**Request Headers**:
- `Authorization: Bearer <token>`

**Response**: `200 OK`
```json
{
  "users": [
    {
      "id": "string",
      "email": "string",
      "full_name": "string | null | undefined",
      "company_name": "string | null | undefined",
      "email_verified": boolean,
      "plan": "string",
      "is_admin": boolean,
      "created_at": "string",  // ISO 8601 datetime
      "updated_at": "string",  // ISO 8601 datetime
      "quota": {
        "plan_name": "string",
        "plan": {
          "name": "string",
          "display_name": "string",
          "price": number,
          "max_ram_mb": number,
          "max_disk_mb": number,
          "max_apps": number,
          "always_on": boolean,
          "auto_deploy": boolean,
          "health_checks": boolean,
          "logs": boolean,
          "zero_downtime": boolean,
          "workers": boolean,
          "priority_builds": boolean,
          "manual_deploy_only": boolean
        },
        "app_count": number,
        "total_ram_mb": number,
        "total_disk_mb": number
      } | null | undefined
    }
  ],
  "total": number,
  "limit": number,
  "offset": number
}
```

**Error Codes**:
- `401`: Unauthorized
- `403`: Forbidden (not an admin)
- `500`: Internal server error

---

### GET /admin/users/{id}
**Auth Required**: Yes (Bearer token, admin role)

**Path Parameters**:
- `id`: string (user ID)

**Request Headers**:
- `Authorization: Bearer <token>`

**Response**: `200 OK`
```json
{
  "id": "string",
  "email": "string",
  "full_name": "string | null | undefined",
  "company_name": "string | null | undefined",
  "email_verified": boolean,
  "plan": "string",
  "is_admin": boolean,
  "created_at": "string",  // ISO 8601 datetime
  "updated_at": "string",  // ISO 8601 datetime
  "quota": {
    "plan_name": "string",
    "plan": {
      "name": "string",
      "display_name": "string",
      "price": number,
      "max_ram_mb": number,
      "max_disk_mb": number,
      "max_apps": number,
      "always_on": boolean,
      "auto_deploy": boolean,
      "health_checks": boolean,
      "logs": boolean,
      "zero_downtime": boolean,
      "workers": boolean,
      "priority_builds": boolean,
      "manual_deploy_only": boolean
    },
    "app_count": number,
    "total_ram_mb": number,
    "total_disk_mb": number
  } | null | undefined
}
```

**Error Codes**:
- `401`: Unauthorized
- `403`: Forbidden (not an admin)
- `404`: User not found

---

### PATCH /admin/users/{id}/plan
**Auth Required**: Yes (Bearer token, admin role)

**Path Parameters**:
- `id`: string (user ID)

**Request Headers**:
- `Authorization: Bearer <token>`
- `Content-Type: application/json`

**Request Body**:
```json
{
  "plan": "string"  // Plan name: "free", "starter", "builder", "pro"
}
```

**Response**: `200 OK`
```json
{
  "message": "string",
  "user_id": "string",
  "plan": "string",
  "user": {
    // User object (same as GET /admin/users/{id} response)
  } | null | undefined,
  "quota": {
    // Quota object (same structure as user.quota in GET /admin/users/{id})
  } | null | undefined
}
```

**Error Codes**:
- `400`: Invalid plan name
- `401`: Unauthorized
- `403`: Forbidden (not an admin)
- `404`: User not found
- `500`: Internal server error

---

### GET /admin/apps
**Auth Required**: Yes (Bearer token, admin role)

**Query Parameters**:
- `limit`: number (default: 50)
- `offset`: number (default: 0)

**Request Headers**:
- `Authorization: Bearer <token>`

**Response**: `200 OK`
```json
{
  "apps": [
    {
      "id": "string",
      "name": "string",
      "slug": "string",
      "status": "string",
      "url": "string",
      "repo_url": "string",
      "branch": "string",
      "created_at": "string",  // ISO 8601 datetime
      "updated_at": "string",  // ISO 8601 datetime
      "deployment_count": number | null | undefined,
      "latest_status": "string | null | undefined"
    }
  ],
  "total": number,
  "limit": number,
  "offset": number
}
```

**Error Codes**:
- `401`: Unauthorized
- `403`: Forbidden (not an admin)
- `500`: Internal server error

---

### POST /admin/apps/{id}/stop
**Auth Required**: Yes (Bearer token, admin role)

**Path Parameters**:
- `id`: string (app ID)

**Request Headers**:
- `Authorization: Bearer <token>`

**Request Body**: None

**Response**: `200 OK`
```json
{
  "message": "string",
  "app_id": number,
  "stopped_containers": number
}
```

**Error Codes**:
- `401`: Unauthorized
- `403`: Forbidden (not an admin)
- `404`: App not found
- `500`: Internal server error

---

### POST /admin/apps/{id}/start
**Auth Required**: Yes (Bearer token, admin role)

**Path Parameters**:
- `id`: string (app ID)

**Request Headers**:
- `Authorization: Bearer <token>`

**Request Body**: None

**Response**: `200 OK`
```json
{
  "message": "string",
  "app_id": number
}
```

**Error Codes**:
- `401`: Unauthorized
- `403`: Forbidden (not an admin)
- `404`: App not found
- `500`: Internal server error

---

### POST /admin/apps/{id}/redeploy
**Auth Required**: Yes (Bearer token, admin role)

**Path Parameters**:
- `id`: string (app ID)

**Request Headers**:
- `Authorization: Bearer <token>`

**Request Body**: None

**Response**: `200 OK`
```json
{
  "message": "string",
  "app_id": number,
  "deployment": {
    // Deployment object (same structure as GET /api/v1/deployments/{id} response)
  }
}
```

**Error Codes**:
- `401`: Unauthorized
- `403`: Forbidden (not an admin)
- `404`: App not found
- `500`: Internal server error

---

## Important Notes

### NullString Serialization
Go's `sql.NullString` type may be serialized in two formats:
1. As a regular JSON string: `"value"` or `null`
2. As an object: `{ "String": "value", "Valid": true }` or `{ "String": "", "Valid": false }`

The frontend handles both formats. The backend should consistently use one format (recommended: standard JSON strings when valid, null when invalid).

### Array Responses
Endpoints that return arrays (`GET /api/apps`, `GET /api/v1/apps/{id}/deployments`, `GET /api/v1/apps/{id}/env`) must always return arrays, never null or undefined. The frontend converts null/undefined to empty arrays, but the backend should return proper arrays.

### Authentication & Authorization
- Bearer token authentication is required for most endpoints
- Token is passed via `Authorization: Bearer <token>` header
- 401 responses should trigger frontend logout/redirect to login
- 403 responses indicate insufficient permissions (e.g., not admin, or user doesn't own resource)
- Admin endpoints require both authentication and admin role

### Error Handling
- All errors should return JSON: `{ "error": "error message" }`
- HTTP status codes must match the documented error codes
- Frontend checks `response.ok` and expects JSON error format when false
- Network errors (connection failures) are handled separately by the frontend

### Timeouts
- `DELETE /api/v1/apps/{id}` has a 120-second (2 minute) timeout on the frontend
- Other endpoints use a 10-second default timeout
- Backend should handle long-running operations appropriately

### CORS
- Frontend sends requests with `credentials: 'omit'`
- Backend must enable CORS headers appropriately

---

## Summary

**Total Endpoints**: 22

**By Category**:
- Health: 1
- Authentication: 2
- User: 1
- Apps: 6
- Deployments: 2
- Environment Variables: 3
- Admin: 7

**By Method**:
- GET: 13
- POST: 7
- DELETE: 2
- PATCH: 1

This contract is extracted directly from the frontend codebase (`frontend/src/lib/api.ts` and `cms/src/lib/api.ts`) and must be implemented exactly as specified to ensure frontend compatibility.

