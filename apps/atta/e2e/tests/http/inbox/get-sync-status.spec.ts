import { expect } from '@playwright/test';
import { test } from '../../support/api.fixture';
import { expectStatus, readJson } from '../../support/http-assertions';

const OPERATION = 'GET /api/v1/inbox/sync-status';

test.describe(OPERATION, () => {
  test('lista vacía en tenant nuevo', async ({ sharedTenant, platformApi }) => {
    // given
    const context = sharedTenant;

    // when
    const response = await platformApi.listInboxSyncStatus(context.auth, context.tenant);

    // then
    await expectStatus(response, 200, OPERATION);
    const payload = await readJson<unknown[]>(response, OPERATION);
    expect(payload, `${OPERATION}: body should be an empty list`).toEqual([]);
  });
});
