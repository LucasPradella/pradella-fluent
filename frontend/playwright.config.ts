import { defineConfig, devices } from '@playwright/test';

// E2E runs against the real API (Go) + the built PWA (vite preview), so the
// service worker is active (quickstart "Run locally" production-like mode).
const DATABASE_URL =
  process.env.DATABASE_URL ?? 'postgres://fluentdev:fluentdev@localhost:5433/fluentdev?sslmode=disable';

export default defineConfig({
  testDir: './tests/e2e',
  timeout: 60_000,
  fullyParallel: false, // shared backend state (rate limits) — keep serial
  workers: 1,
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? 'github' : 'list',
  use: {
    baseURL: 'http://localhost:4173',
    trace: 'retain-on-failure',
    locale: 'pt-BR',
  },
  projects: [
    {
      name: 'chromium',
      use: {
        ...devices['Desktop Chrome'],
        launchOptions: {
          // Fake mic input so speaking exercises can record (US3).
          args: [
            '--use-fake-device-for-media-stream',
            '--use-fake-ui-for-media-stream',
          ],
        },
      },
    },
  ],
  webServer: [
    {
      command: 'go run ./cmd/api -migrate -seed',
      cwd: '../backend',
      port: 8080,
      reuseExistingServer: !process.env.CI,
      timeout: 120_000,
      env: {
        DATABASE_URL,
        ADDR: ':8080',
        APP_BASE_URL: 'http://localhost:4173',
      },
    },
    {
      command: 'npm run preview',
      port: 4173,
      reuseExistingServer: !process.env.CI,
      timeout: 120_000,
    },
  ],
});
