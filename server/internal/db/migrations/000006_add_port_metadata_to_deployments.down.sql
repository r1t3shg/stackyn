-- Remove port metadata columns from deployments table

DROP INDEX IF EXISTS idx_deployments_port_source;

ALTER TABLE deployments 
DROP COLUMN IF EXISTS port_warning,
DROP COLUMN IF EXISTS port_source,
DROP COLUMN IF EXISTS runtime_port,
DROP COLUMN IF EXISTS detected_port;

