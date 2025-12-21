// Frontend Entry Point
// This is the main entry point for the React application.
// It sets up the React root, router, and error boundary.
//
// Application Structure:
//   - React Router for client-side routing
//   - Error Boundary for graceful error handling
//   - API base URL configured via VITE_API_BASE_URL environment variable
//
// Routes are defined in App.tsx component.

import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { BrowserRouter } from 'react-router-dom';
import App from './App.tsx';
import ErrorBoundary from './components/ErrorBoundary.tsx';
import './index.css';

// Log API URL for debugging
// Note: VITE_API_BASE_URL must be set at build time (not runtime)
console.log('API Base URL:', import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080');

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <ErrorBoundary>
      <BrowserRouter>
        <App />
      </BrowserRouter>
    </ErrorBoundary>
  </StrictMode>,
);


