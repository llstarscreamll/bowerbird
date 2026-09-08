import { expect } from '@playwright/test';
import { test } from '../../support/api.fixture';
import { expectStatus, readJson } from '../../support/http-assertions';

const OPERATION = 'GET /api/v1/connections';

test.describe(OPERATION, () => {
  test('colección vacía en tenant nuevo', async ({ sharedTenant, platformApi }) => {
    // given
    const context = sharedTenant;

    // when
    const response = await platformApi.listConnections(context.auth, context.tenant);

    // then
    await expectStatus(response, 200, OPERATION);
    const payload = await readJson<{ data: unknown[] }>(response, OPERATION);
    expect(payload.data, `${OPERATION}: data should be an empty collection`).toEqual([]);
  });

  test('401 sin token', async ({ sharedTenant, platformApi }) => {
    // given
    const { tenant } = sharedTenant;

    // when
    const response = await platformApi.call('/api/v1/connections', { tenant });

    // then
    await expectStatus(response, 401, OPERATION);
  });
});
