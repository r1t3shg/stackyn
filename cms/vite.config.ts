import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  base: '/', // Base path for the CMS (root of subdomain). Ensures assets are served from /assets/
  server: {
    port: 3001, // Match the port in docker-compose.yml
    host: true,
  },
  build: {
    outDir: 'dist',
    sourcemap: false,
    // Ensure assets are referenced correctly
    assetsDir: 'assets',
  },
})

