import { expect } from '@playwright/test';
import { test } from '../../support/api.fixture';
import { expectStatus, readJson } from '../../support/http-assertions';
import { newUlid } from '../../support/ulid';

const TEMPLATE = 'GET /api/v1/catalog/imports/template';
const CREATE = 'POST /api/v1/catalog/imports';
const LIST = 'GET /api/v1/catalog/imports';
const GET = 'GET /api/v1/catalog/imports/{id}';
const ERRORS = 'GET /api/v1/catalog/imports/{id}/errors';
const CANCEL = 'POST /api/v1/catalog/imports/{id}/cancel';
const ITEMS = 'GET /api/v1/catalog/items';

test.describe('catalog imports', () => {
  test('descarga la plantilla CSV', async ({ sharedTenant, platformApi }) => {
    const { auth, tenant } = sharedTenant;
    const response = await platformApi.call('/api/v1/catalog/imports/template', { auth, tenant });
    await expectStatus(response, 200, TEMPLATE);
    const body = await response.text();
    expect(body, `${TEMPLATE}: header`).toContain('internal_code,name,kind');
  });

  test('400 si falta file_key', async ({ sharedTenant, platformApi }) => {
    const { auth, tenant } = sharedTenant;
    const response = await platformApi.call('/api/v1/catalog/imports', {
      method: 'POST',
      auth,
      tenant,
      data: { data: { type: 'catalog_imports', id: newUlid(), attributes: { file_key: '' } } },
    });
    await expectStatus(response, 400, CREATE);
  });

  test('400 si el archivo no existe', async ({ sharedTenant, platformApi }) => {
    const { auth, tenant } = sharedTenant;
    const response = await platformApi.call('/api/v1/catalog/imports', {
      method: 'POST',
      auth,
      tenant,
      data: {
        data: {
          type: 'catalog_imports',
          id: newUlid(),
          attributes: { file_key: `1-day/tenants/${tenant.tenantSlug}/uploads/catalog/missing.csv` },
        },
      },
    });
    await expectStatus(response, 400, CREATE);
  });

  test('encola import, copia requester y pagina errores', async ({ sharedTenant, platformApi, playwright }) => {
    const { auth, tenant } = sharedTenant;
    const csv = 'internal_code,name,kind\nE2E-IMP,Item importado,bien\n,Sin código,bien\n';
    const presign = await platformApi.call('/api/v1/files/uploads/presigned', {
      method: 'POST',
      auth,
      tenant,
      data: { filename: 'catalog.csv', content_type: 'text/csv', module: 'catalog' },
    });
    await expectStatus(presign, 200, 'POST /api/v1/files/uploads/presigned');
    const upload = await readJson<{ url: string; method: string; headers: Record<string, string>; reference: { key: string } }>(presign, 'POST /api/v1/files/uploads/presigned');
    const storage = await playwright.request.newContext({ ignoreHTTPSErrors: true });
    try {
      const put = await storage.fetch(upload.url, { method: upload.method, headers: upload.headers, data: csv });
      expect(put.status(), 'PUT csv').toBeLessThan(300);
    } finally {
      await storage.dispose();
    }

    const importId = newUlid();
    const created = await platformApi.call('/api/v1/catalog/imports', {
      method: 'POST',
      auth,
      tenant,
      data: { data: { type: 'catalog_imports', id: importId, attributes: { file_key: upload.reference.key } } },
    });
    await expectStatus(created, 202, CREATE);
    const payload = await readJson<{
      data: { type: string; id: string; attributes: { status: string; requested_by: { email: string; name: string }; cancelled_by: unknown } };
    }>(created, CREATE);
    expect(payload.data.type).toBe('catalog_imports');
    expect(payload.data.id).toBe(importId);
    expect(payload.data.attributes.requested_by.email).toBeTruthy();
    expect(payload.data.attributes.cancelled_by).toBeNull();

    const detail = await platformApi.call(`/api/v1/catalog/imports/${importId}`, { auth, tenant });
    await expectStatus(detail, 200, GET);
    const detailPayload = await readJson<{ data: { attributes: { failed_count: number } } }>(detail, GET);
    expect(detailPayload.data.attributes.failed_count, `${GET}: no embebe errores`).toBeDefined();

    const errors = await platformApi.call(`/api/v1/catalog/imports/${importId}/errors?page[size]=50`, { auth, tenant });
    await expectStatus(errors, 200, ERRORS);
    const errorPayload = await readJson<{ data: unknown[]; meta: { total: number } }>(errors, ERRORS);
    expect(Array.isArray(errorPayload.data)).toBe(true);
    expect(errorPayload.meta.total).toBeGreaterThanOrEqual(0);

    const list = await platformApi.call('/api/v1/catalog/imports?page[size]=20', { auth, tenant });
    await expectStatus(list, 200, LIST);
    const listPayload = await readJson<{ data: Array<{ id: string }>; meta: { has_more: boolean } }>(list, LIST);
    expect(listPayload.data.map((row) => row.id)).toContain(importId);

    const cancel = await platformApi.call(`/api/v1/catalog/imports/${importId}/cancel`, { method: 'POST', auth, tenant });
    expect([200, 409], `${CANCEL}: active or already terminal`).toContain(cancel.status());
    if (cancel.status() === 200) {
      const cancelled = await readJson<{ data: { attributes: { status: string; cancelled_by: { email: string } | null } } }>(cancel, CANCEL);
      expect(cancelled.data.attributes.status).toBe('cancelled');
      expect(cancelled.data.attributes.cancelled_by?.email).toBeTruthy();
    }
  });

  test('401 sin token', async ({ sharedTenant, platformApi }) => {
    const { tenant } = sharedTenant;
    const response = await platformApi.call('/api/v1/catalog/imports', { tenant });
    await expectStatus(response, 401, LIST);
  });
});

test.describe(ITEMS, () => {
  test('pagina el listado con page[size]', async ({ sharedTenant, platformApi }) => {
    const { auth, tenant } = sharedTenant;
    const stamp = `${Date.now()}`;
    for (const suffix of ['A', 'B']) {
      await expectStatus(
        await platformApi.call('/api/v1/catalog/items', {
          method: 'POST',
          auth,
          tenant,
          data: {
            data: {
              type: 'catalog_items',
              id: newUlid(),
              attributes: { name: `Paged ${stamp} ${suffix}`, kind: 'goods', internal_code: `PAGE-${stamp}-${suffix}` },
            },
          },
        }),
        201,
        'POST /api/v1/catalog/items',
      );
    }
    const response = await platformApi.call(`/api/v1/catalog/items?page[size]=1&search=${encodeURIComponent(`Paged ${stamp}`)}`, {
      auth,
      tenant,
    });
    await expectStatus(response, 200, ITEMS);
    const payload = await readJson<{ data: unknown[]; meta: { has_more: boolean; cursor?: string } }>(response, ITEMS);
    expect(payload.data).toHaveLength(1);
    expect(payload.meta.has_more).toBe(true);
    expect(payload.meta.cursor).toBeTruthy();
  });
});
