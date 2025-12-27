-- Add unique constraint on subdomain column
-- This ensures each deployment gets a unique subdomain
-- First, remove any duplicate subdomains (keep the most recent one)
UPDATE deployments d1
SET subdomain = NULL
WHERE d1.id NOT IN (
    SELECT DISTINCT ON (subdomain) id
    FROM deployments
    WHERE subdomain IS NOT NULL
    ORDER BY subdomain, created_at DESC
);

-- Now add the unique constraint
-- Note: This will allow NULL values (multiple NULLs are allowed in unique constraints)
ALTER TABLE deployments 
ADD CONSTRAINT unique_subdomain UNIQUE (subdomain);

-- Create index on subdomain for faster lookups (if not already exists)
CREATE INDEX IF NOT EXISTS idx_deployments_subdomain ON deployments(subdomain);

