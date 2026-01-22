-- Add lemon_customer_id and cancel_at_period_end columns to subscriptions table
-- This migration adds support for customer portal and cancellation tracking

-- Add lemon_customer_id column (nullable - not all subscriptions have customer ID)
ALTER TABLE subscriptions 
ADD COLUMN IF NOT EXISTS lemon_customer_id VARCHAR(255);

-- Add cancel_at_period_end column (default false)
ALTER TABLE subscriptions 
ADD COLUMN IF NOT EXISTS cancel_at_period_end BOOLEAN DEFAULT false NOT NULL;

-- Create index for faster lookups by customer_id
CREATE INDEX IF NOT EXISTS idx_subscriptions_lemon_customer_id 
ON subscriptions(lemon_customer_id) 
WHERE lemon_customer_id IS NOT NULL;

-- Create index for cancel_at_period_end queries
CREATE INDEX IF NOT EXISTS idx_subscriptions_cancel_at_period_end 
ON subscriptions(cancel_at_period_end) 
WHERE cancel_at_period_end = true;

