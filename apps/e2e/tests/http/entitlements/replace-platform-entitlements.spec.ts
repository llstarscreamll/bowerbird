import { expect } from '@playwright/test';
import { test } from '../../support/api.fixture';
import { expectStatus, readJson } from '../../support/http-assertions';

const OPERATION = 'PUT /api/v1/platform/tenants/{id}/entitlements';

test.describe(OPERATION, () => {
  test('403 si intenta activar mail.send sin ser operator', async ({ sharedTenant, platformApi }) => {
    // given
    const { auth, tenant } = sharedTenant;

    // when
    const response = await platformApi.call(`/api/v1/platform/tenants/${tenant.tenantId}/entitlements`, {
      method: 'PUT',
      auth,
      data: {
        product: 'mail',
        feature: 'mail.send',
        enabled: true,
      },
    });

    // then
    await expectStatus(response, 403, OPERATION);
    const payload = await readJson<{ errors: Array<{ code?: string; detail?: string }> }>(response, OPERATION);
    expect(payload.errors[0].code, `${OPERATION}: errors[0].code`).toBe('ERR_FORBIDDEN');
  });
});
