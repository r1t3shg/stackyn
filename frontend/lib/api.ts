import { API_ENDPOINTS } from './config';
import type { App, Deployment, DeploymentLogs, CreateAppRequest, CreateAppResponse, EnvVar, CreateEnvVarRequest } from './types';

// Helper function to handle API responses
async function handleResponse<T>(response: Response): Promise<T> {
  if (!response.ok) {
    // Check if response is JSON before trying to parse
    const contentType = response.headers.get('content-type');
    let error: { error?: string } = { error: response.statusText };
    
    if (contentType && contentType.includes('application/json')) {
      try {
        error = await response.json();
      } catch {
        // If JSON parsing fails, use default error
        error = { error: response.statusText };
      }
    } else {
      // Response is not JSON (likely HTML error page)
      await response.text().catch(() => '');
      if (response.status === 404) {
        error = { error: `API endpoint not found. Please check that the backend server is running and the endpoint exists.` };
      } else if (response.status >= 500) {
        error = { error: `Server error (${response.status}). Please try again later.` };
      } else {
        error = { error: `HTTP ${response.status}: ${response.statusText}` };
      }
    }
    
    throw new Error(error.error || `HTTP error! status: ${response.status}`);
  }
  
  // Check content type before parsing JSON
  const contentType = response.headers.get('content-type');
  if (contentType && contentType.includes('application/json')) {
    return response.json();
  } else {
    // If response is not JSON, consume the body and throw error
    await response.text().catch(() => '');
    throw new Error(`Expected JSON response but received ${contentType || 'unknown content type'}`);
  }
}

// Helper to handle fetch errors with better messages
async function safeFetch(url: string, options?: RequestInit, requireAuth: boolean = true): Promise<Response> {
  // Log the request in development
  if (process.env.NODE_ENV === 'development') {
    console.log('Making API request to:', url, options ? `Method: ${options.method || 'GET'}` : '');
  }
  
  try {
    // Add Authorization header if auth is required
    const headers = new Headers(options?.headers);
    if (requireAuth && typeof window !== 'undefined') {
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
    });
    
    if (process.env.NODE_ENV === 'development') {
      console.log('API response status:', response.status, response.statusText);
    }
    
    // If unauthorized, clear auth
    if (response.status === 401 && requireAuth && typeof window !== 'undefined') {
      localStorage.removeItem('auth_token');
      localStorage.removeItem('auth_user');
    }
    
    return response;
  } catch (err) {
    console.error('Fetch error:', err);
    if (err instanceof TypeError && err.message === 'Failed to fetch') {
      throw new Error(`Cannot connect to backend API at ${url}. Make sure the backend server is running on port 8080 and CORS is enabled.`);
    }
    throw err;
  }
}

// Apps API
export const appsApi = {
  // List all apps
  list: async (): Promise<App[]> => {
    const response = await safeFetch(API_ENDPOINTS.apps);
    return handleResponse<App[]>(response);
  },

  // Get app by ID
  getById: async (id: string | number): Promise<App> => {
    const response = await safeFetch(`${API_ENDPOINTS.apps}/${id}`);
    return handleResponse<App>(response);
  },

  // Create a new app
  create: async (data: CreateAppRequest): Promise<CreateAppResponse> => {
    const response = await safeFetch(API_ENDPOINTS.apps, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(data),
    });
    return handleResponse<CreateAppResponse>(response);
  },

  // Delete an app
  delete: async (id: string | number): Promise<void> => {
    const response = await safeFetch(`${API_ENDPOINTS.apps}/${id}`, {
      method: 'DELETE',
    });
    if (!response.ok) {
      const error = await response.json().catch(() => ({ error: response.statusText }));
      throw new Error(error.error || `HTTP error! status: ${response.status}`);
    }
  },

  // Redeploy an app
  redeploy: async (id: string | number): Promise<CreateAppResponse> => {
    const response = await safeFetch(`${API_ENDPOINTS.apps}/${id}/redeploy`, {
      method: 'POST',
    });
    return handleResponse<CreateAppResponse>(response);
  },

  // Get deployments for an app
  getDeployments: async (id: string | number): Promise<Deployment[]> => {
    const response = await safeFetch(`${API_ENDPOINTS.apps}/${id}/deployments`);
    return handleResponse<Deployment[]>(response);
  },

  // Get environment variables for an app
  getEnvVars: async (id: string | number): Promise<EnvVar[]> => {
    const response = await safeFetch(`${API_ENDPOINTS.apps}/${id}/env`);
    const data = await handleResponse<EnvVar[]>(response);
    // Ensure we always return an array, never null or undefined
    return Array.isArray(data) ? data : [];
  },

  // Create or update an environment variable
  createEnvVar: async (id: string | number, data: CreateEnvVarRequest): Promise<EnvVar> => {
    const response = await safeFetch(`${API_ENDPOINTS.apps}/${id}/env`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(data),
    });
    return handleResponse<EnvVar>(response);
  },

  // Delete an environment variable
  deleteEnvVar: async (id: string | number, key: string): Promise<void> => {
    const response = await safeFetch(`${API_ENDPOINTS.apps}/${id}/env/${encodeURIComponent(key)}`, {
      method: 'DELETE',
    });
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
    const response = await safeFetch(`${API_ENDPOINTS.deployments}/${id}`);
    return handleResponse<Deployment>(response);
  },

  // Get deployment logs
  getLogs: async (id: string | number): Promise<DeploymentLogs> => {
    const response = await safeFetch(`${API_ENDPOINTS.deployments}/${id}/logs`);
    return handleResponse<DeploymentLogs>(response);
  },
};

// Health check
export const healthCheck = async (): Promise<{ status: string }> => {
  const response = await safeFetch(API_ENDPOINTS.health, undefined, false);
  return handleResponse<{ status: string }>(response);
};

// Billing API
export const billingApi = {
  // Create checkout session for a plan
  createCheckout: async (plan: 'starter' | 'pro'): Promise<{ checkout_url: string }> => {
    const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL || 'http://localhost:8080';
    const response = await safeFetch(`${API_BASE_URL}/api/billing/checkout`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ plan }),
    }, true);
    return handleResponse<{ checkout_url: string }>(response);
  },
};


