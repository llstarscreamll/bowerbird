import { expect } from '@playwright/test';
import { test } from '../../support/api.fixture';
import { expectStatus, readJson } from '../../support/http-assertions';

const OPERATION = 'GET /api/v1/identity/me';

test.describe(OPERATION, () => {
  test('devuelve el perfil autenticado', async ({ sharedTenant, platformApi }) => {
    // given
    const { auth, user } = sharedTenant;

    // when
    const response = await platformApi.call('/api/v1/identity/me', { auth });

    // then
    await expectStatus(response, 200, OPERATION);
    const payload = await readJson<{
      id: string;
      email: string;
      platform_operator: boolean;
    }>(response, OPERATION);
    expect(payload.id, `${OPERATION}: id`).toBeTruthy();
    expect(payload.email, `${OPERATION}: email`).toBe(user.email);
    expect(payload.platform_operator, `${OPERATION}: platform_operator`).toBe(false);
  });

  test('403 si X-Tenant-ID es de otro tenant', async ({ sharedTenant, foreignTenant, platformApi }) => {
    // given

    // when
    const response = await platformApi.call('/api/v1/identity/me', {
      auth: sharedTenant.auth,
      tenant: foreignTenant.tenant,
    });

    // then
    await expectStatus(response, 403, OPERATION);
    const payload = await readJson<{ errors: Array<{ code?: string }> }>(response, OPERATION);
    expect(payload.errors[0].code, `${OPERATION}: errors[0].code`).toBe('ERR_FORBIDDEN');
  });

  test('401 sin token', async ({ platformApi }) => {
    // given

    // when
    const response = await platformApi.call('/api/v1/identity/me');

    // then
    await expectStatus(response, 401, OPERATION);
  });

  test('401 con token basura', async ({ platformApi }) => {
    // given
    const token = 'eyJhbGciOiJub25lInR5cCI6IkpXVCJ9.eyJzdWIiOiJhZG1pbiIsInBsYXRmb3JtX29wZXJhdG9yIjp0cnVlfQ.';

    // when
    const response = await platformApi.call('/api/v1/identity/me', { token });

    // then
    await expectStatus(response, 401, OPERATION);
  });
});
