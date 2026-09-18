import { expect, test } from '@playwright/test';
import { OBSERVATORY_CORS_ORIGIN, expectObservatoryAPlus, probeCors, probeHttpRedirect, scanMozillaObservatory, setCookiesFromResponse } from '../../support/mozilla-observatory';
import { appOrigin, mediaOrigin } from '../../support/origins';

const scanDocument = async (request: Parameters<typeof probeCors>[0], url: string, operation: string) => {
  const response = await request.get(url);
  const cors = await probeCors(request, url, OBSERVATORY_CORS_ORIGIN);
  const httpRedirect = await probeHttpRedirect(request, url);
  const contentType = response.headers()['content-type'] ?? '';
  const html = contentType.includes('html') ? await response.text() : undefined;
  const scan = scanMozillaObservatory({
    url: response.url(),
    status: response.status(),
    headers: response.headers(),
    html,
    setCookies: setCookiesFromResponse(response),
    cors,
    httpRedirect,
  });
  expect(response.status(), `${operation}: must not 5xx`).toBeLessThan(500);
  return scan;
};

test.describe('Mozilla HTTP Observatory — PWA', () => {
  test('GET / (documento) tiene grado A+', async ({ request }) => {
    // given
    const operation = `observatory GET ${appOrigin()}/`;

    // when
    const scan = await scanDocument(request, `${appOrigin()}/`, operation);

    // then
    await expectObservatoryAPlus(scan, operation);
  });

  test('GET /login tiene grado A+', async ({ request }) => {
    // given
    const operation = `observatory GET ${appOrigin()}/login`;

    // when
    const scan = await scanDocument(request, `${appOrigin()}/login`, operation);

    // then
    await expectObservatoryAPlus(scan, operation);
  });

  test('GET ruta SPA desconocida no pierde la nota', async ({ request }) => {
    // given
    const operation = `observatory GET ${appOrigin()}/__missing_observatory_probe`;

    // when
    const scan = await scanDocument(request, `${appOrigin()}/__missing_observatory_probe`, operation);

    // then
    await expectObservatoryAPlus(scan, operation);
  });

  test('HTTP redirige a HTTPS en el mismo host', async ({ request }) => {
    // given
    const operation = 'observatory HTTP→HTTPS';
    const httpsUrl = `${appOrigin()}/`;

    // when
    const probe = await probeHttpRedirect(request, httpsUrl);

    // then
    if (!probe.reached) {
      return;
    }
    expect(probe.status, `${operation}: HTTP ${probe.status} Location=${probe.location}`).toBeGreaterThanOrEqual(300);
    expect(probe.status, `${operation}: redirect`).toBeLessThan(400);
    const target = new URL(probe.location ?? '', httpsUrl);
    expect(target.protocol, `${operation}: Location must be https`).toBe('https:');
    expect(target.hostname, `${operation}: first hop must stay on the same host`).toBe(new URL(httpsUrl).hostname);
  });

  test('scripts del documento no cargan CDN sin SRI ni http://', async ({ request }) => {
    // given
    const operation = 'observatory SRI scripts';

    // when
    const response = await request.get(`${appOrigin()}/`);
    const html = await response.text();
    const scan = scanMozillaObservatory({
      url: response.url(),
      status: response.status(),
      headers: response.headers(),
      html,
      setCookies: setCookiesFromResponse(response),
    });
    const sri = scan.findings.find((item) => item.test === 'subresource-integrity');

    // then
    expect(sri?.pass, `${operation}: ${sri?.result} ${sri?.detail}\n${scan.report}`).toBe(true);
    expect(sri?.modifier, `${operation}: SRI must not penalize\n${scan.report}`).toBeGreaterThanOrEqual(0);
  });

  test('crossdomain.xml / clientaccesspolicy.xml no dan acceso universal', async ({ request }) => {
    // given
    const operation = 'observatory flash cross-domain';

    // when
    const crossdomain = await request.get(`${appOrigin()}/crossdomain.xml`);
    const clientAccess = await request.get(`${appOrigin()}/clientaccesspolicy.xml`);
    const bodies = `${await crossdomain.text()}\n${await clientAccess.text()}`.toLowerCase();

    // then
    expect.soft(bodies.includes('domain="*"'), `${operation}: crossdomain.xml allow-access-from domain=* is Observatory -50`).toBe(false);
    expect.soft(bodies.includes('allow-from-http-handlers'), `${operation}: Silverlight * policy`).toBe(false);
  });
});

test.describe('Mozilla HTTP Observatory — media', () => {
  test('GET origen de objetos no es un agujero Observatory', async ({ request }) => {
    // given
    const operation = `observatory GET ${mediaOrigin()}/`;

    // when
    const scan = await scanDocument(request, `${mediaOrigin()}/`, operation);

    // then
    await expectObservatoryAPlus(scan, operation);
  });
});
