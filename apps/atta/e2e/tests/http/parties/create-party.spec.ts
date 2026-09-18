import { expect } from '@playwright/test';
import { test } from '../../support/api.fixture';
import { expectStatus, readJson } from '../../support/http-assertions';

const OPERATION = 'POST /api/v1/parties';

test.describe(OPERATION, () => {
  test('crea la parte', async ({ sharedTenant, platformApi }) => {
    // given
    const { auth, tenant } = sharedTenant;
    const stamp = `${Date.now()}`;
    const name = `E2E Party ${stamp}`;
    const taxId = `900${stamp}`;

    // when
    const response = await platformApi.call('/api/v1/parties', {
      method: 'POST',
      auth,
      tenant,
      data: {
        data: {
          attributes: { name, tax_id: taxId, roles: ['supplier'] },
        },
      },
    });

    // then
    await expectStatus(response, 201, OPERATION);
    const payload = await readJson<{
      data: { type: string; id: string; attributes: { name: string; tax_id: string; roles: string[]; status: string } };
    }>(response, OPERATION);
    expect(payload.data.type, `${OPERATION}: data.type`).toBe('parties');
    expect(payload.data.id, `${OPERATION}: id`).toBeTruthy();
    expect(payload.data.attributes.name, `${OPERATION}: name`).toBe(name);
    expect(payload.data.attributes.tax_id, `${OPERATION}: tax_id`).toBe(taxId);
    expect(payload.data.attributes.roles, `${OPERATION}: roles`).toEqual(['supplier']);
    expect(payload.data.attributes.status, `${OPERATION}: status`).toBe('confirmed');
  });

  test('409 si el tax id ya existe', async ({ sharedTenant, platformApi }) => {
    // given
    const { auth, tenant } = sharedTenant;
    const taxId = `901${Date.now()}`;
    await expectStatus(
      await platformApi.call('/api/v1/parties', {
        method: 'POST',
        auth,
        tenant,
        data: { data: { attributes: { name: 'First party', tax_id: taxId, roles: ['customer'] } } },
      }),
      201,
      OPERATION,
    );

    // when
    const response = await platformApi.call('/api/v1/parties', {
      method: 'POST',
      auth,
      tenant,
      data: { data: { attributes: { name: 'Duplicate party', tax_id: taxId, roles: ['customer'] } } },
    });

    // then
    await expectStatus(response, 409, OPERATION);
    const payload = await readJson<{ errors: Array<{ code?: string; detail?: string }> }>(response, OPERATION);
    expect(payload.errors[0].code, `${OPERATION}: errors[0].code`).toBe('ERR_CONFLICT');
    expect(payload.errors[0].detail, `${OPERATION}: errors[0].detail`).toContain('tax id already exists');
  });

  test('400 si el rol es inválido', async ({ sharedTenant, platformApi }) => {
    // given
    const { auth, tenant } = sharedTenant;

    // when
    const response = await platformApi.call('/api/v1/parties', {
      method: 'POST',
      auth,
      tenant,
      data: { data: { attributes: { name: 'Evil', tax_id: `906${Date.now()}`, roles: ['admin'] } } },
    });

    // then
    await expectStatus(response, 400, OPERATION);
  });

  test('400 si roles está vacío', async ({ sharedTenant, platformApi }) => {
    // given
    const { auth, tenant } = sharedTenant;

    // when
    const response = await platformApi.call('/api/v1/parties', {
      method: 'POST',
      auth,
      tenant,
      data: { data: { attributes: { name: 'No roles', tax_id: `907${Date.now()}`, roles: [] } } },
    });

    // then
    await expectStatus(response, 400, OPERATION);
  });

  test('400 si falta tax id', async ({ sharedTenant, platformApi }) => {
    // given
    const { auth, tenant } = sharedTenant;

    // when
    const response = await platformApi.call('/api/v1/parties', {
      method: 'POST',
      auth,
      tenant,
      data: { data: { attributes: { name: 'No tax', tax_id: '', roles: ['supplier'] } } },
    });

    // then
    await expectStatus(response, 400, OPERATION);
  });
});
