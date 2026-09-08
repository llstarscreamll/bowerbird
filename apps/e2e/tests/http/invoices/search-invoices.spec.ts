import { expect } from '@playwright/test';
import { test } from '../../support/api.fixture';
import { expectClientError, expectStatus, readJson } from '../../support/http-assertions';

const OPERATION = 'GET /api/v1/invoicing/invoices';

test.describe(OPERATION, () => {
  test('colección vacía en tenant nuevo', async ({ sharedTenant, platformApi }) => {
    // given
    const { auth, tenant } = sharedTenant;

    // when
    const response = await platformApi.call('/api/v1/invoicing/invoices', { auth, tenant });

    // then
    await expectStatus(response, 200, OPERATION);
    const payload = await readJson<{ data: unknown[]; meta: { has_more: boolean } }>(response, OPERATION);
    expect(payload.data, `${OPERATION}: data should be an empty collection`).toEqual([]);
    expect(payload.meta.has_more, `${OPERATION}: meta.has_more`).toBe(false);
  });

  test('400 si el cursor es basura', async ({ sharedTenant, platformApi }) => {
    // given
    const { auth, tenant } = sharedTenant;

    // when
    const response = await platformApi.call('/api/v1/invoicing/invoices?cursor=not-a-ulid', { auth, tenant });

    // then
    await expectStatus(response, 400, OPERATION);
  });

  test('400 si el limit es negativo', async ({ sharedTenant, platformApi }) => {
    // given
    const { auth, tenant } = sharedTenant;

    // when
    const response = await platformApi.call('/api/v1/invoicing/invoices?limit=-1', { auth, tenant });

    // then
    await expectClientError(response, OPERATION);
  });

  test('401 sin token', async ({ sharedTenant, platformApi }) => {
    // given
    const { tenant } = sharedTenant;

    // when
    const response = await platformApi.call('/api/v1/invoicing/invoices', { tenant });

    // then
    await expectStatus(response, 401, OPERATION);
  });
});
