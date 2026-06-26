import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// In dev, proxy the API (REST + the /api/console WebSocket) to the Go server
// so the app uses same-origin requests and skips CORS. Override the target
// with VITE_API_PROXY when the API runs elsewhere.
const target = process.env.VITE_API_PROXY ?? 'http://localhost:8080'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/api': { target, changeOrigin: true, ws: true },
    },
  },
})
