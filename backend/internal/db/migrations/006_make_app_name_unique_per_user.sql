-- Make app names unique per user instead of globally unique
-- This allows different users to have apps with the same name

-- First, drop the existing UNIQUE constraint on name
ALTER TABLE apps DROP CONSTRAINT IF EXISTS apps_name_key;

-- Add a composite UNIQUE constraint on (user_id, name)
-- This ensures that each user can have unique app names, but different users can have the same app name
-- Note: PostgreSQL treats NULL values as distinct in unique constraints, so apps with NULL user_id
-- can have duplicate names. Since authentication is now required, all new apps will have a user_id.
CREATE UNIQUE INDEX IF NOT EXISTS idx_apps_user_id_name_unique ON apps(user_id, name);

