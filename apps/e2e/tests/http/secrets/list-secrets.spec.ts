import { expect } from '@playwright/test';
import { test } from '../../support/api.fixture';
import { expectStatus, readJson } from '../../support/http-assertions';

const OPERATION = 'GET /api/v1/secrets';

test.describe(OPERATION, () => {
  test('lista el secreto por purpose', async ({ sharedTenant, platformApi }) => {
    // given
    const { auth, tenant } = sharedTenant;
    const label = `List secret ${Date.now()}`;
    const created = await platformApi.call('/api/v1/secrets', {
      method: 'POST',
      auth,
      tenant,
      data: {
        data: {
          type: 'secrets',
          attributes: {
            purpose: 'integrations.api_key',
            label,
            value: 'api-key-value',
          },
        },
      },
    });
    await expectStatus(created, 201, 'POST /api/v1/secrets');
    const createdBody = await readJson<{ data: { id: string } }>(created, 'POST /api/v1/secrets');

    // when
    const response = await platformApi.call('/api/v1/secrets?purpose=integrations.api_key', { auth, tenant });

    // then
    await expectStatus(response, 200, OPERATION);
    const payload = await readJson<{ data: Array<{ id: string; attributes: { purpose: string; label: string } }> }>(response, OPERATION);
    const found = payload.data.find((item) => item.id === createdBody.data.id);
    expect(found, `${OPERATION}: created secret`).toBeTruthy();
    expect(found?.attributes.purpose, `${OPERATION}: purpose`).toBe('integrations.api_key');
    expect(found?.attributes.label, `${OPERATION}: label`).toBe(label);
  });
});
