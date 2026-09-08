import { expect, test } from '@playwright/test';
import { expectObservatoryAPlus, scanMozillaObservatory } from '../../support/mozilla-observatory';

test.describe('Mozilla HTTP Observatory — documento en el navegador', () => {
  test('navegación a /login aplica A+', async ({ page }) => {
    // given
    const operation = 'observatory browser GET /login';

    // when
    const response = await page.goto('/login', { waitUntil: 'domcontentloaded' });

    // then
    expect(response, `${operation}: expected a document response`).toBeTruthy();
    const documentResponse = response!;
    const html = await page.content();
    const scan = scanMozillaObservatory({
      url: documentResponse.url(),
      status: documentResponse.status(),
      headers: documentResponse.headers(),
      html,
    });
    await expectObservatoryAPlus(scan, operation);
  });

  test('navegación a / (SPA) aplica A+ y no carga scripts foráneos sin SRI', async ({ page }) => {
    // given
    const operation = 'observatory browser GET /';

    // when
    const response = await page.goto('/', { waitUntil: 'domcontentloaded' });

    // then
    expect(response, `${operation}: expected a document response`).toBeTruthy();
    const documentResponse = response!;
    const html = await page.content();
    const scan = scanMozillaObservatory({
      url: documentResponse.url(),
      status: documentResponse.status(),
      headers: documentResponse.headers(),
      html,
    });
    const sri = scan.findings.find((item) => item.test === 'subresource-integrity');
    expect.soft(sri?.pass, `${operation}: SRI ${sri?.result} ${sri?.detail}`).toBe(true);
    await expectObservatoryAPlus(scan, operation);
  });
});
