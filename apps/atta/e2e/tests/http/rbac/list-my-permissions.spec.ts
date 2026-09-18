import { expect } from '@playwright/test';
import { test } from '../../support/api.fixture';
import { expectStatus, readJson } from '../../support/http-assertions';

const OPERATION = 'GET /api/v1/rbac/me/permissions';

test.describe(OPERATION, () => {
  test('incluye permisos de secretos para el owner', async ({ sharedTenant, platformApi }) => {
    // given
    const { auth, tenant } = sharedTenant;

    // when
    const response = await platformApi.call('/api/v1/rbac/me/permissions', { auth, tenant });

    // then
    await expectStatus(response, 200, OPERATION);
    const payload = await readJson<{ permissions: string[] }>(response, OPERATION);
    expect(payload.permissions, `${OPERATION}: permissions`).toEqual(expect.arrayContaining(['secrets:read', 'secrets:write', 'secrets:delete']));
  });

  test('401 sin token', async ({ sharedTenant, platformApi }) => {
    // given
    const { tenant } = sharedTenant;

    // when
    const response = await platformApi.call('/api/v1/rbac/me/permissions', { tenant });

    // then
    await expectStatus(response, 401, OPERATION);
  });
});
