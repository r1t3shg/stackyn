// Types for CMS

export interface User {
  id: string;
  email: string;
  full_name?: string;
  company_name?: string;
  email_verified: boolean;
  plan: string;
  is_admin: boolean;
  created_at: string;
  updated_at: string;
  quota?: UserQuota;
  subscription?: UserSubscription;
}

export interface UserSubscription {
  status: string;
  plan: string;
  trial_started_at?: string;
  trial_ends_at?: string;
  ram_limit_mb: number;
  disk_limit_gb: number;
}

export interface UserQuota {
  plan_name: string;
  plan: Plan;
  app_count: number;
  total_ram_mb: number;
  total_disk_mb: number;
}

export interface Plan {
  name: string;
  display_name: string;
  price: number;
  max_ram_mb: number;
  max_disk_mb: number;
  max_apps: number;
  always_on: boolean;
  auto_deploy: boolean;
  health_checks: boolean;
  logs: boolean;
  zero_downtime: boolean;
  workers: boolean;
  priority_builds: boolean;
  manual_deploy_only: boolean;
}

export interface App {
  id: string;
  name: string;
  slug: string;
  status: string;
  url: string;
  repo_url: string;
  branch: string;
  created_at: string;
  updated_at: string;
  deployment_count?: number;
  latest_status?: string;
}

export interface UsersListResponse {
  users: User[];
  total: number;
  limit: number;
  offset: number;
}

export interface AppsListResponse {
  apps: App[];
  total: number;
  limit: number;
  offset: number;
}

