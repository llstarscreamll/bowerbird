import { test as base, type APIRequestContext, expect } from '@playwright/test';
import { AuthApiClient } from './auth-api.client';
import { apiOrigin } from './origins';
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

const buildAuthApiClient = (request: APIRequestContext): AuthApiClient => {
  return new AuthApiClient(request, apiOrigin());
};

const buildPlatformApiClient = (request: APIRequestContext): PlatformApiClient => {
  return new PlatformApiClient(request, apiOrigin());
};

export const test = base.extend<ApiFixtures, WorkerFixtures>({
  sharedTenant: [
    async ({ playwright }, use) => {
      const request = await playwright.request.newContext({ ignoreHTTPSErrors: true });
      try {
        const platformApi = buildPlatformApiClient(request);
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
        const platformApi = buildPlatformApiClient(request);
        await use(await bootstrapAuthenticatedTenantContext(buildLocalUserCredentials(), platformApi));
      } finally {
        await request.dispose();
      }
    },
    { scope: 'worker' },
  ],
  authApi: async ({ request }, use) => {
    await use(buildAuthApiClient(request));
  },
  platformApi: async ({ request }, use) => {
    await use(buildPlatformApiClient(request));
  },
  newUser: async ({}, use) => {
    await use(buildLocalUserCredentials());
  },
});

export { expect };
