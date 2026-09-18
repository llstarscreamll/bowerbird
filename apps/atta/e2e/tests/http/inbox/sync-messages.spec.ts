import { expect } from '@playwright/test';
import { test } from '../../support/api.fixture';
import { expectStatus, readJson } from '../../support/http-assertions';

const OPERATION = 'POST /api/v1/inbox/sync';

test.describe(OPERATION, () => {
  test('acepta sync cuando no hay cuentas activas', async ({ sharedTenant, platformApi }) => {
    // given
    const context = sharedTenant;

    // when
    const response = await platformApi.triggerInboxSync(context.auth, context.tenant);

    // then
    await expectStatus(response, 202, OPERATION);
    const payload = await readJson<{ message: string }>(response, OPERATION);
    expect(payload.message, `${OPERATION}: message`).toBe('Sync triggered');
  });

  test('acepta sync con account_id desconocido', async ({ sharedTenant, platformApi }) => {
    // given
    const context = sharedTenant;

    // when
    const response = await platformApi.triggerInboxSync(context.auth, context.tenant, 'missing-account-id');

    // then
    await expectStatus(response, 202, OPERATION);
  });
});
