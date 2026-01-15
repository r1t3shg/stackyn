-- Add billing fields to users table for quick access
-- Note: subscriptions table remains the source of truth, but these fields
-- provide fast access without joins for billing checks

-- Add billing status field (trial | active | expired)
ALTER TABLE users
  ADD COLUMN IF NOT EXISTS billing_status VARCHAR(50) DEFAULT 'trial';

-- Add plan field (free_trial | starter | pro)
ALTER TABLE users
  ADD COLUMN IF NOT EXISTS plan VARCHAR(50);

-- Add trial dates
ALTER TABLE users
  ADD COLUMN IF NOT EXISTS trial_started_at TIMESTAMP,
  ADD COLUMN IF NOT EXISTS trial_ends_at TIMESTAMP;

-- Add subscription_id for quick lookup
ALTER TABLE users
  ADD COLUMN IF NOT EXISTS subscription_id VARCHAR(255);

-- Create index for billing status checks (used by RequireActiveBilling)
CREATE INDEX IF NOT EXISTS idx_users_billing_status 
  ON users(billing_status) 
  WHERE billing_status IN ('trial', 'active');

-- Create index for trial expiration checks (used by background worker)
CREATE INDEX IF NOT EXISTS idx_users_trial_expiration 
  ON users(billing_status, trial_ends_at) 
  WHERE billing_status = 'trial' AND trial_ends_at IS NOT NULL;

-- Update existing users: if they have an active/trial subscription, sync the fields
-- This migration is idempotent - safe to run multiple times
UPDATE users u
SET 
  billing_status = COALESCE(
    (SELECT status FROM subscriptions s WHERE s.user_id = u.id AND s.status IN ('trial', 'active') ORDER BY s.created_at DESC LIMIT 1),
    'expired'
  ),
  plan = COALESCE(
    (SELECT plan FROM subscriptions s WHERE s.user_id = u.id AND s.status IN ('trial', 'active') ORDER BY s.created_at DESC LIMIT 1),
    NULL
  ),
  trial_started_at = (SELECT trial_started_at FROM subscriptions s WHERE s.user_id = u.id AND s.status IN ('trial', 'active') ORDER BY s.created_at DESC LIMIT 1),
  trial_ends_at = (SELECT trial_ends_at FROM subscriptions s WHERE s.user_id = u.id AND s.status IN ('trial', 'active') ORDER BY s.created_at DESC LIMIT 1),
  subscription_id = (SELECT lemon_subscription_id FROM subscriptions s WHERE s.user_id = u.id AND s.status IN ('trial', 'active') ORDER BY s.created_at DESC LIMIT 1)
WHERE EXISTS (
  SELECT 1 FROM subscriptions s WHERE s.user_id = u.id
);

-- For users without subscriptions, set default trial status
UPDATE users
SET billing_status = 'trial'
WHERE billing_status IS NULL;

