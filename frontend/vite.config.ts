// Vite Configuration for Stackyn Frontend
// This file configures the Vite build tool for the React/TypeScript frontend.
//
// Key Configuration:
//   - React plugin for JSX/TSX support
//   - Path aliases (@ -> ./src) for cleaner imports
//   - Development server on port 3000
//   - Build output to 'dist' directory
//
// Environment Variables:
//   - VITE_API_BASE_URL: API endpoint URL (set at build time)
//   - Access via import.meta.env.VITE_API_BASE_URL
//
// Migration Notes:
//   - Migrated from Create React App to Vite for faster builds
//   - Build process is significantly faster with Vite
//   - Hot Module Replacement (HMR) for better development experience

import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import path from 'path';

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    port: 3000,
    host: true,
  },
  build: {
    outDir: 'dist',
    sourcemap: false,
  },
});


