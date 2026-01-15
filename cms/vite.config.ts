import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  base: '/cms/', // Base path for the CMS
  server: {
    port: 5174, // Different port from main frontend (3000) to avoid conflicts
    host: true,
  },
  build: {
    outDir: 'dist',
    sourcemap: false,
  },
})

