-- Migration: Add UNIQUE constraint to slug column and backfill existing apps
-- This migration adds a UNIQUE constraint to the slug column and backfills
-- existing apps with generated slugs in the format: app-{short-id}

-- Step 1: Backfill existing apps with generated slugs
-- Generate slugs for apps that have NULL or empty slugs
-- Format: app-{first-8-chars-of-id}
UPDATE apps
SET slug = 'app-' || SUBSTRING(id::text FROM 1 FOR 8)
WHERE slug IS NULL OR slug = '';

-- Step 1b: Handle duplicate slugs by appending sequence numbers
-- For apps with duplicate slugs, append sequence numbers to make them unique
WITH duplicate_slugs AS (
    SELECT id, slug, ROW_NUMBER() OVER (PARTITION BY slug ORDER BY created_at) as rn
    FROM apps
    WHERE slug IN (
        SELECT slug
        FROM apps
        GROUP BY slug
        HAVING COUNT(*) > 1
    )
)
UPDATE apps
SET slug = apps.slug || '-' || (duplicate_slugs.rn - 1)::text
FROM duplicate_slugs
WHERE apps.id = duplicate_slugs.id AND duplicate_slugs.rn > 1;

-- Step 2: Ensure all apps have a slug (should not be needed after step 1, but safety check)
UPDATE apps
SET slug = 'app-' || SUBSTRING(id::text FROM 1 FOR 8)
WHERE slug IS NULL OR slug = '';

-- Step 3: Create a unique index on slug (this will also enforce uniqueness going forward)
-- Note: We use CREATE UNIQUE INDEX IF NOT EXISTS to be safe, but since we're migrating,
-- we'll drop any existing index first if needed
DROP INDEX IF EXISTS idx_apps_slug;

-- Step 4: Create unique index (replaces the non-unique index from initial schema)
CREATE UNIQUE INDEX idx_apps_slug ON apps(slug);

-- Step 5: Add NOT NULL constraint to slug column if it doesn't already exist
-- First check if constraint exists, if not add it
DO $$ 
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'apps_slug_not_null' 
        AND conrelid = 'apps'::regclass
    ) THEN
        ALTER TABLE apps ALTER COLUMN slug SET NOT NULL;
        -- Note: PostgreSQL doesn't support named NOT NULL constraints directly,
        -- so we'll just ensure it's NOT NULL
    END IF;
END $$;

