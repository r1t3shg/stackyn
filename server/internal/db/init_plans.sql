-- Initialize default pricing plans
-- This script should be run after the database is created
-- It inserts default plans: starter and pro

-- Insert Starter Plan ($19/month)
INSERT INTO plans (name, display_name, price, max_ram_mb, max_disk_mb, max_apps, 
                   always_on, auto_deploy, health_checks, logs, zero_downtime, 
                   workers, priority_builds, manual_deploy_only)
VALUES ('starter', 'Starter', 1900, 512, 5120, 1,
        true, true, true, true, false,
        false, false, false)
ON CONFLICT (name) DO NOTHING;

-- Insert Pro Plan ($49/month)
INSERT INTO plans (name, display_name, price, max_ram_mb, max_disk_mb, max_apps,
                   always_on, auto_deploy, health_checks, logs, zero_downtime,
                   workers, priority_builds, manual_deploy_only)
VALUES ('pro', 'Pro', 4900, 2048, 20480, 3,
        true, true, true, true, true,
        false, true, false)
ON CONFLICT (name) DO NOTHING;

-- Note: Price is in cents (1900 = $19.00, 4900 = $49.00)
-- Starter: 1 app, 1 VPS, 512 MB RAM, 5 GB Disk
-- Pro: Up to 3 apps, 1 VPS, 2 GB RAM (shared), 20 GB Disk

