-- Rollback trial fields from subscriptions table

-- Remove indexes
DROP INDEX IF EXISTS idx_subscriptions_trial_status;
DROP INDEX IF EXISTS idx_subscriptions_user_active_trial;

-- Remove columns
ALTER TABLE subscriptions
  DROP COLUMN IF EXISTS trial_started_at,
  DROP COLUMN IF EXISTS trial_ends_at,
  DROP COLUMN IF EXISTS ram_limit_mb,
  DROP COLUMN IF EXISTS disk_limit_gb;

-- Rename back to subscription_id and make it NOT NULL
ALTER TABLE subscriptions
  RENAME COLUMN lemon_subscription_id TO subscription_id;

ALTER TABLE subscriptions
  ALTER COLUMN subscription_id SET NOT NULL;

