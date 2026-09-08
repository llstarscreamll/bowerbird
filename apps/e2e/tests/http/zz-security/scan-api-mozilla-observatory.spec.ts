import { expect, test } from '@playwright/test';
import { OBSERVATORY_CORS_ORIGIN, expectObservatoryAPlus, probeCors, probeHttpRedirect, scanMozillaObservatory, setCookiesFromResponse } from '../../support/mozilla-observatory';
import { PlatformApiClient } from '../../support/platform-api.client';
import { apiOrigin } from '../../support/origins';
import { buildLocalUserCredentials } from '../../support/user.factory';

const scanApi = async (
  response: Awaited<ReturnType<PlatformApiClient['call']>>,
  extras: { cors?: Awaited<ReturnType<typeof probeCors>>; httpRedirect?: Awaited<ReturnType<typeof probeHttpRedirect>> },
) => {
  const html = (response.headers()['content-type'] ?? '').includes('html') ? await response.text() : undefined;
  return scanMozillaObservatory({
    url: response.url(),
    status: response.status(),
    headers: response.headers(),
    html,
    setCookies: setCookiesFromResponse(response),
    cors: extras.cors,
    httpRedirect: extras.httpRedirect,
  });
};

test.describe('Mozilla HTTP Observatory — API', () => {
  test('GET /health tiene grado A+', async ({ request }) => {
    // given
    const operation = 'observatory GET /health';
    const origin = apiOrigin();
    const client = new PlatformApiClient(request, origin);

    // when
    const response = await client.call('/health');
    const cors = await probeCors(request, `${origin}/health`, OBSERVATORY_CORS_ORIGIN);
    const httpRedirect = await probeHttpRedirect(request, `${origin}/health`);
    const scan = await scanApi(response, { cors, httpRedirect });

    // then
    await expectObservatoryAPlus(scan, operation);
  });

  test('GET 401 no pierde la nota Observatory', async ({ request }) => {
    // given
    const operation = 'observatory GET /api/v1/identity/me 401';
    const origin = apiOrigin();
    const client = new PlatformApiClient(request, origin);

    // when
    const response = await client.call('/api/v1/identity/me');
    const cors = await probeCors(request, `${origin}/api/v1/identity/me`, OBSERVATORY_CORS_ORIGIN);
    const scan = await scanApi(response, { cors });

    // then
    expect(response.status(), `${operation}: expected 401`).toBe(401);
    await expectObservatoryAPlus(scan, operation);
  });

  test('GET 404 no pierde la nota Observatory', async ({ request }) => {
    // given
    const operation = 'observatory GET missing 404';
    const origin = apiOrigin();
    const client = new PlatformApiClient(request, origin);

    // when
    const response = await client.call('/api/v1/__missing_observatory_probe');
    const cors = await probeCors(request, `${origin}/api/v1/__missing_observatory_probe`, OBSERVATORY_CORS_ORIGIN);
    const scan = await scanApi(response, { cors });

    // then
    expect(response.status(), `${operation}: expected 404`).toBe(404);
    await expectObservatoryAPlus(scan, operation);
  });

  test('CORS no refleja un Origin atacante con credentials', async ({ request }) => {
    // given
    const operation = 'observatory CORS attacker origin';
    const origin = apiOrigin();
    const attacker = 'https://evil.example';

    // when
    const cors = await probeCors(request, `${origin}/api/v1/identity/me`, attacker);

    // then
    expect.soft(cors.acao, `${operation}: must not reflect attacker Origin`).not.toBe(attacker);
    expect.soft(cors.acao, `${operation}: must not be *`).not.toBe('*');
    if (cors.credentials.toLowerCase() === 'true') {
      expect(cors.acao, `${operation}: credentials + reflected Origin is universal CORS (-50)`).not.toBe(attacker);
    }
  });

  test('Set-Cookie de login pasa cookies-secure-with-httponly-sessions-and-samesite', async ({ request }) => {
    // given
    const operation = 'observatory login cookies';
    const origin = apiOrigin();
    const client = new PlatformApiClient(request, origin);
    const user = buildLocalUserCredentials();
    await client.registerLocalOrFail(user);

    // when
    const response = await client.loginLocal(user);
    const cors = await probeCors(request, `${origin}/api/v1/auth/login-local`, OBSERVATORY_CORS_ORIGIN);
    const scan = await scanApi(response, { cors });

    // then
    expect(response.ok(), `${operation}: login must succeed`).toBe(true);
    const cookies = scan.findings.find((item) => item.test === 'cookies');
    expect(cookies?.result, `${operation}: ${scan.report}`).toBe('cookies-secure-with-httponly-sessions-and-samesite');
    await expectObservatoryAPlus(scan, operation);
  });
});
