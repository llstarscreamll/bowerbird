import { expect } from '@playwright/test';
import { test } from '../../support/api.fixture';
import { expectStatus, readJson } from '../../support/http-assertions';
import { newUlid } from '../../support/ulid';

const OPERATION = 'PATCH /api/v1/catalog/items/{id}';

test.describe(OPERATION, () => {
  test('actualiza el nombre', async ({ sharedTenant, platformApi }) => {
    // given
    const { auth, tenant } = sharedTenant;
    const id = newUlid();
    await expectStatus(
      await platformApi.call('/api/v1/catalog/items', {
        method: 'POST',
        auth,
        tenant,
        data: {
          data: {
            type: 'catalog_items',
            id,
            attributes: { name: `Before ${Date.now()}`, kind: 'goods', internal_sku: `SKU-PATCH-${Date.now()}` },
          },
        },
      }),
      201,
      'POST /api/v1/catalog/items',
    );
    const name = `After ${Date.now()}`;

    // when
    const response = await platformApi.call(`/api/v1/catalog/items/${id}`, {
      method: 'PATCH',
      auth,
      tenant,
      data: { data: { attributes: { name } } },
    });

    // then
    await expectStatus(response, 200, OPERATION);
    const payload = await readJson<{ data: { attributes: { name: string } } }>(response, OPERATION);
    expect(payload.data.attributes.name, `${OPERATION}: name`).toBe(name);
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
            attributes: { name: 'Do not patch', kind: 'goods', internal_sku: `SKU-X-${Date.now()}` },
          },
        },
      }),
      201,
      'POST /api/v1/catalog/items',
    );

    // when
    const response = await platformApi.call(`/api/v1/catalog/items/${id}`, {
      method: 'PATCH',
      auth: foreignTenant.auth,
      tenant: foreignTenant.tenant,
      data: { data: { attributes: { name: 'Hacked' } } },
    });

    // then
    await expectStatus(response, 404, OPERATION);
    const lookup = await platformApi.call(`/api/v1/catalog/items/${id}`, {
      auth: sharedTenant.auth,
      tenant: sharedTenant.tenant,
    });
    const original = await readJson<{ data: { attributes: { name: string } } }>(lookup, OPERATION);
    expect(original.data.attributes.name, `${OPERATION}: original name must stay`).toBe('Do not patch');
  });
});
