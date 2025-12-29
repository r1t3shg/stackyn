# Database Layer

This package provides database access using sqlc and pgx/v5.

## Structure

- `schema/` - SQL schema definitions (used by sqlc)
- `queries/` - SQL queries (used by sqlc to generate Go code)
- `migrations/` - Database migrations (golang-migrate)
- `db.go` - Database connection and transaction handling
- `migrate.go` - Migration runner

## Generating Code with sqlc

After modifying SQL queries or schema, regenerate the Go code:

```bash
# Install sqlc if not already installed
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

# Generate code
sqlc generate
```

This will generate:
- `models.go` - Go structs matching database tables
- `queries.go` - Query functions and interfaces
- `db.go` - Database connection helpers (already exists, won't overwrite)

## Running Migrations

Migrations are embedded in the binary and run automatically on startup. To run manually:

```go
import "stackyn/server/internal/db"

sqlDB, err := dbInstance.GetSQLDB()
if err != nil {
    log.Fatal(err)
}
defer sqlDB.Close()

if err := db.RunMigrations(sqlDB, logger); err != nil {
    log.Fatal(err)
}
```

## Adding New Queries

1. Add SQL query to appropriate file in `queries/` directory
2. Use sqlc query annotations:
   ```sql
   -- name: GetUserByID :one
   SELECT * FROM users WHERE id = $1;
   ```
3. Run `sqlc generate` to generate Go code
4. Use generated functions in your code

## Query Types

- `:one` - Returns a single row
- `:many` - Returns multiple rows
- `:exec` - Executes without returning rows
- `:execrows` - Executes and returns number of affected rows

## Transactions

Use `WithTx` for transactions:

```go
err := db.WithTx(ctx, func(q *db.Queries) error {
    // Use q for queries within transaction
    user, err := q.GetUserByID(ctx, userID)
    // ...
    return nil
})
```

