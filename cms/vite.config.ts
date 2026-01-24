import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  base: '/cms/', // Base path for the CMS (with trailing slash for Vite asset resolution)
  server: {
    port: 3001, // Match the port in docker-compose.yml
    host: true,
  },
  build: {
    outDir: 'dist',
    sourcemap: false,
  },
})

