import { expect } from '@playwright/test';
import { test } from '../../support/api.fixture';
import { expectClientError, expectStatus, readJson } from '../../support/http-assertions';

const OPERATION = 'GET /api/v1/entitlements';

test.describe(OPERATION, () => {
  test('devuelve el pack por defecto sin mail.send', async ({ sharedTenant, platformApi }) => {
    // given
    const { auth, tenant } = sharedTenant;

    // when
    const response = await platformApi.call('/api/v1/entitlements', { auth, tenant });

    // then
    await expectStatus(response, 200, OPERATION);
    const payload = await readJson<{ features: string[] }>(response, OPERATION);
    expect(payload.features, `${OPERATION}: default pack`).toEqual(expect.arrayContaining(['mail.inbox', 'mail.organize', 'invoicing.workspace', 'invoicing.capture_from_email']));
    expect(payload.features, `${OPERATION}: mail.send excluded`).not.toContain('mail.send');
  });

  test('401 sin token', async ({ sharedTenant, platformApi }) => {
    // given
    const { tenant } = sharedTenant;

    // when
    const response = await platformApi.call('/api/v1/entitlements', { tenant });

    // then
    await expectStatus(response, 401, OPERATION);
  });

  test('rechaza request sin tenant', async ({ sharedTenant, platformApi }) => {
    // given
    const { auth } = sharedTenant;

    // when
    const response = await platformApi.call('/api/v1/entitlements', { auth });

    // then
    await expectClientError(response, OPERATION);
  });
});
