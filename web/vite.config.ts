import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// Весь не-vite трафик проксируем на Go-бэкенд, чтобы не возиться с CORS.
const backend = 'http://127.0.0.1:8080'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/auth': backend,
      '/kitchens': backend,
      '/clients': backend,
      '/products': backend,
      '/orders': backend,
    },
  },
})
