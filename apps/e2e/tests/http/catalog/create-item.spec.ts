import { expect } from '@playwright/test';
import { test } from '../../support/api.fixture';
import { expectClientError, expectStatus, readJson } from '../../support/http-assertions';
import { newUlid } from '../../support/ulid';

const OPERATION = 'POST /api/v1/catalog/items';

test.describe(OPERATION, () => {
  test('crea el ítem', async ({ sharedTenant, platformApi }) => {
    // given
    const { auth, tenant } = sharedTenant;
    const id = newUlid();
    const stamp = `${Date.now()}`;
    const name = `E2E Item ${stamp}`;
    const sku = `SKU-${stamp}`;

    // when
    const response = await platformApi.call('/api/v1/catalog/items', {
      method: 'POST',
      auth,
      tenant,
      data: {
        data: {
          type: 'catalog_items',
          id,
          attributes: { name, kind: 'goods', internal_sku: sku },
        },
      },
    });

    // then
    await expectStatus(response, 201, OPERATION);
    const payload = await readJson<{
      data: { type: string; id: string; attributes: { name: string; kind: string; internal_sku: string | null; status: string } };
    }>(response, OPERATION);
    expect(payload.data.type, `${OPERATION}: data.type`).toBe('catalog_items');
    expect(payload.data.id, `${OPERATION}: data.id`).toBe(id);
    expect(payload.data.attributes.name, `${OPERATION}: name`).toBe(name);
    expect(payload.data.attributes.kind, `${OPERATION}: kind`).toBe('goods');
    expect(payload.data.attributes.internal_sku, `${OPERATION}: internal_sku`).toBe(sku);
    expect(payload.data.attributes.status, `${OPERATION}: status`).toBe('confirmed');
  });

  test('400 si el kind es inválido', async ({ sharedTenant, platformApi }) => {
    // given
    const { auth, tenant } = sharedTenant;

    // when
    const response = await platformApi.call('/api/v1/catalog/items', {
      method: 'POST',
      auth,
      tenant,
      data: {
        data: {
          type: 'catalog_items',
          id: newUlid(),
          attributes: { name: 'Bad kind', kind: 'widget', internal_sku: `SKU-BAD-${Date.now()}` },
        },
      },
    });

    // then
    await expectStatus(response, 400, OPERATION);
    const payload = await readJson<{ errors: Array<{ code?: string; detail?: string }> }>(response, OPERATION);
    expect(payload.errors[0].code, `${OPERATION}: errors[0].code`).toBe('ERR_VALIDATION');
    expect(payload.errors[0].detail, `${OPERATION}: errors[0].detail`).toContain('invalid item kind');
  });

  test('400 si el nombre está vacío', async ({ sharedTenant, platformApi }) => {
    // given
    const { auth, tenant } = sharedTenant;

    // when
    const response = await platformApi.call('/api/v1/catalog/items', {
      method: 'POST',
      auth,
      tenant,
      data: {
        data: {
          type: 'catalog_items',
          id: newUlid(),
          attributes: { name: '   ', kind: 'goods', internal_sku: `SKU-EMPTY-${Date.now()}` },
        },
      },
    });

    // then
    await expectStatus(response, 400, OPERATION);
  });

  test('rechaza JSON malformado', async ({ sharedTenant, platformApi }) => {
    // given
    const { auth, tenant } = sharedTenant;

    // when
    const response = await platformApi.call('/api/v1/catalog/items', {
      method: 'POST',
      auth,
      tenant,
      rawBody: '{"data":',
      contentType: 'application/json',
    });

    // then
    await expectClientError(response, OPERATION);
  });

  test('409 si reusa el mismo id en el tenant', async ({ sharedTenant, platformApi }) => {
    // given
    const { auth, tenant } = sharedTenant;
    const id = newUlid();
    const payload = {
      data: {
        type: 'catalog_items',
        id,
        attributes: { name: 'Dup', kind: 'goods', internal_sku: `SKU-DUP-${Date.now()}` },
      },
    };
    await expectStatus(await platformApi.call('/api/v1/catalog/items', { method: 'POST', auth, tenant, data: payload }), 201, OPERATION);

    // when
    const response = await platformApi.call('/api/v1/catalog/items', {
      method: 'POST',
      auth,
      tenant,
      data: {
        data: {
          type: 'catalog_items',
          id,
          attributes: { name: 'Dup 2', kind: 'goods', internal_sku: `SKU-DUP2-${Date.now()}` },
        },
      },
    });

    // then
    await expectStatus(response, 409, OPERATION);
  });

  test('el mismo id en otro tenant no choca', async ({ sharedTenant, foreignTenant, platformApi }) => {
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
            attributes: { name: 'Tenant A', kind: 'goods', internal_sku: `SKU-A-${Date.now()}` },
          },
        },
      }),
      201,
      OPERATION,
    );

    // when
    const response = await platformApi.call('/api/v1/catalog/items', {
      method: 'POST',
      auth: foreignTenant.auth,
      tenant: foreignTenant.tenant,
      data: {
        data: {
          type: 'catalog_items',
          id,
          attributes: { name: 'Tenant B', kind: 'goods', internal_sku: `SKU-B-${Date.now()}` },
        },
      },
    });

    // then
    await expectStatus(response, 201, OPERATION);
  });
});
