.PHONY: sqlc migrate build

# Generate code from SQL queries
sqlc:
	sqlc generate

# Run migrations (requires DATABASE_URL env var)
migrate:
	go run -tags migration ./cmd/migrate

# Build all binaries
build:
	go build -o bin/api.exe ./cmd/api
	go build -o bin/build-worker.exe ./cmd/build-worker
	go build -o bin/deploy-worker.exe ./cmd/deploy-worker
	go build -o bin/cleanup-worker.exe ./cmd/cleanup-worker

# Install sqlc
install-sqlc:
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

