import { expect } from '@playwright/test';
import { test } from '../../support/api.fixture';
import { expectStatus, readJson } from '../../support/http-assertions';
import type { AuthSession, PlatformApiClient, TenantContext } from '../../support/platform-api.client';
import { newUlid } from '../../support/ulid';

const OPERATION = 'POST /api/v1/catalog/items/not-duplicates';
const CREATE = 'POST /api/v1/catalog/items';
const CLUSTERS = 'GET /api/v1/catalog/items/duplicate-clusters';

async function createItem(platformApi: PlatformApiClient, auth: AuthSession, tenant: TenantContext, name: string, code: string): Promise<string> {
  const id = newUlid();
  await expectStatus(
    await platformApi.call('/api/v1/catalog/items', {
      method: 'POST',
      auth,
      tenant,
      data: { data: { type: 'catalog_items', id, attributes: { name, kind: 'goods', internal_code: code } } },
    }),
    201,
    CREATE,
  );
  return id;
}

test.describe(OPERATION, () => {
  test('el par deja de salir en clusters', async ({ sharedTenant, platformApi }) => {
    const { auth, tenant } = sharedTenant;
    const stamp = `${Date.now()}`;
    const name = `Not Dup ${stamp}`;
    const a = await createItem(platformApi, auth, tenant, name, `INT-NA-${stamp}`);
    const b = await createItem(platformApi, auth, tenant, name, `INT-NB-${stamp}`);

    const marked = await platformApi.call('/api/v1/catalog/items/not-duplicates', {
      method: 'POST',
      auth,
      tenant,
      data: { data: { type: 'catalog_item_not_duplicates', attributes: { item_ids: [a, b] } } },
    });
    await expectStatus(marked, 204, OPERATION);

    const clusters = await platformApi.call('/api/v1/catalog/items/duplicate-clusters', { auth, tenant });
    await expectStatus(clusters, 200, CLUSTERS);
    const payload = await readJson<{ data: Array<{ attributes: { item_ids: string[] } }> }>(clusters, CLUSTERS);
    const still = payload.data.some((row) => {
      const ids = new Set(row.attributes.item_ids);
      return ids.has(a) && ids.has(b);
    });
    expect(still, `${OPERATION}: pair must leave clusters`).toBe(false);
  });

  test('400 con menos de 2 ids', async ({ sharedTenant, platformApi }) => {
    const { auth, tenant } = sharedTenant;
    const response = await platformApi.call('/api/v1/catalog/items/not-duplicates', {
      method: 'POST',
      auth,
      tenant,
      data: { data: { type: 'catalog_item_not_duplicates', attributes: { item_ids: [newUlid()] } } },
    });
    await expectStatus(response, 400, OPERATION);
  });

  test('401 sin token', async ({ sharedTenant, platformApi }) => {
    const { tenant } = sharedTenant;
    const response = await platformApi.call('/api/v1/catalog/items/not-duplicates', {
      method: 'POST',
      tenant,
      data: { data: { type: 'catalog_item_not_duplicates', attributes: { item_ids: [newUlid(), newUlid()] } } },
    });
    await expectStatus(response, 401, OPERATION);
  });
});
