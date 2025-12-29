-- Task state persistence table
CREATE TABLE task_states (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id VARCHAR(255) NOT NULL UNIQUE,
    task_type VARCHAR(50) NOT NULL,
    queue_name VARCHAR(50) NOT NULL,
    payload JSONB NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    retry_count INTEGER NOT NULL DEFAULT 0,
    max_retries INTEGER NOT NULL DEFAULT 3,
    error_message TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP,
    failed_at TIMESTAMP
);

CREATE INDEX idx_task_states_task_id ON task_states(task_id);
CREATE INDEX idx_task_states_task_type ON task_states(task_type);
CREATE INDEX idx_task_states_status ON task_states(status);
CREATE INDEX idx_task_states_queue_name ON task_states(queue_name);
CREATE INDEX idx_task_states_created_at ON task_states(created_at);

