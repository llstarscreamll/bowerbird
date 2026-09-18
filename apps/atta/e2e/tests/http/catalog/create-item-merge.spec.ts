import { expect } from '@playwright/test';
import { test } from '../../support/api.fixture';
import { expectClientError, expectStatus, readJson } from '../../support/http-assertions';
import type { AuthSession, PlatformApiClient, TenantContext } from '../../support/platform-api.client';
import { newUlid } from '../../support/ulid';

const OPERATION = 'POST /api/v1/catalog/items/merges';
const CREATE = 'POST /api/v1/catalog/items';
const ALIAS = 'POST /api/v1/catalog/items/{id}/aliases';

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

async function createParty(platformApi: PlatformApiClient, auth: AuthSession, tenant: TenantContext, stamp: string, suffix: string): Promise<string> {
  const res = await platformApi.call('/api/v1/parties', {
    method: 'POST',
    auth,
    tenant,
    data: { data: { attributes: { name: `Merge Party ${suffix} ${stamp}`, tax_id: `90${suffix}${stamp}`.slice(0, 16), roles: ['supplier'] } } },
  });
  await expectStatus(res, 201, 'POST /api/v1/parties');
  const body = await readJson<{ data: { id: string } }>(res, 'POST /api/v1/parties');
  return body.data.id;
}

async function addAlias(platformApi: PlatformApiClient, auth: AuthSession, tenant: TenantContext, itemId: string, attrs: { scheme: string; value: string; party_id?: string }): Promise<void> {
  const res = await platformApi.call(`/api/v1/catalog/items/${itemId}/aliases`, {
    method: 'POST',
    auth,
    tenant,
    data: { data: { type: 'catalog_item_aliases', id: newUlid(), attributes: attrs } },
  });
  await expectStatus(res, 201, ALIAS);
}

test.describe(OPERATION, () => {
  test('fusiona aliases y oculta el unificado', async ({ sharedTenant, platformApi }) => {
    const { auth, tenant } = sharedTenant;
    const stamp = `${Date.now()}`;
    const name = `Merge Item ${stamp}`;
    const survivor = await createItem(platformApi, auth, tenant, name, `INT-MA-${stamp}`);
    const source = await createItem(platformApi, auth, tenant, `${name} B`, `INT-MB-${stamp}`);
    const partyA = await createParty(platformApi, auth, tenant, stamp, '1');
    const partyB = await createParty(platformApi, auth, tenant, stamp, '2');
    await addAlias(platformApi, auth, tenant, survivor, { scheme: 'supplier_sku', value: `SKU-A-${stamp}`, party_id: partyA });
    await addAlias(platformApi, auth, tenant, source, { scheme: 'supplier_sku', value: `SKU-B-${stamp}`, party_id: partyB });
    const gtin = stamp.padStart(13, '0').slice(0, 13);
    await addAlias(platformApi, auth, tenant, source, { scheme: 'gtin', value: gtin });

    const merged = await platformApi.call('/api/v1/catalog/items/merges', {
      method: 'POST',
      auth,
      tenant,
      data: {
        data: {
          type: 'catalog_item_merges',
          attributes: { survivor_id: survivor, source_ids: [source], name },
        },
      },
    });
    await expectStatus(merged, 200, OPERATION);
    const body = await readJson<{
      data: { id: string; attributes: { name: string; aliases: Array<{ scheme: string; value: string; source: string }> } };
    }>(merged, OPERATION);
    expect(body.data.id).toBe(survivor);
    expect(body.data.attributes.name).toBe(name);
    const aliases = body.data.attributes.aliases || [];
    expect(aliases.some((a) => a.scheme === 'supplier_sku' && a.value === `SKU-A-${stamp}`)).toBe(true);
    expect(aliases.some((a) => a.scheme === 'supplier_sku' && a.value === `SKU-B-${stamp}`)).toBe(true);
    expect(aliases.some((a) => a.scheme === 'gtin' && a.value === gtin && a.source === 'manual')).toBe(true);

    const list = await platformApi.call('/api/v1/catalog/items?search=' + encodeURIComponent(name), { auth, tenant });
    await expectStatus(list, 200, 'GET /api/v1/catalog/items');
    const listBody = await readJson<{ data: Array<{ id: string }> }>(list, 'GET /api/v1/catalog/items');
    expect(listBody.data.map((row) => row.id)).toContain(survivor);
    expect(listBody.data.map((row) => row.id)).not.toContain(source);

    const gone = await platformApi.call(`/api/v1/catalog/items/${source}`, { auth, tenant });
    await expectStatus(gone, 410, 'GET /api/v1/catalog/items/{id}');
    const goneBody = await readJson<{ errors: Array<{ code?: string; meta?: { merged_into_id?: string } }> }>(gone, 'GET /api/v1/catalog/items/{id}');
    expect(goneBody.errors[0].code).toBe('ERR_GONE');
    expect(goneBody.errors[0].meta?.merged_into_id).toBe(survivor);
  });

  test('400 si falta source_ids', async ({ sharedTenant, platformApi }) => {
    const { auth, tenant } = sharedTenant;
    const stamp = `${Date.now()}`;
    const survivor = await createItem(platformApi, auth, tenant, `Merge Solo ${stamp}`, `INT-MS-${stamp}`);
    const response = await platformApi.call('/api/v1/catalog/items/merges', {
      method: 'POST',
      auth,
      tenant,
      data: { data: { type: 'catalog_item_merges', attributes: { survivor_id: survivor, source_ids: [] } } },
    });
    await expectStatus(response, 400, OPERATION);
  });

  test('400 si el survivor no existe', async ({ sharedTenant, platformApi }) => {
    const { auth, tenant } = sharedTenant;
    const stamp = `${Date.now()}`;
    const source = await createItem(platformApi, auth, tenant, `Merge Missing ${stamp}`, `INT-MM-${stamp}`);
    const response = await platformApi.call('/api/v1/catalog/items/merges', {
      method: 'POST',
      auth,
      tenant,
      data: {
        data: {
          type: 'catalog_item_merges',
          attributes: { survivor_id: newUlid(), source_ids: [source] },
        },
      },
    });
    await expectStatus(response, 400, OPERATION);
  });

  test('401 sin token', async ({ sharedTenant, platformApi }) => {
    const { tenant } = sharedTenant;
    const response = await platformApi.call('/api/v1/catalog/items/merges', {
      method: 'POST',
      tenant,
      data: { data: { type: 'catalog_item_merges', attributes: { survivor_id: newUlid(), source_ids: [newUlid()] } } },
    });
    await expectStatus(response, 401, OPERATION);
  });

  test('rechaza JSON malformado', async ({ sharedTenant, platformApi }) => {
    const { auth, tenant } = sharedTenant;
    const response = await platformApi.call('/api/v1/catalog/items/merges', {
      method: 'POST',
      auth,
      tenant,
      rawBody: '{"data":',
      contentType: 'application/json',
    });
    await expectClientError(response, OPERATION);
  });
});
