-- Manual fix script for lemon_subscription_id column
-- Run this if the migration didn't create the column properly

DO $$
BEGIN
  -- Check if lemon_subscription_id column exists
  IF NOT EXISTS (
    SELECT 1 
    FROM information_schema.columns 
    WHERE table_name = 'subscriptions' 
    AND column_name = 'lemon_subscription_id'
  ) THEN
    -- Check if subscription_id exists and rename it
    IF EXISTS (
      SELECT 1 
      FROM information_schema.columns 
      WHERE table_name = 'subscriptions' 
      AND column_name = 'subscription_id'
    ) THEN
      -- Make subscription_id nullable first
      ALTER TABLE subscriptions ALTER COLUMN subscription_id DROP NOT NULL;
      -- Rename subscription_id to lemon_subscription_id
      ALTER TABLE subscriptions RENAME COLUMN subscription_id TO lemon_subscription_id;
    ELSE
      -- Neither exists, add lemon_subscription_id as nullable
      ALTER TABLE subscriptions ADD COLUMN lemon_subscription_id VARCHAR(255);
    END IF;
  END IF;
END $$;

-- Also ensure trial and resource limit columns exist
ALTER TABLE subscriptions
  ADD COLUMN IF NOT EXISTS trial_started_at TIMESTAMP,
  ADD COLUMN IF NOT EXISTS trial_ends_at TIMESTAMP,
  ADD COLUMN IF NOT EXISTS ram_limit_mb INTEGER,
  ADD COLUMN IF NOT EXISTS disk_limit_gb INTEGER;

-- Update existing subscriptions to have default limits if NULL
UPDATE subscriptions 
SET ram_limit_mb = COALESCE(ram_limit_mb, 512), 
    disk_limit_gb = COALESCE(disk_limit_gb, 5) 
WHERE ram_limit_mb IS NULL OR disk_limit_gb IS NULL;

-- Make columns NOT NULL after setting defaults (only if they're still nullable)
DO $$
BEGIN
  -- Check if ram_limit_mb is nullable and make it NOT NULL
  IF EXISTS (
    SELECT 1 
    FROM information_schema.columns 
    WHERE table_name = 'subscriptions' 
    AND column_name = 'ram_limit_mb'
    AND is_nullable = 'YES'
  ) THEN
    ALTER TABLE subscriptions ALTER COLUMN ram_limit_mb SET NOT NULL;
  END IF;
  
  -- Check if disk_limit_gb is nullable and make it NOT NULL
  IF EXISTS (
    SELECT 1 
    FROM information_schema.columns 
    WHERE table_name = 'subscriptions' 
    AND column_name = 'disk_limit_gb'
    AND is_nullable = 'YES'
  ) THEN
    ALTER TABLE subscriptions ALTER COLUMN disk_limit_gb SET NOT NULL;
  END IF;
END $$;

