// Helper function to ensure API base URL has a protocol
function normalizeApiBaseUrl(url: string): string {
  if (!url) return 'http://localhost:8080';
  
  // If URL already has protocol, return as is
  if (url.startsWith('http://') || url.startsWith('https://')) {
    return url;
  }
  
  // If URL starts with //, add https:
  if (url.startsWith('//')) {
    return `https:${url}`;
  }
  
  // Otherwise, assume https for production domains, http for localhost
  if (url.includes('localhost') || url.includes('127.0.0.1')) {
    return `http://${url}`;
  }
  
  // For production domains, use https
  return `https://${url}`;
}

// API configuration
const rawApiBaseUrl = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080';
export const API_BASE_URL = normalizeApiBaseUrl(rawApiBaseUrl);

// Log the API base URL to help debug (always log in production too)
console.log('API Base URL (raw):', rawApiBaseUrl);
console.log('API Base URL (normalized):', API_BASE_URL);

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


