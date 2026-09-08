import { expect } from '@playwright/test';
import { test } from '../../support/api.fixture';
import { expectStatus, readJson } from '../../support/http-assertions';

const OPERATION = 'POST /api/v1/auth/login-local';

test.describe(OPERATION, () => {
  test('401 si las credenciales no existen', async ({ platformApi }) => {
    // given
    const email = `missing.${Date.now()}@example.com`;

    // when
    const response = await platformApi.loginLocal({ email, password: 'P4ssword!e2e' });

    // then
    await expectStatus(response, 401, OPERATION);
    const payload = await readJson<{ errors: Array<{ code?: string; detail?: string }> }>(response, OPERATION);
    expect(payload.errors[0].code, `${OPERATION}: errors[0].code`).toBe('ERR_UNAUTHORIZED');
    expect(payload.errors[0].detail, `${OPERATION}: errors[0].detail`).toContain('invalid credentials');
  });

  test('401 si el password es incorrecto', async ({ sharedTenant, platformApi }) => {
    // given
    const { user } = sharedTenant;

    // when
    const response = await platformApi.loginLocal({ email: user.email, password: `${user.password}-wrong` });

    // then
    await expectStatus(response, 401, OPERATION);
    const payload = await readJson<{ errors: Array<{ code?: string; detail?: string }> }>(response, OPERATION);
    expect(payload.errors[0].code, `${OPERATION}: errors[0].code`).toBe('ERR_UNAUTHORIZED');
  });

  test('401 ante inyección en email', async ({ platformApi }) => {
    // given
    const email = `' OR 1=1 --@example.com`;

    // when
    const response = await platformApi.loginLocal({ email, password: 'x' });

    // then
    await expectStatus(response, 401, OPERATION);
  });
});
