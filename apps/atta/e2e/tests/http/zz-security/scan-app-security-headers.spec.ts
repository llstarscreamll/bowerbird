import { expect, test } from '@playwright/test';
import { appOrigin, mediaOrigin } from '../../support/origins';
import { expectAPlusSecurityHeaders } from '../../support/security-headers';

test.describe('securityheaders.com — PWA', () => {
  test('GET / (documento) tiene grado A+', async ({ request }) => {
    // given
    const operation = `GET ${appOrigin()}/`;

    // when
    const response = await request.get(appOrigin() + '/');

    // then
    expect(response.status(), `${operation}: document status`).toBeLessThan(400);
    await expectAPlusSecurityHeaders(response, operation);
  });

  test('GET /login tiene grado A+', async ({ request }) => {
    // given
    const operation = `GET ${appOrigin()}/login`;

    // when
    const response = await request.get(appOrigin() + '/login');

    // then
    expect(response.status(), `${operation}: login document status`).toBeLessThan(400);
    await expectAPlusSecurityHeaders(response, operation);
  });

  test('GET ruta SPA desconocida no pierde headers', async ({ request }) => {
    // given
    const operation = `GET ${appOrigin()}/__missing_security_headers_probe`;

    // when
    const response = await request.get(appOrigin() + '/__missing_security_headers_probe');

    // then
    expect(response.status(), `${operation}: spa fallback status`).toBeLessThan(500);
    await expectAPlusSecurityHeaders(response, operation);
  });
});

test.describe('securityheaders.com — media', () => {
  test('GET origen de objetos no es un agujero de headers', async ({ request }) => {
    // given
    const operation = `GET ${mediaOrigin()}/`;

    // when
    const response = await request.get(mediaOrigin() + '/');

    // then
    expect(response.status(), `${operation}: must not 5xx`).toBeLessThan(500);
    await expectAPlusSecurityHeaders(response, operation);
  });
});
