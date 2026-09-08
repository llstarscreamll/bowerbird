import { expect } from '@playwright/test';
import { test } from '../../support/api.fixture';
import { expectStatus, readJson } from '../../support/http-assertions';

const OPERATION = 'GET /api/v1/identity/tenants';

test.describe(OPERATION, () => {
  test('incluye el tenant del usuario', async ({ sharedTenant, platformApi }) => {
    // given
    const { auth, tenant } = sharedTenant;

    // when
    const response = await platformApi.call('/api/v1/identity/tenants', { auth });

    // then
    await expectStatus(response, 200, OPERATION);
    const payload = await readJson<Array<{ tenant_id: string; name: string; role: string }>>(response, OPERATION);
    const membership = payload.find((item) => item.tenant_id === tenant.tenantId);
    expect(membership, `${OPERATION}: membership for ${tenant.tenantId}`).toBeTruthy();
    expect(membership?.role, `${OPERATION}: role`).toBe('OWNER');
  });

  test('no incluye tenants ajenos', async ({ sharedTenant, foreignTenant, platformApi }) => {
    // given
    const { auth } = sharedTenant;

    // when
    const response = await platformApi.call('/api/v1/identity/tenants', { auth });

    // then
    await expectStatus(response, 200, OPERATION);
    const payload = await readJson<Array<{ tenant_id: string }>>(response, OPERATION);
    expect(
      payload.map((item) => item.tenant_id),
      `${OPERATION}: must not leak foreign tenant`,
    ).not.toContain(foreignTenant.tenant.tenantId);
  });

  test('401 sin token', async ({ platformApi }) => {
    // given

    // when
    const response = await platformApi.call('/api/v1/identity/tenants');

    // then
    await expectStatus(response, 401, OPERATION);
  });
});
