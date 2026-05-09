import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      // Forward all BFF routes to the local BFF in dev mode.
      // The BFF serves at root with no /api/v1 prefix.
      '/health':        { target: 'http://localhost:8080', changeOrigin: true },
      '/articles':      { target: 'http://localhost:8080', changeOrigin: true },
      '/tickets':       { target: 'http://localhost:8080', changeOrigin: true },
      '/crew':          { target: 'http://localhost:8080', changeOrigin: true },
      '/conversations': { target: 'http://localhost:8080', changeOrigin: true },
      '/chat':          { target: 'http://localhost:8080', changeOrigin: true },
      '/curate':        { target: 'http://localhost:8080', changeOrigin: true },
      '/reset':         { target: 'http://localhost:8080', changeOrigin: true },
    },
  },
})
