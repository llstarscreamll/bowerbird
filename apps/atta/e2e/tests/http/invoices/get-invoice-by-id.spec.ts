import { expect } from '@playwright/test';
import { test } from '../../support/api.fixture';
import { expectStatus, readJson } from '../../support/http-assertions';
import { newUlid } from '../../support/ulid';

const OPERATION = 'GET /api/v1/invoicing/invoices/{id}';

test.describe(OPERATION, () => {
  test('404 si la factura no existe', async ({ sharedTenant, platformApi }) => {
    // given
    const { auth, tenant } = sharedTenant;

    // when
    const response = await platformApi.call(`/api/v1/invoicing/invoices/${newUlid()}`, { auth, tenant });

    // then
    await expectStatus(response, 404, OPERATION);
    const payload = await readJson<{ errors: Array<{ code?: string; detail?: string }> }>(response, OPERATION);
    expect(payload.errors[0].code, `${OPERATION}: errors[0].code`).toBe('ERR_NOT_FOUND');
    expect(payload.errors[0].detail, `${OPERATION}: errors[0].detail`).toContain('invoice not found');
  });
});
