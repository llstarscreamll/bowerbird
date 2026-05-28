import { test as base, type APIRequestContext, expect } from '@playwright/test';
import { LobbyPage } from '../pages/lobby.page';
import { LoginPage } from '../pages/login.page';
import { AuthApiClient } from '../support/auth-api.client';
import { PlatformApiClient } from '../support/platform-api.client';
import { buildLocalUserCredentials, type LocalUserCredentials } from '../support/user.factory';

type AppFixtures = {
  authApi: AuthApiClient;
  platformApi: PlatformApiClient;
  loginPage: LoginPage;
  lobbyPage: LobbyPage;
  newUser: LocalUserCredentials;
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

export const test = base.extend<AppFixtures>({
  authApi: async ({ request, baseURL }, use) => {
    await use(buildAuthApiClient(request, baseURL));
  },
  platformApi: async ({ request, baseURL }, use) => {
    await use(buildPlatformApiClient(request, baseURL));
  },
  loginPage: async ({ page }, use) => {
    await use(new LoginPage(page));
  },
  lobbyPage: async ({ page }, use) => {
    await use(new LobbyPage(page));
  },
  newUser: async ({}, use) => {
    await use(buildLocalUserCredentials());
  },
});

export { expect };
