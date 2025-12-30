-- Users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL UNIQUE,
    full_name VARCHAR(255),
    company_name VARCHAR(255),
    email_verified BOOLEAN NOT NULL DEFAULT false,
    plan_id UUID REFERENCES plans(id),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_plan_id ON users(plan_id);

-- Plans table
CREATE TABLE plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(50) NOT NULL UNIQUE,
    display_name VARCHAR(100) NOT NULL,
    price INTEGER NOT NULL DEFAULT 0,
    max_ram_mb INTEGER NOT NULL DEFAULT 512,
    max_disk_mb INTEGER NOT NULL DEFAULT 1024,
    max_apps INTEGER NOT NULL DEFAULT 1,
    always_on BOOLEAN NOT NULL DEFAULT false,
    auto_deploy BOOLEAN NOT NULL DEFAULT false,
    health_checks BOOLEAN NOT NULL DEFAULT false,
    logs BOOLEAN NOT NULL DEFAULT false,
    zero_downtime BOOLEAN NOT NULL DEFAULT false,
    workers BOOLEAN NOT NULL DEFAULT false,
    priority_builds BOOLEAN NOT NULL DEFAULT false,
    manual_deploy_only BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Apps table
CREATE TABLE apps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    url VARCHAR(500),
    repo_url VARCHAR(500) NOT NULL,
    branch VARCHAR(255) NOT NULL DEFAULT 'main',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, name)
);

CREATE INDEX idx_apps_user_id ON apps(user_id);
CREATE INDEX idx_apps_slug ON apps(slug);
CREATE INDEX idx_apps_status ON apps(status);

-- Build jobs table
CREATE TABLE build_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id UUID NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    build_log TEXT,
    error_message TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_build_jobs_app_id ON build_jobs(app_id);
CREATE INDEX idx_build_jobs_status ON build_jobs(status);

-- Deployments table
CREATE TABLE deployments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id UUID NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
    build_job_id UUID REFERENCES build_jobs(id) ON DELETE SET NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    image_name VARCHAR(500),
    container_id VARCHAR(255),
    subdomain VARCHAR(255),
    build_log TEXT,
    runtime_log TEXT,
    error_message TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_deployments_app_id ON deployments(app_id);
CREATE INDEX idx_deployments_build_job_id ON deployments(build_job_id);
CREATE INDEX idx_deployments_status ON deployments(status);
CREATE INDEX idx_deployments_subdomain ON deployments(subdomain);

-- Runtime instances table
CREATE TABLE runtime_instances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    deployment_id UUID NOT NULL REFERENCES deployments(id) ON DELETE CASCADE,
    container_id VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'running',
    memory_mb INTEGER,
    cpu INTEGER,
    disk_gb INTEGER,
    memory_usage_mb INTEGER,
    memory_usage_percent DECIMAL(5,2),
    disk_usage_gb DECIMAL(10,2),
    disk_usage_percent DECIMAL(5,2),
    restart_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_runtime_instances_deployment_id ON runtime_instances(deployment_id);
CREATE INDEX idx_runtime_instances_container_id ON runtime_instances(container_id);
CREATE INDEX idx_runtime_instances_status ON runtime_instances(status);

-- Environment variables table
CREATE TABLE env_vars (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id UUID NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
    key VARCHAR(255) NOT NULL,
    value TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(app_id, key)
);

CREATE INDEX idx_env_vars_app_id ON env_vars(app_id);

