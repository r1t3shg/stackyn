import { API_ENDPOINTS, API_BASE_URL } from './config';
import type { App, Deployment, DeploymentLogs, CreateAppRequest, CreateAppResponse, EnvVar, CreateEnvVarRequest, UserProfile } from './types';

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
      // Consume the response body to avoid memory leaks
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
async function safeFetch(url: string, options?: RequestInit, requireAuth: boolean = true, timeout: number = 10000): Promise<Response> {
  // Always log requests for debugging
  console.log('Making API request to:', url, options ? `Method: ${options.method || 'GET'}` : '');
  
  try {
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), timeout);
    
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
    
    // If unauthorized, clear auth but don't redirect automatically
    // Let the calling component handle the error and navigation using React Router
    // This prevents page refreshes and allows proper error handling
    if (response.status === 401 && requireAuth) {
      localStorage.removeItem('auth_token');
      localStorage.removeItem('auth_user');
      // Dispatch a custom event so components can listen and handle navigation
      window.dispatchEvent(new CustomEvent('auth:unauthorized'));
    }
    
    return response;
  } catch (err) {
    console.error('Fetch error:', err);
    if (err instanceof TypeError && err.message === 'Failed to fetch') {
      throw new Error(`Cannot connect to backend API at ${url}. Make sure the backend server is running and accessible.`);
    }
    if (err instanceof Error && err.name === 'AbortError') {
      throw new Error(`Request to ${url} timed out after ${timeout / 1000} seconds.`);
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

  // Delete an app (longer timeout for Docker cleanup operations)
  delete: async (id: string | number): Promise<void> => {
    const response = await safeFetch(`${API_ENDPOINTS.appsV1}/${id}`, {
      method: 'DELETE',
    }, true, 120000); // 2 minute timeout for delete operations
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

// Auth API - OTP and Firebase Auth signup flow
export const authApi = {
  // Send OTP to email
  sendOTP: async (email: string): Promise<{ message: string; otp?: string }> => {
    const url = `${API_BASE_URL}/api/auth/send-otp`;
    console.log('Sending OTP request to:', url);
    
    try {
      const response = await safeFetch(url, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ email }),
      }, false);
      return handleResponse<{ message: string; otp?: string }>(response);
    } catch (err) {
      console.error('Error sending OTP:', err);
      // Provide more helpful error message
      if (err instanceof Error) {
        if (err.message.includes('Failed to fetch') || err.message.includes('NetworkError')) {
          throw new Error(`Cannot connect to API server at ${API_BASE_URL}. Please check that the backend is running and accessible.`);
        }
        throw err;
      }
      throw new Error('Failed to send OTP. Please try again.');
    }
  },

  // Verify OTP and get JWT token
  verifyOTP: async (email: string, otp: string, password?: string): Promise<{ token: string; user: { id: string; email: string; full_name?: string; company_name?: string } }> => {
    const body: { email: string; otp: string; password?: string } = { email, otp };
    if (password) {
      body.password = password;
    }
    const response = await safeFetch(`${API_BASE_URL}/api/auth/verify-otp`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(body),
    }, false);
    return handleResponse<{ token: string; user: { id: string; email: string; full_name?: string; company_name?: string } }>(response);
  },

  // Verify Firebase token (legacy)
  verifyToken: async (idToken: string): Promise<{ uid: string; email: string; email_verified: boolean }> => {
    const response = await safeFetch(`${API_BASE_URL}/api/auth/verify-token`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ id_token: idToken }),
    }, false);
    return handleResponse<{ uid: string; email: string; email_verified: boolean }>(response);
  },
};

// User API
export const userApi = {
  // Get current user profile with plan and quota
  getProfile: async (): Promise<UserProfile> => {
    const response = await safeFetch(`${API_BASE_URL}/api/user/me`, undefined, true);
    return handleResponse<UserProfile>(response);
  },
};


