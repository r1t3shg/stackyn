-- Remove subscriptions table
DROP INDEX IF EXISTS idx_subscriptions_plan;
DROP INDEX IF EXISTS idx_subscriptions_status;
DROP INDEX IF EXISTS idx_subscriptions_subscription_id;
DROP INDEX IF EXISTS idx_subscriptions_user_id;
DROP TABLE IF EXISTS subscriptions;

