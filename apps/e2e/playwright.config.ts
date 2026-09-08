import path from 'node:path';
import { config as loadEnv } from 'dotenv';
import { defineConfig, devices } from '@playwright/test';
import { resolveE2EOrigins } from './tests/support/origins';

loadEnv({ path: path.resolve(__dirname, '../../.env') });

const origins = resolveE2EOrigins();
console.log(`[e2e] app=${origins.app} api=${origins.api} media=${origins.media}`);

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
    baseURL: origins.app,
    ignoreHTTPSErrors: true,
    trace: { mode: 'retain-on-failure-and-retries', snapshots: true, sources: true, attachments: true },
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
  },
  projects: [
    {
      name: 'desktop-chromium',
      testMatch: 'browser/**/*.spec.ts',
      use: { ...devices['Desktop Chrome'] },
    },
    {
      name: 'webkit',
      testMatch: 'browser/**/*.spec.ts',
      use: { ...devices['Desktop Safari'] },
    },
    {
      name: 'http',
      testMatch: 'http/**/*.spec.ts',
      workers: 1,
    },
  ],
});
