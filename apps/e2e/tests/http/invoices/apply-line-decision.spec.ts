import { expect } from '@playwright/test';
import { test } from '../../support/api.fixture';
import { expectStatus, readJson } from '../../support/http-assertions';
import { newUlid } from '../../support/ulid';

const OPERATION = 'POST /api/v1/invoicing/invoices/{invoiceId}/lines/{lineId}/decisions';

test.describe(OPERATION, () => {
  test('404 si la línea no existe', async ({ sharedTenant, platformApi }) => {
    // given
    const { auth, tenant } = sharedTenant;
    const invoiceId = newUlid();
    const lineId = newUlid();

    // when
    const response = await platformApi.call(`/api/v1/invoicing/invoices/${invoiceId}/lines/${lineId}/decisions`, {
      method: 'POST',
      auth,
      tenant,
      data: {
        data: {
          type: 'invoice_line_decisions',
          attributes: { item_id: newUlid(), action: 'link' },
        },
      },
    });

    // then
    await expectStatus(response, 404, OPERATION);
    const payload = await readJson<{ errors: Array<{ code?: string; detail?: string }> }>(response, OPERATION);
    expect(payload.errors[0].code, `${OPERATION}: errors[0].code`).toBe('ERR_NOT_FOUND');
    expect(payload.errors[0].detail, `${OPERATION}: errors[0].detail`).toContain('invoice line not found');
  });
});
