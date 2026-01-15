-- Log storage table for Postgres persistence (chunked)
CREATE TABLE app_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id TEXT NOT NULL,
    build_job_id TEXT,
    deployment_id TEXT,
    log_type TEXT NOT NULL, -- 'build' or 'runtime'
    chunk_index INT NOT NULL DEFAULT 0, -- For chunking large logs
    content TEXT NOT NULL,
    size_bytes BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Indexes for efficient querying
CREATE INDEX idx_app_logs_app_id ON app_logs (app_id);
CREATE INDEX idx_app_logs_build_job_id ON app_logs (build_job_id) WHERE build_job_id IS NOT NULL;
CREATE INDEX idx_app_logs_deployment_id ON app_logs (deployment_id) WHERE deployment_id IS NOT NULL;
CREATE INDEX idx_app_logs_log_type ON app_logs (log_type);
CREATE INDEX idx_app_logs_created_at ON app_logs (created_at DESC);

-- Index for storage limit calculations
CREATE INDEX idx_app_logs_app_id_size ON app_logs (app_id, size_bytes);

-- Function to calculate total storage per app
CREATE OR REPLACE FUNCTION get_app_log_storage_size(p_app_id TEXT)
RETURNS BIGINT AS $$
    SELECT COALESCE(SUM(size_bytes), 0)
    FROM app_logs
    WHERE app_id = p_app_id;
$$ LANGUAGE SQL;

