import { expect } from '@playwright/test';
import { test } from '../../support/api.fixture';
import { expectStatus } from '../../support/http-assertions';
import { expectJsonApiError } from '../../support/jsonapi-assertions';

const OPERATION = 'GET /api/v1/inbox/messages/{id}';

test.describe(OPERATION, () => {
  test('404 JSON:API si el mensaje no existe', async ({ sharedTenant, platformApi }) => {
    // given
    const context = sharedTenant;
    const traceId = `e2e-message-not-found-${Date.now()}`;

    // when
    const response = await platformApi.getInboxMessage(context.auth, context.tenant, 'missing-message-id', traceId);

    // then
    await expectStatus(response, 404, OPERATION);
    const payload = await expectJsonApiError(response, OPERATION);
    const error = payload.errors[0];
    expect(error.id, `${OPERATION}: errors[0].id should echo sentry-trace`).toBe(traceId);
    expect(error.status, `${OPERATION}: errors[0].status`).toBe('404');
    expect(error.code, `${OPERATION}: errors[0].code`).toBe('ERR_NOT_FOUND');
    expect(error.detail, `${OPERATION}: errors[0].detail`).toContain('message not found');
  });
});
