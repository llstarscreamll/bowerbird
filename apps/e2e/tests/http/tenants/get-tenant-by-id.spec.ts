import { expect } from '@playwright/test';
import { test } from '../../support/api.fixture';
import { expectStatus, readJson } from '../../support/http-assertions';

const OPERATION = 'GET /api/v1/tenants/{id}';

test.describe(OPERATION, () => {
  test('devuelve el tenant creado', async ({ sharedTenant, platformApi }) => {
    // given
    const { auth, tenant } = sharedTenant;

    // when
    const response = await platformApi.getTenant(auth, tenant.tenantId);

    // then
    await expectStatus(response, 200, OPERATION);
    const fetched = await readJson<{ id: string; slug: string }>(response, OPERATION);
    expect(fetched.id, `${OPERATION}: id`).toBe(tenant.tenantId);
    expect(fetched.slug, `${OPERATION}: slug`).toBe(tenant.tenantSlug);
  });

  test('404 si el tenant no existe', async ({ sharedTenant, platformApi }) => {
    // given
    const { auth } = sharedTenant;

    // when
    const response = await platformApi.getTenant(auth, '01ARZ3NDEKTSV4RRFFQ69G5FAV');

    // then
    await expectStatus(response, 404, OPERATION);
    const payload = await readJson<{ errors: Array<{ code?: string; detail?: string }> }>(response, OPERATION);
    expect(payload.errors[0].code, `${OPERATION}: errors[0].code`).toBe('ERR_NOT_FOUND');
    expect(payload.errors[0].detail, `${OPERATION}: errors[0].detail`).toContain('tenant not found');
  });

  test('404 si el tenant es de otro usuario', async ({ sharedTenant, foreignTenant, platformApi }) => {
    // given
    const { auth } = sharedTenant;

    // when
    const response = await platformApi.getTenant(auth, foreignTenant.tenant.tenantId);

    // then
    await expectStatus(response, 404, OPERATION);
    const payload = await readJson<{ errors: Array<{ code?: string; detail?: string }> }>(response, OPERATION);
    expect(payload.errors[0].code, `${OPERATION}: errors[0].code`).toBe('ERR_NOT_FOUND');
    expect(JSON.stringify(payload), `${OPERATION}: must not leak foreign slug`).not.toContain(foreignTenant.tenant.tenantSlug);
  });

  test('401 sin token', async ({ sharedTenant, platformApi }) => {
    // given
    const { tenant } = sharedTenant;

    // when
    const response = await platformApi.call(`/api/v1/tenants/${tenant.tenantId}`);

    // then
    await expectStatus(response, 401, OPERATION);
  });
});
