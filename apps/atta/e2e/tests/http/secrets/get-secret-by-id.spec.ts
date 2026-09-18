import { expect } from '@playwright/test';
import { test } from '../../support/api.fixture';
import { expectStatus, readJson } from '../../support/http-assertions';
import { newUlid } from '../../support/ulid';

const OPERATION = 'GET /api/v1/secrets/{id}';

test.describe(OPERATION, () => {
  test('devuelve el secreto creado', async ({ sharedTenant, platformApi }) => {
    // given
    const { auth, tenant } = sharedTenant;
    const label = `Get secret ${Date.now()}`;
    const created = await platformApi.call('/api/v1/secrets', {
      method: 'POST',
      auth,
      tenant,
      data: {
        data: {
          type: 'secrets',
          attributes: { purpose: 'invoicing.document_password', label, value: 'pdf-password' },
        },
      },
    });
    await expectStatus(created, 201, 'POST /api/v1/secrets');
    const createdBody = await readJson<{ data: { id: string } }>(created, 'POST /api/v1/secrets');

    // when
    const response = await platformApi.call(`/api/v1/secrets/${createdBody.data.id}`, { auth, tenant });

    // then
    await expectStatus(response, 200, OPERATION);
    const payload = await readJson<{ data: { id: string; attributes: { label: string; kind: string } } }>(response, OPERATION);
    expect(payload.data.id, `${OPERATION}: id`).toBe(createdBody.data.id);
    expect(payload.data.attributes.label, `${OPERATION}: label`).toBe(label);
    expect(payload.data.attributes.kind, `${OPERATION}: kind`).toBe('document_password');
  });

  test('404 si el secreto no existe', async ({ sharedTenant, platformApi }) => {
    // given
    const { auth, tenant } = sharedTenant;

    // when
    const response = await platformApi.call(`/api/v1/secrets/${newUlid()}`, { auth, tenant });

    // then
    await expectStatus(response, 404, OPERATION);
    const payload = await readJson<{ errors: Array<{ code?: string; detail?: string }> }>(response, OPERATION);
    expect(payload.errors[0].code, `${OPERATION}: errors[0].code`).toBe('ERR_NOT_FOUND');
    expect(payload.errors[0].detail, `${OPERATION}: errors[0].detail`).toContain('secret not found');
  });

  test('404 si el secreto es de otro tenant', async ({ sharedTenant, foreignTenant, platformApi }) => {
    // given
    const created = await platformApi.call('/api/v1/secrets', {
      method: 'POST',
      auth: sharedTenant.auth,
      tenant: sharedTenant.tenant,
      data: {
        data: {
          type: 'secrets',
          attributes: { purpose: 'generic.credential', label: 'iso-secret', value: 'super-secret-value' },
        },
      },
    });
    await expectStatus(created, 201, 'POST /api/v1/secrets');
    const createdBody = await readJson<{ data: { id: string } }>(created, 'POST /api/v1/secrets');

    // when
    const response = await platformApi.call(`/api/v1/secrets/${createdBody.data.id}`, {
      auth: foreignTenant.auth,
      tenant: foreignTenant.tenant,
    });

    // then
    await expectStatus(response, 404, OPERATION);
    const payload = await readJson<{ errors: Array<{ detail?: string }> }>(response, OPERATION);
    expect(JSON.stringify(payload), `${OPERATION}: must not leak secret value`).not.toContain('super-secret-value');
  });
});
