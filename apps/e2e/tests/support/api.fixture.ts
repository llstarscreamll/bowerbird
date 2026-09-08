import { test as base, type APIRequestContext, expect } from '@playwright/test';
import { AuthApiClient } from './auth-api.client';
import { PlatformApiClient } from './platform-api.client';
import { bootstrapAuthenticatedTenantContext, type AuthenticatedTenantContext } from './test-context.factory';
import { buildLocalUserCredentials, type LocalUserCredentials } from './user.factory';

type ApiFixtures = {
  authApi: AuthApiClient;
  platformApi: PlatformApiClient;
  newUser: LocalUserCredentials;
};

type WorkerFixtures = {
  sharedTenant: AuthenticatedTenantContext;
  foreignTenant: AuthenticatedTenantContext;
};

const apiBaseUrlFrom = (baseURL?: string): string => {
  const defaultApiUrl = 'https://api.bowerbird.dev';

  if (process.env.E2E_API_BASE_URL) {
    return process.env.E2E_API_BASE_URL;
  }

  if (!baseURL) {
    return defaultApiUrl;
  }

  try {
    const url = new URL(baseURL);
    if (url.hostname.startsWith('app.')) {
      url.hostname = url.hostname.replace(/^app\./, 'api.');
      return url.origin;
    }
  } catch {
    return defaultApiUrl;
  }

  return defaultApiUrl;
};

const buildAuthApiClient = (request: APIRequestContext, baseURL?: string): AuthApiClient => {
  const apiBaseUrl = apiBaseUrlFrom(baseURL);
  return new AuthApiClient(request, apiBaseUrl);
};

const buildPlatformApiClient = (request: APIRequestContext, baseURL?: string): PlatformApiClient => {
  const apiBaseUrl = apiBaseUrlFrom(baseURL);
  return new PlatformApiClient(request, apiBaseUrl);
};

export const test = base.extend<ApiFixtures, WorkerFixtures>({
  sharedTenant: [
    async ({ playwright }, use) => {
      const request = await playwright.request.newContext({ ignoreHTTPSErrors: true });
      try {
        const platformApi = buildPlatformApiClient(request, process.env.E2E_BASE_URL ?? 'https://app.bowerbird.dev');
        await use(await bootstrapAuthenticatedTenantContext(buildLocalUserCredentials(), platformApi));
      } finally {
        await request.dispose();
      }
    },
    { scope: 'worker' },
  ],
  foreignTenant: [
    async ({ playwright }, use) => {
      const request = await playwright.request.newContext({ ignoreHTTPSErrors: true });
      try {
        const platformApi = buildPlatformApiClient(request, process.env.E2E_BASE_URL ?? 'https://app.bowerbird.dev');
        await use(await bootstrapAuthenticatedTenantContext(buildLocalUserCredentials(), platformApi));
      } finally {
        await request.dispose();
      }
    },
    { scope: 'worker' },
  ],
  authApi: async ({ request, baseURL }, use) => {
    await use(buildAuthApiClient(request, baseURL));
  },
  platformApi: async ({ request, baseURL }, use) => {
    await use(buildPlatformApiClient(request, baseURL));
  },
  newUser: async ({}, use) => {
    await use(buildLocalUserCredentials());
  },
});

export { expect };
