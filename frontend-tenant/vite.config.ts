import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'

const devServerHost = process.env.VITE_DEV_SERVER_HOST || '0.0.0.0'
const devServerPort = Number(process.env.VITE_DEV_SERVER_PORT || 5174)
const hmrHost = process.env.VITE_HMR_HOST
const hmrPort = Number(process.env.VITE_HMR_PORT || devServerPort)

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    host: devServerHost,
    port: devServerPort,
    strictPort: true,
    watch: {
      usePolling: process.env.CHOKIDAR_USEPOLLING === 'true',
    },
    hmr: hmrHost
      ? {
          host: hmrHost,
          port: hmrPort,
          clientPort: hmrPort,
        }
      : undefined,
  },
})
