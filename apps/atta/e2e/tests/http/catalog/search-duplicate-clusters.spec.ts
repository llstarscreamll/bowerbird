import { expect } from '@playwright/test';
import { test } from '../../support/api.fixture';
import { expectStatus, readJson } from '../../support/http-assertions';
import type { AuthSession, PlatformApiClient, TenantContext } from '../../support/platform-api.client';
import { newUlid } from '../../support/ulid';

const OPERATION = 'GET /api/v1/catalog/items/duplicate-clusters';
const CREATE = 'POST /api/v1/catalog/items';

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
  test('agrupa ítems con el mismo nombre', async ({ sharedTenant, platformApi }) => {
    const { auth, tenant } = sharedTenant;
    const stamp = `${Date.now()}`;
    const name = `Dup Cluster ${stamp}`;
    const a = await createItem(platformApi, auth, tenant, name, `INT-CA-${stamp}`);
    const b = await createItem(platformApi, auth, tenant, name, `INT-CB-${stamp}`);

    const response = await platformApi.call('/api/v1/catalog/items/duplicate-clusters', { auth, tenant });
    await expectStatus(response, 200, OPERATION);
    const payload = await readJson<{
      data: Array<{ type: string; attributes: { reason: string; item_ids: string[] } }>;
    }>(response, OPERATION);
    const cluster = payload.data.find((row) => {
      const ids = new Set(row.attributes.item_ids);
      return ids.has(a) && ids.has(b);
    });
    expect(cluster, `${OPERATION}: cluster for ${name}`).toBeTruthy();
    expect(cluster?.type).toBe('catalog_duplicate_clusters');
    expect(cluster?.attributes.reason).toBe('normalized_description');
  });

  test('401 sin token', async ({ sharedTenant, platformApi }) => {
    const { tenant } = sharedTenant;
    const response = await platformApi.call('/api/v1/catalog/items/duplicate-clusters', { tenant });
    await expectStatus(response, 401, OPERATION);
  });

  test('no mezcla tenants', async ({ sharedTenant, foreignTenant, platformApi }) => {
    const stamp = `${Date.now()}`;
    const name = `Iso Cluster ${stamp}`;
    const a = await createItem(platformApi, sharedTenant.auth, sharedTenant.tenant, name, `INT-IA-${stamp}`);
    await createItem(platformApi, sharedTenant.auth, sharedTenant.tenant, name, `INT-IB-${stamp}`);
    await createItem(platformApi, foreignTenant.auth, foreignTenant.tenant, name, `INT-FA-${stamp}`);
    await createItem(platformApi, foreignTenant.auth, foreignTenant.tenant, name, `INT-FB-${stamp}`);

    const response = await platformApi.call('/api/v1/catalog/items/duplicate-clusters', {
      auth: foreignTenant.auth,
      tenant: foreignTenant.tenant,
    });
    await expectStatus(response, 200, OPERATION);
    const payload = await readJson<{ data: Array<{ attributes: { item_ids: string[] } }> }>(response, OPERATION);
    const leaked = payload.data.some((row) => row.attributes.item_ids.includes(a));
    expect(leaked, `${OPERATION}: must not leak shared tenant item`).toBe(false);
  });
});
