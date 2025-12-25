-- Create environment_variables table
CREATE TABLE IF NOT EXISTS environment_variables (
    id SERIAL PRIMARY KEY,
    app_id INTEGER NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
    key VARCHAR(255) NOT NULL,
    value TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(app_id, key)
);

-- Create indexes for faster queries
CREATE INDEX IF NOT EXISTS idx_env_vars_app_id ON environment_variables(app_id);
CREATE INDEX IF NOT EXISTS idx_env_vars_key ON environment_variables(key);

