-- Add resource allocation fields to apps table
-- These fields track how much RAM and disk each app uses

ALTER TABLE apps
  ADD COLUMN ram_mb INTEGER DEFAULT 256,
  ADD COLUMN disk_gb INTEGER DEFAULT 1;

-- Update existing apps with default values
UPDATE apps 
SET ram_mb = 256, disk_gb = 1 
WHERE ram_mb IS NULL OR disk_gb IS NULL;

-- Make NOT NULL after setting defaults
ALTER TABLE apps
  ALTER COLUMN ram_mb SET NOT NULL,
  ALTER COLUMN disk_gb SET NOT NULL;

-- Create index for resource usage queries
CREATE INDEX idx_apps_user_resources ON apps(user_id, ram_mb, disk_gb);

