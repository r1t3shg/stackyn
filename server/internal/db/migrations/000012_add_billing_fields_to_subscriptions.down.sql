-- Rollback: Remove billing fields from subscriptions table

DROP INDEX IF EXISTS idx_subscriptions_cancel_at_period_end;
DROP INDEX IF EXISTS idx_subscriptions_lemon_customer_id;

ALTER TABLE subscriptions 
DROP COLUMN IF EXISTS cancel_at_period_end;

ALTER TABLE subscriptions 
DROP COLUMN IF EXISTS lemon_customer_id;

