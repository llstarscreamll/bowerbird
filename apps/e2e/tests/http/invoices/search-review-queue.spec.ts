import { expect } from '@playwright/test';
import { test } from '../../support/api.fixture';
import { expectStatus, readJson } from '../../support/http-assertions';

const OPERATION = 'GET /api/v1/invoicing/review-queue';

test.describe(OPERATION, () => {
  test('cola vacía en tenant nuevo', async ({ sharedTenant, platformApi }) => {
    // given
    const { auth, tenant } = sharedTenant;

    // when
    const response = await platformApi.call('/api/v1/invoicing/review-queue', { auth, tenant });

    // then
    await expectStatus(response, 200, OPERATION);
    const payload = await readJson<{ data: unknown[] }>(response, OPERATION);
    expect(payload.data, `${OPERATION}: data should be an empty collection`).toEqual([]);
  });
});
