-- Add plan field to users table
-- Plan defaults to 'free' for existing users
ALTER TABLE users 
ADD COLUMN IF NOT EXISTS plan VARCHAR(50) DEFAULT 'free';

-- Create index on plan for faster queries
CREATE INDEX IF NOT EXISTS idx_users_plan ON users(plan);

-- Update existing users to have 'free' plan if NULL
UPDATE users SET plan = 'free' WHERE plan IS NULL;

