import { expect } from '@playwright/test';
import { test } from '../../support/api.fixture';
import { expectStatus, readJson } from '../../support/http-assertions';

const OPERATION = 'GET /api/v1/platform/tenants';

test.describe(OPERATION, () => {
  test('403 si el usuario no es operator', async ({ sharedTenant, platformApi }) => {
    // given
    const { auth } = sharedTenant;

    // when
    const response = await platformApi.call('/api/v1/platform/tenants', { auth });

    // then
    await expectStatus(response, 403, OPERATION);
    const payload = await readJson<{ errors: Array<{ code?: string; detail?: string }> }>(response, OPERATION);
    expect(payload.errors[0].code, `${OPERATION}: errors[0].code`).toBe('ERR_FORBIDDEN');
    expect(payload.errors[0].detail, `${OPERATION}: errors[0].detail`).toContain('operator access required');
  });
});
