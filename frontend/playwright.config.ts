import { defineConfig } from '@playwright/test'

export default defineConfig({
  testDir: './tests-e2e',
  timeout: 45_000,
  workers: 1,
  use: {
    headless: true,
    baseURL: 'http://localhost:5173'
  },
  projects: [
    { name: 'chromium', use: { browserName: 'chromium' } }
  ],
  webServer: {
    command: 'npm run preview',
    port: 5173,
    reuseExistingServer: true
  }
})
