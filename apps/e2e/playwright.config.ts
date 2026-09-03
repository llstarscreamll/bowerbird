import path from 'node:path';
import { config as loadEnv } from 'dotenv';
import { defineConfig, devices } from '@playwright/test';

loadEnv({ path: path.resolve(__dirname, '../../.env') });

const appBaseUrl = process.env.E2E_BASE_URL ?? 'https://app.bowerbird.dev';

export default defineConfig({
  testDir: './tests',
  fullyParallel: false,
  timeout: 60_000,
  expect: {
    timeout: 10_000,
  },
  forbidOnly: Boolean(process.env.CI),
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: [['list'], ['html', { outputFolder: 'playwright-report', open: 'never' }]],
  use: {
    baseURL: appBaseUrl,
    ignoreHTTPSErrors: true,
    trace: { mode: 'retain-on-failure-and-retries', snapshots: true, sources: true, attachments: true },
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
  },
  projects: [
    {
      name: 'desktop-chromium',
      use: { ...devices['Desktop Chrome'] },
    },
    {
      name: 'webkit',
      use: { ...devices['Desktop Safari'] },
    },
  ],
});
