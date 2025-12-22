// Package db provides database connection and management functionality.
// This file contains PostgreSQL advisory lock utilities for global coordination.

package db

import (
	"context"
	"database/sql"
	"fmt"
)

// GlobalBuildLockKey is the advisory lock key used to ensure only one build runs at a time.
// Using a fixed integer key (424242) for simplicity. PostgreSQL advisory locks use
// bigint keys, and this value is arbitrary but unique to our build lock.
const GlobalBuildLockKey int64 = 424242

// AcquireGlobalBuildLock attempts to acquire a PostgreSQL advisory lock for global build coordination.
// This ensures only one deployment build can run at any moment across the entire system,
// even when multiple worker instances are running.
//
// PostgreSQL advisory locks are ideal for this use case because:
//   - They are automatically released when the connection closes (crash-safe)
//   - They work across multiple database connections/processes
//   - They are non-blocking with pg_try_advisory_lock (no deadlocks)
//   - They don't require additional infrastructure (no Redis, no message queues)
//
// IMPORTANT: Advisory locks in PostgreSQL are session-scoped. To ensure proper lock release,
// we acquire a dedicated connection from the pool and use it for both lock acquisition and release.
// This connection is returned to the pool after release.
//
// Parameters:
//   - ctx: Context for cancellation/timeout
//   - db: Database connection pool to use for the lock
//
// Returns:
//   - release: Function to call when done (releases the lock). Always call this, even on error.
//   - ok: True if lock was acquired, false if lock is already held by another worker
//   - err: Database error if lock acquisition fails (connection error, etc.)
//
// Usage:
//   release, ok, err := AcquireGlobalBuildLock(ctx, db)
//   if err != nil {
//       return err
//   }
//   if !ok {
//       // Lock is busy, another worker is building
//       return nil
//   }
//   defer release() // Always release, even on panic
//   // ... do work ...
func AcquireGlobalBuildLock(ctx context.Context, db *sql.DB) (release func(), ok bool, err error) {
	// Get a dedicated connection from the pool for lock operations.
	// This ensures we use the same connection for acquire and release.
	// Advisory locks are session-scoped, so we need the same connection.
	conn, err := db.Conn(ctx)
	if err != nil {
		return func() {}, false, fmt.Errorf("failed to get connection for lock: %w", err)
	}

	// Use pg_try_advisory_lock for non-blocking lock acquisition.
	// Returns true if lock was acquired, false if lock is already held.
	// This prevents deadlocks and allows workers to skip and retry later.
	var acquired bool
	err = conn.QueryRowContext(ctx,
		"SELECT pg_try_advisory_lock($1)",
		GlobalBuildLockKey,
	).Scan(&acquired)
	if err != nil {
		// Database error (connection issue, etc.)
		conn.Close() // Return connection to pool
		return func() {}, false, fmt.Errorf("failed to acquire build lock: %w", err)
	}

	if !acquired {
		// Lock is already held by another worker
		conn.Close() // Return connection to pool
		return func() {}, false, nil
	}

	// Lock acquired successfully. Return a release function that unlocks and returns the connection.
	release = func() {
		// Use pg_advisory_unlock to release the lock on the same connection.
		_, unlockErr := conn.ExecContext(ctx, "SELECT pg_advisory_unlock($1)", GlobalBuildLockKey)
		if unlockErr != nil {
			// Log but don't fail - lock will be released when connection closes anyway
			// In production, you might want to log this to a monitoring system
			_ = unlockErr // Suppress unused variable warning
		}
		// Return the connection to the pool
		conn.Close()
	}

	return release, true, nil
}

