-- Fix script: Rename subscription_id to lemon_subscription_id if needed
-- This is a one-time fix for databases where migration 000007 didn't complete properly
-- Run this manually if you see "column lemon_subscription_id does not exist" errors

DO $$
BEGIN
  -- Check if subscription_id column exists (old name)
  IF EXISTS (
    SELECT 1 
    FROM information_schema.columns 
    WHERE table_name = 'subscriptions' 
    AND column_name = 'subscription_id'
    AND table_schema = 'public'
  ) THEN
    -- Make it nullable first
    ALTER TABLE subscriptions ALTER COLUMN subscription_id DROP NOT NULL;
    -- Rename to lemon_subscription_id
    ALTER TABLE subscriptions RENAME COLUMN subscription_id TO lemon_subscription_id;
    RAISE NOTICE 'Renamed subscription_id to lemon_subscription_id';
  ELSE
    RAISE NOTICE 'Column subscription_id does not exist (already renamed or never existed)';
  END IF;
  
  -- Ensure lemon_subscription_id exists (create if it doesn't)
  IF NOT EXISTS (
    SELECT 1 
    FROM information_schema.columns 
    WHERE table_name = 'subscriptions' 
    AND column_name = 'lemon_subscription_id'
    AND table_schema = 'public'
  ) THEN
    -- Add the column if it doesn't exist
    ALTER TABLE subscriptions ADD COLUMN lemon_subscription_id VARCHAR(255);
    RAISE NOTICE 'Added lemon_subscription_id column';
  ELSE
    RAISE NOTICE 'Column lemon_subscription_id already exists';
  END IF;
END $$;

