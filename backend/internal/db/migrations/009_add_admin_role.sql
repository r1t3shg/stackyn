-- Add is_admin column to users table
-- Defaults to false for existing users
ALTER TABLE users 
ADD COLUMN IF NOT EXISTS is_admin BOOLEAN DEFAULT FALSE;

-- Create index on is_admin for faster queries
CREATE INDEX IF NOT EXISTS idx_users_is_admin ON users(is_admin);

-- Update existing users to have is_admin = false if NULL
UPDATE users SET is_admin = FALSE WHERE is_admin IS NULL;

