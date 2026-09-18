import { expect } from '@playwright/test';
import { test } from '../../support/api.fixture';
import { expectClientError, expectStatus, readJson } from '../../support/http-assertions';

const OPERATION = 'POST /api/v1/files/uploads/presigned';

test.describe(OPERATION, () => {
  test('crea URL de carga', async ({ sharedTenant, platformApi }) => {
    // given
    const { auth, tenant } = sharedTenant;

    // when
    const response = await platformApi.call('/api/v1/files/uploads/presigned', {
      method: 'POST',
      auth,
      tenant,
      data: {
        filename: 'invoice.pdf',
        content_type: 'application/pdf',
        module: 'invoices',
      },
    });

    // then
    await expectStatus(response, 200, OPERATION);
    const payload = await readJson<{
      url: string;
      method: string;
      expires_at: string;
      upload_path: string;
      reference: { key: string };
    }>(response, OPERATION);
    expect(payload.url, `${OPERATION}: url`).toBeTruthy();
    expect(payload.method, `${OPERATION}: method`).toBeTruthy();
    expect(payload.expires_at, `${OPERATION}: expires_at`).toBeTruthy();
    expect(payload.upload_path, `${OPERATION}: upload_path`).toContain(`/tenants/${tenant.tenantSlug}/`);
    expect(payload.reference.key, `${OPERATION}: reference.key`).toBe(payload.upload_path);
  });

  test('rechaza path traversal en filename', async ({ sharedTenant, platformApi }) => {
    // given
    const { auth, tenant } = sharedTenant;

    // when
    const response = await platformApi.call('/api/v1/files/uploads/presigned', {
      method: 'POST',
      auth,
      tenant,
      data: {
        filename: '../../etc/passwd.pdf',
        content_type: 'application/pdf',
        module: 'invoices',
      },
    });

    // then
    if (response.status() === 200) {
      const payload = await readJson<{ upload_path: string }>(response, OPERATION);
      expect(payload.upload_path, `${OPERATION}: path must stay under tenant uploads`).not.toContain('..');
      expect(payload.upload_path, `${OPERATION}: path must stay under tenant`).toContain(`/tenants/${tenant.tenantSlug}/`);
    } else {
      await expectClientError(response, OPERATION);
    }
  });

  test('400 si faltan campos', async ({ sharedTenant, platformApi }) => {
    // given
    const { auth, tenant } = sharedTenant;

    // when
    const response = await platformApi.call('/api/v1/files/uploads/presigned', {
      method: 'POST',
      auth,
      tenant,
      data: { filename: '', content_type: '', module: '' },
    });

    // then
    await expectStatus(response, 400, OPERATION);
  });

  test('401 sin token', async ({ sharedTenant, platformApi }) => {
    // given
    const { tenant } = sharedTenant;

    // when
    const response = await platformApi.call('/api/v1/files/uploads/presigned', {
      method: 'POST',
      tenant,
      data: { filename: 'x.pdf', content_type: 'application/pdf', module: 'invoices' },
    });

    // then
    await expectStatus(response, 401, OPERATION);
  });
});
