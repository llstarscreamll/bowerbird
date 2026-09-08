import { expect } from '@playwright/test';
import { test } from '../../support/api.fixture';
import { expectClientError, expectStatus, readJson } from '../../support/http-assertions';
import { newUlid } from '../../support/ulid';

const OPERATION = 'POST /api/v1/invoicing/extractions';

test.describe(OPERATION, () => {
  test('400 si no hay archivos', async ({ sharedTenant, platformApi }) => {
    // given
    const { auth, tenant } = sharedTenant;

    // when
    const response = await platformApi.call('/api/v1/invoicing/extractions', {
      method: 'POST',
      auth,
      tenant,
      data: {
        data: {
          type: 'queue-invoice-extraction',
          id: newUlid(),
          attributes: { files: [] },
        },
      },
    });

    // then
    await expectStatus(response, 400, OPERATION);
    const payload = await readJson<{ errors: Array<{ code?: string; detail?: string }> }>(response, OPERATION);
    expect(payload.errors[0].code, `${OPERATION}: errors[0].code`).toBe('ERR_VALIDATION');
    expect(payload.errors[0].detail, `${OPERATION}: errors[0].detail`).toContain('invalid request body');
  });

  test('400 si el mime_type no está permitido', async ({ sharedTenant, platformApi }) => {
    // given
    const { auth, tenant } = sharedTenant;

    // when
    const response = await platformApi.call('/api/v1/invoicing/extractions', {
      method: 'POST',
      auth,
      tenant,
      data: {
        data: {
          type: 'queue-invoice-extraction',
          id: newUlid(),
          attributes: {
            files: [{ name: 'invoice.txt', path: 'uploads/invoicing/u1/invoice.txt', mime_type: 'text/plain' }],
          },
        },
      },
    });

    // then
    await expectStatus(response, 400, OPERATION);
  });

  test('400 si hay más de 50 archivos', async ({ sharedTenant, platformApi }) => {
    // given
    const { auth, tenant } = sharedTenant;
    const files = Array.from({ length: 51 }, () => ({
      name: 'invoice.pdf',
      path: 'uploads/invoicing/u1/invoice.pdf',
      mime_type: 'application/pdf',
    }));

    // when
    const response = await platformApi.call('/api/v1/invoicing/extractions', {
      method: 'POST',
      auth,
      tenant,
      data: {
        data: {
          type: 'queue-invoice-extraction',
          id: newUlid(),
          attributes: { files },
        },
      },
    });

    // then
    await expectStatus(response, 400, OPERATION);
  });

  test('400 si mime_type y extensión no coinciden', async ({ sharedTenant, platformApi }) => {
    // given
    const { auth, tenant } = sharedTenant;

    // when
    const response = await platformApi.call('/api/v1/invoicing/extractions', {
      method: 'POST',
      auth,
      tenant,
      data: {
        data: {
          type: 'queue-invoice-extraction',
          id: newUlid(),
          attributes: {
            files: [{ name: 'invoice.xml', path: 'uploads/invoicing/u1/invoice.xml', mime_type: 'application/pdf' }],
          },
        },
      },
    });

    // then
    await expectStatus(response, 400, OPERATION);
  });

  test('rechaza path traversal en files[].path', async ({ sharedTenant, platformApi }) => {
    // given
    const { auth, tenant } = sharedTenant;

    // when
    const response = await platformApi.call('/api/v1/invoicing/extractions', {
      method: 'POST',
      auth,
      tenant,
      data: {
        data: {
          type: 'queue-invoice-extraction',
          id: newUlid(),
          attributes: {
            files: [
              {
                name: 'invoice.pdf',
                path: `../../etc/passwd.pdf`,
                mime_type: 'application/pdf',
              },
            ],
          },
        },
      },
    });

    // then
    await expectClientError(response, OPERATION);
    expect(response.status(), `${OPERATION}: traversal must not be accepted`).not.toBe(202);
  });
});
