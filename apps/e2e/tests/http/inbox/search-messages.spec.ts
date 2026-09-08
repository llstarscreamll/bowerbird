import { expect } from '@playwright/test';
import { test } from '../../support/api.fixture';
import { expectStatus, readJson } from '../../support/http-assertions';

const OPERATION = 'GET /api/v1/inbox/messages';

test.describe(OPERATION, () => {
  test('colección vacía en tenant nuevo', async ({ sharedTenant, platformApi }) => {
    // given
    const context = sharedTenant;

    // when
    const response = await platformApi.listInboxMessages(context.auth, context.tenant);

    // then
    await expectStatus(response, 200, OPERATION);
    const payload = await readJson<{ data: unknown[]; total: number }>(response, OPERATION);
    expect(payload.data, `${OPERATION}: data should be an empty collection`).toEqual([]);
    expect(payload.total, `${OPERATION}: total should be 0`).toBe(0);
  });

  test('401 sin token', async ({ sharedTenant, platformApi }) => {
    // given
    const { tenant } = sharedTenant;

    // when
    const response = await platformApi.call('/api/v1/inbox/messages', { tenant });

    // then
    await expectStatus(response, 401, OPERATION);
  });
});
