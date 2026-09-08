import { test } from '../../support/api.fixture';
import { expectStatus, readJson } from '../../support/http-assertions';

const OPERATION = 'DELETE /api/v1/secrets/{id}';

test.describe(OPERATION, () => {
  test('elimina el secreto', async ({ sharedTenant, platformApi }) => {
    // given
    const { auth, tenant } = sharedTenant;
    const created = await platformApi.call('/api/v1/secrets', {
      method: 'POST',
      auth,
      tenant,
      data: {
        data: {
          type: 'secrets',
          attributes: { purpose: 'generic.credential', label: `Delete ${Date.now()}`, value: 'to-delete' },
        },
      },
    });
    await expectStatus(created, 201, 'POST /api/v1/secrets');
    const createdBody = await readJson<{ data: { id: string } }>(created, 'POST /api/v1/secrets');

    // when
    const response = await platformApi.call(`/api/v1/secrets/${createdBody.data.id}`, { method: 'DELETE', auth, tenant });

    // then
    await expectStatus(response, 204, OPERATION);
    const lookup = await platformApi.call(`/api/v1/secrets/${createdBody.data.id}`, { auth, tenant });
    await expectStatus(lookup, 404, 'GET /api/v1/secrets/{id}');
  });

  test('no borra un secreto de otro tenant', async ({ sharedTenant, foreignTenant, platformApi }) => {
    // given
    const created = await platformApi.call('/api/v1/secrets', {
      method: 'POST',
      auth: sharedTenant.auth,
      tenant: sharedTenant.tenant,
      data: {
        data: {
          type: 'secrets',
          attributes: { purpose: 'generic.credential', label: 'do-not-delete', value: 'keep-me' },
        },
      },
    });
    await expectStatus(created, 201, 'POST /api/v1/secrets');
    const createdBody = await readJson<{ data: { id: string } }>(created, 'POST /api/v1/secrets');

    // when
    const response = await platformApi.call(`/api/v1/secrets/${createdBody.data.id}`, {
      method: 'DELETE',
      auth: foreignTenant.auth,
      tenant: foreignTenant.tenant,
    });

    // then
    await expectStatus(response, 404, OPERATION);
    const stillThere = await platformApi.call(`/api/v1/secrets/${createdBody.data.id}`, {
      auth: sharedTenant.auth,
      tenant: sharedTenant.tenant,
    });
    await expectStatus(stillThere, 200, 'GET /api/v1/secrets/{id}');
  });
});
