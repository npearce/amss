import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      // Forward all /api/v1/* requests to the local BFF in dev mode.
      // VITE_API_URL is empty in dev, so the browser calls /api/v1/...
      // which Vite proxies here. The BFF handles /api/v1/* natively.
      '/api/v1': { target: 'http://localhost:8080', changeOrigin: true },
    },
  },
})
