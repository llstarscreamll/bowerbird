import { expect } from '@playwright/test';
import { test } from '../fixtures';
import { expectAPlusSecurityHeaders, normalizeHeaders } from '../../support/security-headers';

test.describe('securityheaders.com — documento en el navegador', () => {
  test('navegación a /login aplica headers A+', async ({ page }) => {
    // given
    const operation = 'browser GET /login';

    // when
    const response = await page.goto('/login', { waitUntil: 'domcontentloaded' });

    // then
    expect(response, `${operation}: expected a document response`).toBeTruthy();
    const documentResponse = response!;
    expect(documentResponse.status(), `${operation}: status`).toBeLessThan(400);
    await expectAPlusSecurityHeaders(
      {
        url: () => documentResponse.url(),
        status: () => documentResponse.status(),
        headers: () => normalizeHeaders(documentResponse.headers()),
      },
      operation,
    );
  });

  test('navegación a / (SPA) aplica headers A+', async ({ page }) => {
    // given
    const operation = 'browser GET /';

    // when
    const response = await page.goto('/', { waitUntil: 'domcontentloaded' });

    // then
    expect(response, `${operation}: expected a document response`).toBeTruthy();
    const documentResponse = response!;
    expect(documentResponse.status(), `${operation}: status`).toBeLessThan(400);
    await expectAPlusSecurityHeaders(
      {
        url: () => documentResponse.url(),
        status: () => documentResponse.status(),
        headers: () => normalizeHeaders(documentResponse.headers()),
      },
      operation,
    );
  });
});
