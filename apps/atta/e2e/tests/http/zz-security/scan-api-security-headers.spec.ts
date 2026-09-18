import { expect, test } from '@playwright/test';
import { apiOrigin, appOrigin } from '../../support/origins';
import { PlatformApiClient } from '../../support/platform-api.client';
import { expectAPlusSecurityHeaders, inspectSetCookie } from '../../support/security-headers';
import { buildLocalUserCredentials } from '../../support/user.factory';

test.describe('securityheaders.com — API', () => {
  test('GET /api/health (200) tiene grado A+', async ({ request }) => {
    // given
    const operation = 'GET /api/health';
    const client = new PlatformApiClient(request, apiOrigin());

    // when
    const response = await client.call('/api/health');

    // then
    await expectAPlusSecurityHeaders(response, operation);
  });

  test('GET /api/v1/identity/me sin token (401) no pierde headers', async ({ request }) => {
    // given
    const operation = 'GET /api/v1/identity/me 401';
    const client = new PlatformApiClient(request, apiOrigin());

    // when
    const response = await client.call('/api/v1/identity/me');

    // then
    expect(response.status(), `${operation}: expected 401`).toBe(401);
    await expectAPlusSecurityHeaders(response, operation);
  });

  test('GET ruta inexistente (404) no pierde headers', async ({ request }) => {
    // given
    const operation = 'GET /api/v1/__missing_security_headers_probe';
    const client = new PlatformApiClient(request, apiOrigin());

    // when
    const response = await client.call('/api/v1/__missing_security_headers_probe');

    // then
    expect(response.status(), `${operation}: expected 404`).toBe(404);
    await expectAPlusSecurityHeaders(response, operation);
  });

  test('OPTIONS CORS preflight no pierde headers', async ({ request }) => {
    // given
    const operation = 'OPTIONS /api/v1/identity/me';
    const origin = apiOrigin();

    // when
    const response = await request.fetch(`${origin}/api/v1/identity/me`, {
      method: 'OPTIONS',
      headers: {
        Origin: appOrigin(),
        'Access-Control-Request-Method': 'GET',
        'Access-Control-Request-Headers': 'Authorization,X-Tenant-ID',
      },
    });

    // then
    expect(response.status(), `${operation}: preflight status`).toBeLessThan(500);
    await expectAPlusSecurityHeaders(response, operation);
  });

  test('Set-Cookie de login-local es Secure+HttpOnly+SameSite', async ({ request }) => {
    // given
    const operation = 'POST /api/v1/auth/login-local Set-Cookie';
    const client = new PlatformApiClient(request, apiOrigin());
    const user = buildLocalUserCredentials();
    await client.registerLocalOrFail(user);

    // when
    const response = await client.loginLocal(user);

    // then
    expect(response.ok(), `${operation}: login must succeed to inspect cookie`).toBe(true);
    const setCookie = response.headers()['set-cookie'] ?? '';
    expect(setCookie, `${operation}: expected a refresh cookie`).toBeTruthy();
    inspectSetCookie(setCookie, operation);
    await expectAPlusSecurityHeaders(response, operation);
  });
});
