-- Complete schema fix for subscriptions table
-- Run this to ensure all required columns exist

-- Step 1: Verify lemon_subscription_id exists
SELECT column_name, data_type, is_nullable
FROM information_schema.columns 
WHERE table_name = 'subscriptions'
  AND column_name IN ('lemon_subscription_id', 'subscription_id')
ORDER BY column_name;

-- Step 2: Add lemon_subscription_id if it doesn't exist (idempotent)
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns 
    WHERE table_name = 'subscriptions' 
    AND column_name = 'lemon_subscription_id'
  ) THEN
    IF EXISTS (
      SELECT 1 FROM information_schema.columns 
      WHERE table_name = 'subscriptions' 
      AND column_name = 'subscription_id'
    ) THEN
      ALTER TABLE subscriptions ALTER COLUMN subscription_id DROP NOT NULL;
      ALTER TABLE subscriptions RENAME COLUMN subscription_id TO lemon_subscription_id;
      RAISE NOTICE 'Renamed subscription_id to lemon_subscription_id';
    ELSE
      ALTER TABLE subscriptions ADD COLUMN lemon_subscription_id VARCHAR(255);
      RAISE NOTICE 'Added lemon_subscription_id column';
    END IF;
  ELSE
    RAISE NOTICE 'lemon_subscription_id already exists';
  END IF;
END $$;

-- Step 3: Add trial and resource limit columns
ALTER TABLE subscriptions
  ADD COLUMN IF NOT EXISTS trial_started_at TIMESTAMP,
  ADD COLUMN IF NOT EXISTS trial_ends_at TIMESTAMP,
  ADD COLUMN IF NOT EXISTS ram_limit_mb INTEGER,
  ADD COLUMN IF NOT EXISTS disk_limit_gb INTEGER;

-- Step 4: Set default values for existing rows
UPDATE subscriptions 
SET 
  ram_limit_mb = COALESCE(ram_limit_mb, 512),
  disk_limit_gb = COALESCE(disk_limit_gb, 5)
WHERE ram_limit_mb IS NULL OR disk_limit_gb IS NULL;

-- Step 5: Make columns NOT NULL (only if they're currently nullable and have no NULL values)
DO $$
BEGIN
  -- Check if ram_limit_mb has any NULL values
  IF NOT EXISTS (SELECT 1 FROM subscriptions WHERE ram_limit_mb IS NULL) THEN
    -- Only set NOT NULL if column is currently nullable
    IF EXISTS (
      SELECT 1 FROM information_schema.columns 
      WHERE table_name = 'subscriptions' 
      AND column_name = 'ram_limit_mb' 
      AND is_nullable = 'YES'
    ) THEN
      ALTER TABLE subscriptions ALTER COLUMN ram_limit_mb SET NOT NULL;
      RAISE NOTICE 'Set ram_limit_mb to NOT NULL';
    END IF;
  END IF;
  
  -- Check if disk_limit_gb has any NULL values
  IF NOT EXISTS (SELECT 1 FROM subscriptions WHERE disk_limit_gb IS NULL) THEN
    -- Only set NOT NULL if column is currently nullable
    IF EXISTS (
      SELECT 1 FROM information_schema.columns 
      WHERE table_name = 'subscriptions' 
      AND column_name = 'disk_limit_gb' 
      AND is_nullable = 'YES'
    ) THEN
      ALTER TABLE subscriptions ALTER COLUMN disk_limit_gb SET NOT NULL;
      RAISE NOTICE 'Set disk_limit_gb to NOT NULL';
    END IF;
  END IF;
END $$;

-- Step 6: Create indexes if they don't exist
CREATE UNIQUE INDEX IF NOT EXISTS idx_subscriptions_user_active_trial 
  ON subscriptions(user_id) 
  WHERE status IN ('trial', 'active');

CREATE INDEX IF NOT EXISTS idx_subscriptions_trial_status 
  ON subscriptions(status, trial_ends_at) 
  WHERE status = 'trial';

-- Step 7: Verify all columns exist
SELECT 
  column_name, 
  data_type, 
  is_nullable,
  column_default
FROM information_schema.columns 
WHERE table_name = 'subscriptions'
  AND column_name IN ('lemon_subscription_id', 'trial_started_at', 'trial_ends_at', 'ram_limit_mb', 'disk_limit_gb')
ORDER BY column_name;

