import { expect } from '@playwright/test';
import { test } from '../../support/api.fixture';
import { expectStatus, readJson } from '../../support/http-assertions';

const OPERATION = 'POST /api/v1/legal-entities';

test.describe(OPERATION, () => {
  test('crea la entidad legal y rechaza una segunda', async ({ sharedTenant, platformApi }) => {
    const { auth } = sharedTenant;
    const stamp = `${Date.now()}`;
    const tenant = await platformApi.createTenantOrFail(auth, {
      name: `E2E Legal ${stamp}`,
      slug: `e2e-legal-${stamp}`,
    });

    const listEmpty = await platformApi.call('/api/v1/legal-entities', { method: 'GET', auth, tenant });
    await expectStatus(listEmpty, 200, 'GET /api/v1/legal-entities');
    const emptyPayload = await readJson<{ data: unknown[] }>(listEmpty, 'GET /api/v1/legal-entities');
    expect(emptyPayload.data, 'GET /api/v1/legal-entities: data').toEqual([]);

    const created = await platformApi.call('/api/v1/legal-entities', {
      method: 'POST',
      auth,
      tenant,
      data: {
        data: {
          attributes: { tax_id: `900.${stamp.slice(-6)}`, scheme_id: '31', legal_name: 'Acme SAS' },
        },
      },
    });
    await expectStatus(created, 201, OPERATION);
    const payload = await readJson<{
      data: { type: string; id: string; attributes: { tax_id: string; scheme_id: string; legal_name: string } };
    }>(created, OPERATION);
    expect(payload.data.type, `${OPERATION}: data.type`).toBe('legal-entities');
    expect(payload.data.id, `${OPERATION}: id`).toBeTruthy();
    expect(payload.data.attributes.tax_id, `${OPERATION}: tax_id`).toMatch(/^\d+$/);
    expect(payload.data.attributes.scheme_id, `${OPERATION}: scheme_id`).toBe('31');
    expect(payload.data.attributes.legal_name, `${OPERATION}: legal_name`).toBe('Acme SAS');

    const duplicate = await platformApi.call('/api/v1/legal-entities', {
      method: 'POST',
      auth,
      tenant,
      data: {
        data: {
          attributes: { tax_id: '901000111', scheme_id: '31', legal_name: 'Other SAS' },
        },
      },
    });
    await expectStatus(duplicate, 409, OPERATION);
    const conflict = await readJson<{ errors: Array<{ code?: string }> }>(duplicate, OPERATION);
    expect(conflict.errors[0].code, `${OPERATION}: errors[0].code`).toBe('ERR_CONFLICT');

    const patched = await platformApi.call(`/api/v1/legal-entities/${payload.data.id}`, {
      method: 'PATCH',
      auth,
      tenant,
      data: { data: { attributes: { legal_name: 'Acme S.A.S.' } } },
    });
    await expectStatus(patched, 200, 'PATCH /api/v1/legal-entities/{id}');
    const updated = await readJson<{ data: { attributes: { legal_name: string } } }>(patched, 'PATCH /api/v1/legal-entities/{id}');
    expect(updated.data.attributes.legal_name, 'PATCH: legal_name').toBe('Acme S.A.S.');
  });
});
