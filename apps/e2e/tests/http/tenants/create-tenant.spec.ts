import { expect } from '@playwright/test';
import { test } from '../../support/api.fixture';
import { expectClientError, expectStatus, readJson } from '../../support/http-assertions';

const OPERATION = 'POST /api/v1/tenants';

test.describe(OPERATION, () => {
  test('crea el tenant', async ({ sharedTenant, platformApi }) => {
    // given
    const { auth } = sharedTenant;
    const randomId = Date.now();
    const name = `E2E Tenant ${randomId}`;
    const slug = `e2e-tenant-${randomId}`;

    // when
    const response = await platformApi.createTenant(auth, { name, slug });

    // then
    await expectStatus(response, 201, OPERATION);
    const created = await readJson<{
      id: string;
      name: string;
      slug: string;
      status: string;
      created_at: string;
    }>(response, OPERATION);
    expect(created.id, `${OPERATION}: id`).toBeTruthy();
    expect(created.name, `${OPERATION}: name`).toBe(name);
    expect(created.slug, `${OPERATION}: slug`).toBe(slug);
    expect(created.status, `${OPERATION}: status`).toBeTruthy();
    expect(created.created_at, `${OPERATION}: created_at`).toBeTruthy();
  });

  test('409 JSON:API si el slug ya existe', async ({ sharedTenant, platformApi }) => {
    // given
    const { auth } = sharedTenant;
    const randomId = Date.now();
    const slug = `e2e-org-conflict-${randomId}`;
    await platformApi.createTenantOrFail(auth, {
      name: `E2E Conflict A ${randomId}`,
      slug,
    });

    // when
    const response = await platformApi.createTenant(auth, {
      name: `E2E Conflict B ${randomId}`,
      slug,
    });

    // then
    await expectStatus(response, 409, OPERATION);
    const payload = await readJson<{ errors: Array<{ code?: string; detail?: string }> }>(response, OPERATION);
    expect(payload.errors[0].code, `${OPERATION}: errors[0].code`).toBe('ERR_CONFLICT');
    expect(payload.errors[0].detail?.toLowerCase(), `${OPERATION}: errors[0].detail should mention slug`).toContain('slug');
  });

  test('401 sin token', async ({ platformApi }) => {
    // given

    // when
    const response = await platformApi.call('/api/v1/tenants', {
      method: 'POST',
      data: { name: 'Nope', slug: `unauth-${Date.now()}` },
    });

    // then
    await expectStatus(response, 401, OPERATION);
  });

  test('rechaza slug malicioso', async ({ sharedTenant, platformApi }) => {
    // given
    const { auth } = sharedTenant;

    // when
    const response = await platformApi.createTenant(auth, {
      name: 'Evil',
      slug: '../admin;drop-table',
    });

    // then
    await expectClientError(response, OPERATION);
  });
});
