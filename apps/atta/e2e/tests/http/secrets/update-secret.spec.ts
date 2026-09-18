import { expect } from '@playwright/test';
import { test } from '../../support/api.fixture';
import { expectStatus, readJson } from '../../support/http-assertions';

const OPERATION = 'PUT /api/v1/secrets/{id}';

test.describe(OPERATION, () => {
  test('actualiza el label', async ({ sharedTenant, platformApi }) => {
    // given
    const { auth, tenant } = sharedTenant;
    const created = await platformApi.call('/api/v1/secrets', {
      method: 'POST',
      auth,
      tenant,
      data: {
        data: {
          type: 'secrets',
          attributes: { purpose: 'generic.credential', label: `Before ${Date.now()}`, value: 'initial' },
        },
      },
    });
    await expectStatus(created, 201, 'POST /api/v1/secrets');
    const createdBody = await readJson<{ data: { id: string } }>(created, 'POST /api/v1/secrets');
    const label = `After ${Date.now()}`;

    // when
    const response = await platformApi.call(`/api/v1/secrets/${createdBody.data.id}`, {
      method: 'PUT',
      auth,
      tenant,
      data: { data: { type: 'secrets', attributes: { label } } },
    });

    // then
    await expectStatus(response, 200, OPERATION);
    const payload = await readJson<{ data: { attributes: { label: string } } }>(response, OPERATION);
    expect(payload.data.attributes.label, `${OPERATION}: label`).toBe(label);
  });
});
