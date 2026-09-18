import fs from 'node:fs';
import path from 'node:path';
import { config as loadEnv } from 'dotenv';
import { defineConfig, devices } from '@playwright/test';
import { resolveE2EOrigins } from './tests/support/origins';

const repoRoot = path.resolve(__dirname, '../../..');
const envFileRaw = process.env.ENV_FILE?.trim();
const envPath = envFileRaw ? (path.isAbsolute(envFileRaw) ? envFileRaw : path.resolve(repoRoot, envFileRaw)) : path.join(repoRoot, 'apps/atta/.env');
if (envFileRaw && !fs.existsSync(envPath)) {
  throw new Error(`ENV_FILE not found: ${envPath}`);
}
if (fs.existsSync(envPath)) {
  loadEnv({ path: envPath, override: true });
}

const origins = resolveE2EOrigins();
console.log(`[e2e] env=${envPath} app=${origins.app} api=${origins.api} media=${origins.media}`);

export default defineConfig({
  testDir: './tests',
  fullyParallel: false,
  timeout: 60_000,
  expect: {
    timeout: 10_000,
  },
  forbidOnly: Boolean(process.env.CI),
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : 5,
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
