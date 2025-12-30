-- Add subscriptions table for billing
CREATE TABLE subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    subscription_id VARCHAR(255) NOT NULL UNIQUE, -- External subscription ID (e.g., Lemon Squeezy)
    plan VARCHAR(50) NOT NULL, -- Plan name/identifier
    status VARCHAR(50) NOT NULL DEFAULT 'active', -- active, inactive, canceled, expired, past_due, trialing
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, subscription_id)
);

CREATE INDEX idx_subscriptions_user_id ON subscriptions(user_id);
CREATE INDEX idx_subscriptions_subscription_id ON subscriptions(subscription_id);
CREATE INDEX idx_subscriptions_status ON subscriptions(status);
CREATE INDEX idx_subscriptions_plan ON subscriptions(plan);

