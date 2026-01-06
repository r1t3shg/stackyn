-- Initialize default pricing plans
-- This script should be run after the database is created
-- It inserts default plans: free, pro, and enterprise

-- Insert Free Plan
INSERT INTO plans (name, display_name, price, max_ram_mb, max_disk_mb, max_apps, 
                   always_on, auto_deploy, health_checks, logs, zero_downtime, 
                   workers, priority_builds, manual_deploy_only)
VALUES ('free', 'Free', 0, 1024, 5120, 3,
        false, false, true, true, false,
        false, false, false)
ON CONFLICT (name) DO NOTHING;

-- Insert Pro Plan
INSERT INTO plans (name, display_name, price, max_ram_mb, max_disk_mb, max_apps,
                   always_on, auto_deploy, health_checks, logs, zero_downtime,
                   workers, priority_builds, manual_deploy_only)
VALUES ('pro', 'Pro', 2000, 4096, 20480, 10,
        true, true, true, true, true,
        false, true, false)
ON CONFLICT (name) DO NOTHING;

-- Insert Enterprise Plan
INSERT INTO plans (name, display_name, price, max_ram_mb, max_disk_mb, max_apps,
                   always_on, auto_deploy, health_checks, logs, zero_downtime,
                   workers, priority_builds, manual_deploy_only)
VALUES ('enterprise', 'Enterprise', 10000, 16384, 102400, 50,
        true, true, true, true, true,
        true, true, false)
ON CONFLICT (name) DO NOTHING;

-- Note: Price is in cents (2000 = $20.00, 10000 = $100.00)

