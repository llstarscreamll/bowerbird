import { expect } from '@playwright/test';
import { test } from '../../support/api.fixture';
import { expectStatus, readJson } from '../../support/http-assertions';

const OPERATION = 'GET /api/v1/parties';

test.describe(OPERATION, () => {
  test('encuentra la parte por tax id', async ({ sharedTenant, platformApi }) => {
    // given
    const { auth, tenant } = sharedTenant;
    const stamp = `${Date.now()}`;
    const taxId = `902${stamp}`;
    const created = await platformApi.call('/api/v1/parties', {
      method: 'POST',
      auth,
      tenant,
      data: { data: { attributes: { name: `Search Party ${stamp}`, tax_id: taxId, roles: ['supplier'] } } },
    });
    await expectStatus(created, 201, 'POST /api/v1/parties');
    const createdBody = await readJson<{ data: { id: string } }>(created, 'POST /api/v1/parties');

    // when
    const response = await platformApi.call(`/api/v1/parties?search=${encodeURIComponent(taxId)}`, { auth, tenant });

    // then
    await expectStatus(response, 200, OPERATION);
    const payload = await readJson<{ data: Array<{ id: string; attributes: { tax_id: string } }> }>(response, OPERATION);
    expect(
      payload.data.map((item) => item.id),
      `${OPERATION}: ids`,
    ).toContain(createdBody.data.id);
    expect(payload.data.find((item) => item.id === createdBody.data.id)?.attributes.tax_id, `${OPERATION}: tax_id`).toBe(taxId);
  });
});
