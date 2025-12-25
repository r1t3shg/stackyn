// API configuration
export const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080';

// Log the API base URL to help debug (always log in production too)
console.log('API Base URL:', API_BASE_URL);

export const API_ENDPOINTS = {
  apps: `${API_BASE_URL}/api/apps`, // Authenticated endpoint for user's apps
  appsV1: `${API_BASE_URL}/api/v1/apps`, // For creating/updating apps
  deployments: `${API_BASE_URL}/api/v1/deployments`,
  health: `${API_BASE_URL}/health`,
} as const;

// Get auth token from localStorage
export function getAuthToken(): string | null {
  return localStorage.getItem('auth_token');
}


