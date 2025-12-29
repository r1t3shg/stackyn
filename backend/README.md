# Stackyn Backend API

Go 1.22 backend API skeleton with mocked responses for frontend integration.

## Overview

This is a minimal backend implementation that provides API skeletons for all endpoints required by the frontend. All endpoints return mocked data matching the exact frontend schema expectations.

## Running

```bash
# Build
go build -o backend.exe

# Run
./backend.exe
```

The server will start on `http://localhost:8080`

## API Endpoints

All endpoints are documented in `API_CONTRACT.md`. This implementation provides:

- **Health Check**: `GET /health`
- **Authentication**: `POST /api/auth/verify-token`, `POST /api/auth/login`
- **User**: `GET /api/user/me`
- **Apps**: 6 endpoints for app management
- **Deployments**: 2 endpoints for deployment management
- **Environment Variables**: 3 endpoints for env var management
- **Admin**: 7 endpoints for admin operations

## Authentication

All endpoints (except health check and auth endpoints) require Bearer token authentication via the `Authorization` header:
```
Authorization: Bearer <token>
```

The mock implementation accepts any Bearer token.

## CORS

CORS is enabled for all origins to allow frontend integration during development.

## Response Format

All responses match the exact schema expected by the frontend:
- Success responses: JSON objects/arrays matching TypeScript interfaces
- Error responses: `{ "error": "error message" }` with appropriate HTTP status codes

## Notes

- This is a **skeleton implementation** - all data is mocked
- No database or persistent storage
- Authentication is mocked (accepts any Bearer token)
- All responses return mock data matching frontend expectations
- Designed for frontend end-to-end testing and development

