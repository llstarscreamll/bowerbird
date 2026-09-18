import { expect } from '@playwright/test';
import { test } from '../../support/api.fixture';
import { expectStatus, readJson } from '../../support/http-assertions';

const OPERATION = 'PATCH /api/v1/parties/{id}';

test.describe(OPERATION, () => {
  test('actualiza el nombre', async ({ sharedTenant, platformApi }) => {
    // given
    const { auth, tenant } = sharedTenant;
    const created = await platformApi.call('/api/v1/parties', {
      method: 'POST',
      auth,
      tenant,
      data: { data: { attributes: { name: `Before ${Date.now()}`, tax_id: `904${Date.now()}`, roles: ['supplier'] } } },
    });
    await expectStatus(created, 201, 'POST /api/v1/parties');
    const createdBody = await readJson<{ data: { id: string } }>(created, 'POST /api/v1/parties');
    const name = `After ${Date.now()}`;

    // when
    const response = await platformApi.call(`/api/v1/parties/${createdBody.data.id}`, {
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

  test('ignora un tax_id distinto en el patch', async ({ sharedTenant, platformApi }) => {
    // given
    const { auth, tenant } = sharedTenant;
    const taxId = `905${Date.now()}`;
    const created = await platformApi.call('/api/v1/parties', {
      method: 'POST',
      auth,
      tenant,
      data: { data: { attributes: { name: `Keep tax ${Date.now()}`, tax_id: taxId, roles: ['supplier'] } } },
    });
    await expectStatus(created, 201, 'POST /api/v1/parties');
    const createdBody = await readJson<{ data: { id: string } }>(created, 'POST /api/v1/parties');

    // when
    const response = await platformApi.call(`/api/v1/parties/${createdBody.data.id}`, {
      method: 'PATCH',
      auth,
      tenant,
      data: { data: { attributes: { name: 'Renamed', tax_id: `999${Date.now()}` } } },
    });

    // then
    await expectStatus(response, 200, OPERATION);
    const payload = await readJson<{ data: { attributes: { tax_id: string; name: string } } }>(response, OPERATION);
    expect(payload.data.attributes.tax_id, `${OPERATION}: tax_id must stay immutable`).toBe(taxId);
    expect(payload.data.attributes.name, `${OPERATION}: name`).toBe('Renamed');
  });
});
