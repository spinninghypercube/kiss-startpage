import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: './tests/e2e',
  fullyParallel: false,
  workers: 1,
  timeout: 60_000,
  expect: {
    timeout: 8_000
  },
  use: {
    baseURL: 'http://127.0.0.1:18788',
    browserName: 'chromium',
    trace: 'retain-on-failure'
  },
  webServer: {
    command: 'npm run build && node tests/e2e/server.mjs',
    url: 'http://127.0.0.1:18788/health',
    reuseExistingServer: false,
    timeout: 120_000
  }
});
