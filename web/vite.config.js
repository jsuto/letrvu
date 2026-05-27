import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [
    vue(),
    tailwindcss(),
  ],
  build: {
    // Build output is embedded into the Go binary via go:embed
    outDir: '../internal/api/static',
    emptyOutDir: true,
    // Disable the modulepreload polyfill — it injects an inline script that
    // violates CSP script-src 'self'. All modern browsers support modulepreload natively.
    modulePreload: { polyfill: false },
  },
  server: {
    port: 5173,
    proxy: {
      '/api': 'http://localhost:8080',
    },
  },
  test: {
    environment: 'jsdom',
    globals: true,
  },
})
