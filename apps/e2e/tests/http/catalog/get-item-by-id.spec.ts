import { expect } from '@playwright/test';
import { test } from '../../support/api.fixture';
import { expectStatus, readJson } from '../../support/http-assertions';
import { newUlid } from '../../support/ulid';

const OPERATION = 'GET /api/v1/catalog/items/{id}';

test.describe(OPERATION, () => {
  test('devuelve el ítem creado', async ({ sharedTenant, platformApi }) => {
    // given
    const { auth, tenant } = sharedTenant;
    const id = newUlid();
    const name = `Get Item ${Date.now()}`;
    await expectStatus(
      await platformApi.call('/api/v1/catalog/items', {
        method: 'POST',
        auth,
        tenant,
        data: {
          data: {
            type: 'catalog_items',
            id,
            attributes: { name, kind: 'asset', internal_sku: `SKU-GET-${Date.now()}` },
          },
        },
      }),
      201,
      'POST /api/v1/catalog/items',
    );

    // when
    const response = await platformApi.call(`/api/v1/catalog/items/${id}`, { auth, tenant });

    // then
    await expectStatus(response, 200, OPERATION);
    const payload = await readJson<{ data: { id: string; attributes: { name: string } } }>(response, OPERATION);
    expect(payload.data.id, `${OPERATION}: id`).toBe(id);
    expect(payload.data.attributes.name, `${OPERATION}: name`).toBe(name);
  });

  test('404 si el ítem no existe', async ({ sharedTenant, platformApi }) => {
    // given
    const { auth, tenant } = sharedTenant;

    // when
    const response = await platformApi.call(`/api/v1/catalog/items/${newUlid()}`, { auth, tenant });

    // then
    await expectStatus(response, 404, OPERATION);
    const payload = await readJson<{ errors: Array<{ code?: string; detail?: string }> }>(response, OPERATION);
    expect(payload.errors[0].code, `${OPERATION}: errors[0].code`).toBe('ERR_NOT_FOUND');
    expect(payload.errors[0].detail, `${OPERATION}: errors[0].detail`).toContain('catalog item not found');
  });

  test('404 si el ítem es de otro tenant', async ({ sharedTenant, foreignTenant, platformApi }) => {
    // given
    const id = newUlid();
    await expectStatus(
      await platformApi.call('/api/v1/catalog/items', {
        method: 'POST',
        auth: sharedTenant.auth,
        tenant: sharedTenant.tenant,
        data: {
          data: {
            type: 'catalog_items',
            id,
            attributes: { name: 'Secret item', kind: 'goods', internal_sku: `SKU-ISO-${Date.now()}` },
          },
        },
      }),
      201,
      'POST /api/v1/catalog/items',
    );

    // when
    const response = await platformApi.call(`/api/v1/catalog/items/${id}`, {
      auth: foreignTenant.auth,
      tenant: foreignTenant.tenant,
    });

    // then
    await expectStatus(response, 404, OPERATION);
    const payload = await readJson<{ errors: Array<{ code?: string; detail?: string }> }>(response, OPERATION);
    expect(payload.errors[0].code, `${OPERATION}: errors[0].code`).toBe('ERR_NOT_FOUND');
    expect(JSON.stringify(payload), `${OPERATION}: must not leak item name`).not.toContain('Secret item');
  });
});
