// API configuration for CMS
export const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080';

export const API_ENDPOINTS = {
  admin: {
    users: `${API_BASE_URL}/admin/users`,
    apps: `${API_BASE_URL}/admin/apps`,
  },
  auth: {
    login: `${API_BASE_URL}/api/auth/login`,
  },
} as const;

// Get auth token from localStorage
export function getAuthToken(): string | null {
  return localStorage.getItem('auth_token');
}

// Set auth token in localStorage
export function setAuthToken(token: string): void {
  localStorage.setItem('auth_token', token);
}

// Remove auth token from localStorage
export function removeAuthToken(): void {
  localStorage.removeItem('auth_token');
}

