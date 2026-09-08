import { expect } from '@playwright/test';
import { test } from '../../support/api.fixture';
import { expectStatus, readJson } from '../../support/http-assertions';

const OPERATION = 'POST /api/v1/inbox/messages';

test.describe(OPERATION, () => {
  test('403 sin entitlement mail.send', async ({ sharedTenant, platformApi }) => {
    // given
    const context = sharedTenant;

    // when
    const response = await platformApi.sendInboxMessage(context.auth, context.tenant, {
      account_id: 'missing-account',
      to: ['someone@example.com'],
      subject: 'Hello',
      body_text: 'Hi',
    });

    // then
    await expectStatus(response, 403, OPERATION);
    const payload = await readJson<{ errors: Array<{ code?: string; meta?: { feature_key?: string } }> }>(response, OPERATION);
    expect(payload.errors[0].code, `${OPERATION}: errors[0].code`).toBe('ERR_FORBIDDEN');
    expect(payload.errors[0].meta?.feature_key, `${OPERATION}: errors[0].meta.feature_key`).toBe('mail.send');
  });

  test('401 sin token', async ({ sharedTenant, platformApi }) => {
    // given
    const { tenant } = sharedTenant;

    // when
    const response = await platformApi.call('/api/v1/inbox/messages', {
      method: 'POST',
      tenant,
      data: { account_id: 'x', to: ['a@b.c'], subject: 'x', body_text: 'x' },
    });

    // then
    await expectStatus(response, 401, OPERATION);
  });
});
