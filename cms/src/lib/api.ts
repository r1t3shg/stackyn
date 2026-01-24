import { API_ENDPOINTS, getAuthToken } from './config';
import type { User, UsersListResponse, App, AppsListResponse } from './types';

// Helper function to handle API responses
async function handleResponse<T>(response: Response): Promise<T> {
  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: response.statusText }));
    throw new Error(error.error || `HTTP error! status: ${response.status}`);
  }
  return response.json();
}

// Helper to handle fetch with auth
async function safeFetch(url: string, options?: RequestInit): Promise<Response> {
  const headers = new Headers(options?.headers);
  const token = getAuthToken();
  if (token) {
    headers.set('Authorization', `Bearer ${token}`);
  }
  if (!headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json');
  }

  const response = await fetch(url, {
    ...options,
    headers,
    credentials: 'omit',
  });

  // If unauthorized, redirect to login
  if (response.status === 401 || response.status === 403) {
    localStorage.removeItem('auth_token');
    // Use relative path since we're already under /cms base path
    window.location.href = '/cms/login';
  }

  return response;
}

// Admin Users API
export const adminUsersApi = {
  list: async (limit = 50, offset = 0, search = ''): Promise<UsersListResponse> => {
    const params = new URLSearchParams({
      limit: limit.toString(),
      offset: offset.toString(),
    });
    if (search) {
      params.append('search', search);
    }
    const response = await safeFetch(`${API_ENDPOINTS.admin.users}?${params}`);
    return handleResponse<UsersListResponse>(response);
  },

  getById: async (id: string): Promise<User> => {
    const response = await safeFetch(`${API_ENDPOINTS.admin.users}/${id}`);
    return handleResponse<User>(response);
  },

  updatePlan: async (id: string, plan: string): Promise<{ message: string; user_id: string; plan: string; user?: any; quota?: any }> => {
    const response = await safeFetch(`${API_ENDPOINTS.admin.users}/${id}/plan`, {
      method: 'PATCH',
      body: JSON.stringify({ plan }),
    });
    return handleResponse(response);
  },

  delete: async (id: string): Promise<{ message: string; user_id: string; apps_deleted?: number }> => {
    const response = await safeFetch(`${API_ENDPOINTS.admin.users}/${id}`, {
      method: 'DELETE',
    });
    return handleResponse(response);
  },
};

// Admin Apps API
export const adminAppsApi = {
  list: async (limit = 50, offset = 0): Promise<AppsListResponse> => {
    const params = new URLSearchParams({
      limit: limit.toString(),
      offset: offset.toString(),
    });
    const response = await safeFetch(`${API_ENDPOINTS.admin.apps}?${params}`);
    return handleResponse<AppsListResponse>(response);
  },

  stop: async (id: string): Promise<{ message: string; app_id: number; stopped_containers: number }> => {
    const response = await safeFetch(`${API_ENDPOINTS.admin.apps}/${id}/stop`, {
      method: 'POST',
    });
    return handleResponse(response);
  },

  start: async (id: string): Promise<{ message: string; app_id: number }> => {
    const response = await safeFetch(`${API_ENDPOINTS.admin.apps}/${id}/start`, {
      method: 'POST',
    });
    return handleResponse(response);
  },

  redeploy: async (id: string): Promise<{ message: string; app_id: number; deployment: any }> => {
    const response = await safeFetch(`${API_ENDPOINTS.admin.apps}/${id}/redeploy`, {
      method: 'POST',
    });
    return handleResponse(response);
  },

  delete: async (id: string): Promise<{ message: string; app_id: string }> => {
    const response = await safeFetch(`${API_ENDPOINTS.admin.apps}/${id}`, {
      method: 'DELETE',
    });
    return handleResponse(response);
  },
};

// Auth API
export const authApi = {
  login: async (email: string, password: string): Promise<{ user: { id: string; email: string }; token: string }> => {
    const response = await fetch(API_ENDPOINTS.auth.login, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ email, password }),
      credentials: 'omit',
    });
    return handleResponse(response);
  },
};

