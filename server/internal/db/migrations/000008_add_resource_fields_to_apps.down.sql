-- Rollback resource fields from apps table

DROP INDEX IF EXISTS idx_apps_user_resources;

ALTER TABLE apps
  DROP COLUMN IF EXISTS ram_mb,
  DROP COLUMN IF EXISTS disk_gb;

