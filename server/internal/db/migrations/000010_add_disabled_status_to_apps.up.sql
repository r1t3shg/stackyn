-- Add 'disabled' status to apps table
-- This status is used when billing expires and apps are stopped

-- Note: The status column already exists as VARCHAR(50), so we just need to ensure
-- the application code handles 'disabled' as a valid status value.
-- No schema change needed, but this migration documents the new status.

-- Add comment to document the status values
COMMENT ON COLUMN apps.status IS 'App status: pending, building, deploying, running, stopped, failed, disabled (disabled = billing expired)';

