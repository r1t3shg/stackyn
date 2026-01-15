-- Remove billing fields from users table
DROP INDEX IF EXISTS idx_users_trial_expiration;
DROP INDEX IF EXISTS idx_users_billing_status;

ALTER TABLE users
  DROP COLUMN IF EXISTS subscription_id,
  DROP COLUMN IF EXISTS trial_ends_at,
  DROP COLUMN IF EXISTS trial_started_at,
  DROP COLUMN IF EXISTS plan,
  DROP COLUMN IF EXISTS billing_status;

