import { expect } from '@playwright/test';
import { test } from '../../support/api.fixture';
import { expectStatus, readJson } from '../../support/http-assertions';
import { newUlid } from '../../support/ulid';

const OPERATION = 'GET /api/v1/catalog/items';

test.describe(OPERATION, () => {
  test('encuentra el ítem por SKU', async ({ sharedTenant, platformApi }) => {
    // given
    const { auth, tenant } = sharedTenant;
    const id = newUlid();
    const stamp = `${Date.now()}`;
    const sku = `SKU-SEARCH-${stamp}`;
    await expectStatus(
      await platformApi.call('/api/v1/catalog/items', {
        method: 'POST',
        auth,
        tenant,
        data: {
          data: {
            type: 'catalog_items',
            id,
            attributes: { name: `Search Item ${stamp}`, kind: 'service', internal_sku: sku },
          },
        },
      }),
      201,
      'POST /api/v1/catalog/items',
    );

    // when
    const response = await platformApi.call(`/api/v1/catalog/items?search=${encodeURIComponent(sku)}`, { auth, tenant });

    // then
    await expectStatus(response, 200, OPERATION);
    const payload = await readJson<{ data: Array<{ id: string; attributes: { internal_sku: string | null } }> }>(response, OPERATION);
    expect(
      payload.data.map((item) => item.id),
      `${OPERATION}: ids`,
    ).toContain(id);
    expect(payload.data.find((item) => item.id === id)?.attributes.internal_sku, `${OPERATION}: internal_sku`).toBe(sku);
  });

  test('no 500 ante inyección en search', async ({ sharedTenant, platformApi }) => {
    // given
    const { auth, tenant } = sharedTenant;
    const needle = `' OR 1=1; DROP TABLE catalog_items; --`;

    // when
    const response = await platformApi.call(`/api/v1/catalog/items?search=${encodeURIComponent(needle)}`, {
      auth,
      tenant,
    });

    // then
    expect(response.status(), `${OPERATION} must not 5xx on injection`).toBeLessThan(500);
    if (response.status() === 200) {
      const payload = await readJson<{ data: unknown[] }>(response, OPERATION);
      expect(Array.isArray(payload.data), `${OPERATION}: data must stay an array`).toBe(true);
    }
  });

  test('404 si review-queue ya no existe', async ({ sharedTenant, platformApi }) => {
    // given
    const { auth, tenant } = sharedTenant;

    // when
    const response = await platformApi.call('/api/v1/catalog/review-queue', { auth, tenant });

    // then
    await expectStatus(response, 404, 'GET /api/v1/catalog/review-queue');
  });

  test('401 sin token', async ({ sharedTenant, platformApi }) => {
    // given
    const { tenant } = sharedTenant;

    // when
    const response = await platformApi.call('/api/v1/catalog/items', { tenant });

    // then
    await expectStatus(response, 401, OPERATION);
  });
});
