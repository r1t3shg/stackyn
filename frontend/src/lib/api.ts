import { API_ENDPOINTS } from './config';
import type { App, Deployment, DeploymentLogs, CreateAppRequest, CreateAppResponse, EnvVar, CreateEnvVarRequest } from './types';

// Helper function to handle API responses
async function handleResponse<T>(response: Response): Promise<T> {
  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: response.statusText }));
    throw new Error(error.error || `HTTP error! status: ${response.status}`);
  }
  return response.json();
}

// Helper to handle fetch errors with better messages
async function safeFetch(url: string, options?: RequestInit, requireAuth: boolean = true): Promise<Response> {
  // Always log requests for debugging
  console.log('Making API request to:', url, options ? `Method: ${options.method || 'GET'}` : '');
  
  try {
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), 10000); // 10 second timeout
    
    // Add Authorization header if auth is required
    const headers = new Headers(options?.headers);
    if (requireAuth) {
      const token = localStorage.getItem('auth_token');
      if (token) {
        headers.set('Authorization', `Bearer ${token}`);
      }
    }
    
    const response = await fetch(url, {
      ...options,
      headers,
      // Ensure credentials are not sent (for CORS)
      credentials: 'omit',
      signal: controller.signal,
    });
    
    clearTimeout(timeoutId);
    
    console.log('API response status:', response.status, response.statusText);
    
    // If unauthorized, clear auth and redirect to login
    if (response.status === 401 && requireAuth) {
      localStorage.removeItem('auth_token');
      localStorage.removeItem('auth_user');
      window.location.href = '/login';
    }
    
    return response;
  } catch (err) {
    console.error('Fetch error:', err);
    if (err instanceof TypeError && err.message === 'Failed to fetch') {
      throw new Error(`Cannot connect to backend API at ${url}. Make sure the backend server is running and accessible.`);
    }
    if (err instanceof Error && err.name === 'AbortError') {
      throw new Error(`Request to ${url} timed out after 10 seconds.`);
    }
    throw err;
  }
}

// Apps API
export const appsApi = {
  // List all apps for authenticated user
  list: async (): Promise<App[]> => {
    const response = await safeFetch(API_ENDPOINTS.apps, undefined, true);
    const data = await handleResponse<App[]>(response);
    // Ensure we always return an array
    return Array.isArray(data) ? data : [];
  },

  // Get app by ID
  getById: async (id: string | number): Promise<App> => {
    const response = await safeFetch(`${API_ENDPOINTS.appsV1}/${id}`, undefined, true);
    return handleResponse<App>(response);
  },

  // Create a new app
  create: async (data: CreateAppRequest): Promise<CreateAppResponse> => {
    const response = await safeFetch(API_ENDPOINTS.appsV1, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(data),
    }, true);
    return handleResponse<CreateAppResponse>(response);
  },

  // Delete an app
  delete: async (id: string | number): Promise<void> => {
    const response = await safeFetch(`${API_ENDPOINTS.appsV1}/${id}`, {
      method: 'DELETE',
    }, true);
    if (!response.ok) {
      const error = await response.json().catch(() => ({ error: response.statusText }));
      throw new Error(error.error || `HTTP error! status: ${response.status}`);
    }
  },

  // Redeploy an app
  redeploy: async (id: string | number): Promise<CreateAppResponse> => {
    const response = await safeFetch(`${API_ENDPOINTS.appsV1}/${id}/redeploy`, {
      method: 'POST',
    }, true);
    return handleResponse<CreateAppResponse>(response);
  },

  // Get deployments for an app
  getDeployments: async (id: string | number): Promise<Deployment[]> => {
    const response = await safeFetch(`${API_ENDPOINTS.appsV1}/${id}/deployments`, undefined, true);
    const data = await handleResponse<Deployment[]>(response);
    // Ensure we always return an array
    return Array.isArray(data) ? data : [];
  },

  // Get environment variables for an app
  getEnvVars: async (id: string | number): Promise<EnvVar[]> => {
    const response = await safeFetch(`${API_ENDPOINTS.appsV1}/${id}/env`, undefined, true);
    const data = await handleResponse<EnvVar[]>(response);
    // Ensure we always return an array, never null or undefined
    return Array.isArray(data) ? data : [];
  },

  // Create or update an environment variable
  createEnvVar: async (id: string | number, data: CreateEnvVarRequest): Promise<EnvVar> => {
    const response = await safeFetch(`${API_ENDPOINTS.appsV1}/${id}/env`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(data),
    }, true);
    return handleResponse<EnvVar>(response);
  },

  // Delete an environment variable
  deleteEnvVar: async (id: string | number, key: string): Promise<void> => {
    const response = await safeFetch(`${API_ENDPOINTS.appsV1}/${id}/env/${encodeURIComponent(key)}`, {
      method: 'DELETE',
    }, true);
    if (!response.ok) {
      const error = await response.json().catch(() => ({ error: response.statusText }));
      throw new Error(error.error || `HTTP error! status: ${response.status}`);
    }
  },
};

// Deployments API
export const deploymentsApi = {
  // Get deployment by ID
  getById: async (id: string | number): Promise<Deployment> => {
    const response = await safeFetch(`${API_ENDPOINTS.deployments}/${id}`, undefined, true);
    return handleResponse<Deployment>(response);
  },

  // Get deployment logs
  getLogs: async (id: string | number): Promise<DeploymentLogs> => {
    const response = await safeFetch(`${API_ENDPOINTS.deployments}/${id}/logs`, undefined, true);
    return handleResponse<DeploymentLogs>(response);
  },
};

// Health check (no auth required)
export const healthCheck = async (): Promise<{ status: string }> => {
  const response = await safeFetch(API_ENDPOINTS.health, undefined, false);
  return handleResponse<{ status: string }>(response);
};


