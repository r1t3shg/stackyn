-- Add runtime_log column to deployments table
ALTER TABLE deployments 
ADD COLUMN IF NOT EXISTS runtime_log TEXT;

