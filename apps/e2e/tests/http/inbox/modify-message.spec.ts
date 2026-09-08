import { expect } from '@playwright/test';
import { test } from '../../support/api.fixture';
import { expectStatus, readJson } from '../../support/http-assertions';

const OPERATION = 'POST /api/v1/inbox/messages/{messageID}/{action}';

test.describe(OPERATION, () => {
  test('404 si el mensaje no existe', async ({ sharedTenant, platformApi }) => {
    // given
    const { auth, tenant } = sharedTenant;

    // when
    const response = await platformApi.call('/api/v1/inbox/messages/missing-message-id/archive', {
      method: 'POST',
      auth,
      tenant,
    });

    // then
    await expectStatus(response, 404, OPERATION);
    const payload = await readJson<{ errors: Array<{ code?: string; detail?: string }> }>(response, OPERATION);
    expect(payload.errors[0].code, `${OPERATION}: errors[0].code`).toBe('ERR_NOT_FOUND');
    expect(payload.errors[0].detail, `${OPERATION}: errors[0].detail`).toContain('message not found');
  });

  test('400 si la acción es desconocida', async ({ sharedTenant, platformApi }) => {
    // given
    const { auth, tenant } = sharedTenant;

    // when
    const response = await platformApi.call('/api/v1/inbox/messages/missing-message-id/explode', {
      method: 'POST',
      auth,
      tenant,
    });

    // then
    await expectStatus(response, 400, OPERATION);
    const payload = await readJson<{ errors: Array<{ code?: string }> }>(response, OPERATION);
    expect(payload.errors[0].code, `${OPERATION}: errors[0].code`).toBe('ERR_VALIDATION');
  });
});
