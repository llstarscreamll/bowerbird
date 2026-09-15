import { expect } from '@playwright/test';
import { test } from '../../support/api.fixture';
import { expectStatus, readJson } from '../../support/http-assertions';
import { newUlid } from '../../support/ulid';

const OPERATION = 'POST /api/v1/catalog/items/{id}/aliases';

test.describe(OPERATION, () => {
  test('CRUD de aliases y conflicto con item_id dueño', async ({ sharedTenant, platformApi }) => {
    const { auth, tenant } = sharedTenant;
    const itemA = newUlid();
    const itemB = newUlid();
    const aliasId = newUlid();
    const stamp = `${Date.now()}`;

    const party = await platformApi.call('/api/v1/parties', {
      method: 'POST',
      auth,
      tenant,
      data: { data: { attributes: { name: `Alias Party ${stamp}`, tax_id: `901${stamp}`, roles: ['supplier'] } } },
    });
    await expectStatus(party, 201, 'POST /api/v1/parties');
    const partyBody = await readJson<{ data: { id: string } }>(party, 'POST /api/v1/parties');

    await expectStatus(
      await platformApi.call('/api/v1/catalog/items', {
        method: 'POST',
        auth,
        tenant,
        data: { data: { type: 'catalog_items', id: itemA, attributes: { name: 'Item A', kind: 'goods', internal_code: `INT-A-${stamp}` } } },
      }),
      201,
      'POST /api/v1/catalog/items',
    );
    await expectStatus(
      await platformApi.call('/api/v1/catalog/items', {
        method: 'POST',
        auth,
        tenant,
        data: { data: { type: 'catalog_items', id: itemB, attributes: { name: 'Item B', kind: 'goods', internal_code: `INT-B-${stamp}` } } },
      }),
      201,
      'POST /api/v1/catalog/items',
    );

    const created = await platformApi.call(`/api/v1/catalog/items/${itemA}/aliases`, {
      method: 'POST',
      auth,
      tenant,
      data: {
        data: {
          type: 'catalog_item_aliases',
          id: aliasId,
          attributes: { scheme: 'supplier_sku', value: `SKU-${stamp}`, party_id: partyBody.data.id },
        },
      },
    });
    await expectStatus(created, 201, OPERATION);
    const createdBody = await readJson<{ data: { id: string; attributes: { scheme: string; source: string; value: string } } }>(created, OPERATION);
    expect(createdBody.data.id).toBe(aliasId);
    expect(createdBody.data.attributes.scheme).toBe('supplier_sku');
    expect(createdBody.data.attributes.source).toBe('manual');

    const detail = await platformApi.call(`/api/v1/catalog/items/${itemA}`, { auth, tenant });
    await expectStatus(detail, 200, 'GET /api/v1/catalog/items/{id}');
    const detailBody = await readJson<{ data: { attributes: { aliases: Array<{ id: string; scheme: string; value: string }> } } }>(detail, 'GET /api/v1/catalog/items/{id}');
    expect(detailBody.data.attributes.aliases.some((a) => a.id === aliasId)).toBe(true);

    const conflict = await platformApi.call(`/api/v1/catalog/items/${itemB}/aliases`, {
      method: 'POST',
      auth,
      tenant,
      data: {
        data: {
          type: 'catalog_item_aliases',
          id: newUlid(),
          attributes: { scheme: 'supplier_sku', value: `SKU-${stamp}`, party_id: partyBody.data.id },
        },
      },
    });
    await expectStatus(conflict, 409, OPERATION);
    const conflictBody = await readJson<{ errors: Array<{ code?: string; meta?: { item_id?: string } }> }>(conflict, OPERATION);
    expect(conflictBody.errors[0].code).toBe('ERR_CONFLICT');
    expect(conflictBody.errors[0].meta?.item_id).toBe(itemA);

    const deleted = await platformApi.call(`/api/v1/catalog/items/${itemA}/aliases/${aliasId}`, {
      method: 'DELETE',
      auth,
      tenant,
    });
    await expectStatus(deleted, 204, 'DELETE /api/v1/catalog/items/{id}/aliases/{aliasId}');
  });
});
