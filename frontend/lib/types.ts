// Type definitions for the API models

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
  deployment?: {
    active_deployment_id: string | null;
    last_deployed_at: string | null;
    state: string;
  };
}

// Go's sql.NullString serializes as {String: "...", Valid: true}
interface NullString {
  String: string;
  Valid: boolean;
}

export interface Deployment {
  id: number;
  app_id: number;
  status: 'pending' | 'building' | 'running' | 'failed' | 'stopped';
  image_name?: string | null | NullString;
  container_id?: string | null | NullString;
  subdomain?: string | null | NullString;
  build_log?: string | null | NullString;
  runtime_log?: string | null | NullString;
  error_message?: string | null | NullString;
  created_at: string;
  updated_at: string;
}

// Helper function to extract string value from sql.NullString or regular string
export function extractString(value: string | null | NullString | undefined): string | null {
  if (!value) return null;
  if (typeof value === 'string') return value;
  if (typeof value === 'object' && 'Valid' in value && 'String' in value) {
    return value.Valid ? value.String : null;
  }
  return null;
}

export interface DeploymentLogs {
  deployment_id: number;
  status: string;
  build_log?: string | null;
  runtime_log?: string | null;
  error_message?: string | null;
}

export interface CreateAppRequest {
  name: string;
  repo_url: string;
  branch: string;
}

export interface CreateAppResponse {
  app: App;
  deployment: Deployment;
  error?: string;
}

export interface EnvVar {
  id: number;
  app_id: number;
  key: string;
  value: string;
  created_at: string;
  updated_at: string;
}

export interface CreateEnvVarRequest {
  key: string;
  value: string;
}


