// Module mvp-be is the root module for the Stackyn PaaS backend.
// Stackyn is a Platform-as-a-Service that enables deployment of applications
// from Git repositories using Docker containers.
//
// This module contains:
//   - API server (cmd/api)
//   - Deployment worker (cmd/worker)
//   - Internal packages for database, Docker, Git operations
//
// Dependencies:
//   - github.com/go-chi/chi/v5: HTTP router
//   - github.com/lib/pq: PostgreSQL driver
//   - github.com/docker/docker: Docker API client
//
// Build:
//   go build -o bin/api ./cmd/api
//   go build -o bin/worker ./cmd/worker
module mvp-be

go 1.23
