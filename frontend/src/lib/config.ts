// Helper function to ensure API base URL has a protocol
function normalizeApiBaseUrl(url: string): string {
  if (!url) return '';
  
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
// In development mode (dev server), use relative URLs to leverage Vite proxy
// In production, use the full URL from environment variable
const rawApiBaseUrl = import.meta.env.VITE_API_BASE_URL;

// For local development, always use localhost:8080
// Check if we're running on localhost (frontend dev server)
// This check works for both dev server and production builds served locally
const isLocalDev = typeof window !== 'undefined' && 
  (window.location.hostname === 'localhost' || 
   window.location.hostname === '127.0.0.1' ||
   window.location.hostname === ''); // Handle file:// protocol or edge cases

// In local development, always use localhost:8080 (ignore any staging URLs)
// This ensures that even production builds served locally will use the local backend
// In production (non-localhost), use the environment variable or default to localhost
let effectiveApiUrl: string;
if (isLocalDev) {
  // Always use localhost when running locally, regardless of environment variable
  effectiveApiUrl = 'http://localhost:8080';
} else if (rawApiBaseUrl) {
  // Use environment variable when not on localhost
  effectiveApiUrl = rawApiBaseUrl;
} else {
  // Default fallback
  effectiveApiUrl = 'http://localhost:8080';
}

export const API_BASE_URL = normalizeApiBaseUrl(effectiveApiUrl);

// Log the API base URL to help debug (always log in production too)
console.log('API Base URL (raw env):', rawApiBaseUrl);
console.log('API Base URL (effective):', effectiveApiUrl);
console.log('API Base URL (normalized):', API_BASE_URL);
console.log('Is local dev:', isLocalDev);

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


