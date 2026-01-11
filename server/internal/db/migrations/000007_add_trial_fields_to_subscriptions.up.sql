-- Add trial and resource limit fields to subscriptions table
-- This migration adds support for 7-day free trials and resource limits

ALTER TABLE subscriptions
  -- Make subscription_id nullable (trials don't have external subscription ID initially)
  ALTER COLUMN subscription_id DROP NOT NULL,
  -- Rename subscription_id to lemon_subscription_id for clarity
  RENAME COLUMN subscription_id TO lemon_subscription_id;

-- Add trial-related fields
ALTER TABLE subscriptions
  ADD COLUMN trial_started_at TIMESTAMP,
  ADD COLUMN trial_ends_at TIMESTAMP;

-- Add resource limit fields (limits are enforced per subscription)
-- First add columns as nullable
ALTER TABLE subscriptions
  ADD COLUMN IF NOT EXISTS ram_limit_mb INTEGER,
  ADD COLUMN IF NOT EXISTS disk_limit_gb INTEGER;

-- Update constraint: Only one active or trial subscription per user
-- This constraint ensures data integrity
-- Note: PostgreSQL partial unique index ensures only one active/trial subscription per user
CREATE UNIQUE INDEX IF NOT EXISTS idx_subscriptions_user_active_trial 
  ON subscriptions(user_id) 
  WHERE status IN ('trial', 'active');

-- Create index for trial lifecycle management (cron job queries)
CREATE INDEX idx_subscriptions_trial_status 
  ON subscriptions(status, trial_ends_at) 
  WHERE status = 'trial';

-- Update existing subscriptions to have default limits if NULL
UPDATE subscriptions 
SET ram_limit_mb = 512, disk_limit_gb = 5 
WHERE ram_limit_mb IS NULL OR disk_limit_gb IS NULL;

-- Now make columns NOT NULL after setting defaults
ALTER TABLE subscriptions
  ALTER COLUMN ram_limit_mb SET NOT NULL,
  ALTER COLUMN disk_limit_gb SET NOT NULL;

