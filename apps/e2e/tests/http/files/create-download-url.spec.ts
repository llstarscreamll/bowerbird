import { expect } from '@playwright/test';
import { test } from '../../support/api.fixture';
import { expectClientError, expectStatus, readJson } from '../../support/http-assertions';

const OPERATION = 'POST /api/v1/files/downloads/presigned';

test.describe(OPERATION, () => {
  test('400 si falta la key', async ({ sharedTenant, platformApi }) => {
    // given
    const { auth, tenant } = sharedTenant;

    // when
    const response = await platformApi.call('/api/v1/files/downloads/presigned', {
      method: 'POST',
      auth,
      tenant,
      data: { key: '' },
    });

    // then
    await expectStatus(response, 400, OPERATION);
  });

  test('404 si el archivo no existe', async ({ sharedTenant, platformApi }) => {
    // given
    const { auth, tenant } = sharedTenant;

    // when
    const response = await platformApi.call('/api/v1/files/downloads/presigned', {
      method: 'POST',
      auth,
      tenant,
      data: { key: `1-day/${tenant.tenantSlug}/e2e-missing.pdf` },
    });

    // then
    await expectStatus(response, 404, OPERATION);
    const payload = await readJson<{ errors: Array<{ code?: string; detail?: string }> }>(response, OPERATION);
    expect(payload.errors[0].code, `${OPERATION}: errors[0].code`).toBe('ERR_NOT_FOUND');
    expect(payload.errors[0].detail, `${OPERATION}: errors[0].detail`).toContain('file not found');
  });

  test('rechaza path traversal', async ({ sharedTenant, platformApi }) => {
    // given
    const { auth, tenant } = sharedTenant;

    // when
    const response = await platformApi.call('/api/v1/files/downloads/presigned', {
      method: 'POST',
      auth,
      tenant,
      data: { key: `1-day/${tenant.tenantSlug}/../../etc/passwd` },
    });

    // then
    await expectClientError(response, OPERATION);
    expect(response.status(), `${OPERATION}: traversal must not succeed`).not.toBe(200);
  });

  test('rechaza key de otro tenant', async ({ sharedTenant, foreignTenant, platformApi }) => {
    // given

    // when
    const response = await platformApi.call('/api/v1/files/downloads/presigned', {
      method: 'POST',
      auth: sharedTenant.auth,
      tenant: sharedTenant.tenant,
      data: { key: `1-day/${foreignTenant.tenant.tenantSlug}/invoice.pdf` },
    });

    // then
    await expectClientError(response, OPERATION);
    expect(response.status(), `${OPERATION}: foreign key must not succeed`).not.toBe(200);
  });
});
