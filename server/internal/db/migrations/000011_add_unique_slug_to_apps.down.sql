-- Migration Rollback: Remove UNIQUE constraint from slug column

-- Step 1: Drop unique index
DROP INDEX IF EXISTS idx_apps_slug;

-- Step 2: Recreate non-unique index (as it was in original schema)
CREATE INDEX idx_apps_slug ON apps(slug);

-- Note: We don't remove NOT NULL constraint on slug as it was intended to be NOT NULL
-- in the original schema (even though the constraint wasn't explicitly enforced)

