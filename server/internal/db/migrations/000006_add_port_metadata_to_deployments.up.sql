-- Add port metadata columns to deployments table
-- This allows tracking detected ports, runtime ports, and port source information

ALTER TABLE deployments 
ADD COLUMN IF NOT EXISTS detected_port INTEGER,
ADD COLUMN IF NOT EXISTS runtime_port INTEGER DEFAULT 8080,
ADD COLUMN IF NOT EXISTS port_source VARCHAR(50) DEFAULT 'env',
ADD COLUMN IF NOT EXISTS port_warning TEXT;

-- Add comment for documentation
COMMENT ON COLUMN deployments.detected_port IS 'Port number detected in source code (NULL if using env vars)';
COMMENT ON COLUMN deployments.runtime_port IS 'Port number used at runtime (always 8080 for Stackyn)';
COMMENT ON COLUMN deployments.port_source IS 'Source of port: hardcoded, env, or none';
COMMENT ON COLUMN deployments.port_warning IS 'Warning message if hardcoded port detected';

-- Create index for queries filtering by port_source
CREATE INDEX IF NOT EXISTS idx_deployments_port_source ON deployments(port_source);

