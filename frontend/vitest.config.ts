import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'

// Include the Vue plugin so Vitest can handle .vue SFC imports during tests
export default defineConfig({
  plugins: [vue()],
  test: {
    environment: 'jsdom',
    exclude: [
      'tests-e2e/**',
      'node_modules/**',
      'dist/**'
    ]
  }
})
