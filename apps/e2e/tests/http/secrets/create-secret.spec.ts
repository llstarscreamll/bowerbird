import { expect } from '@playwright/test';
import { test } from '../../support/api.fixture';
import { expectStatus, readJson } from '../../support/http-assertions';

const OPERATION = 'POST /api/v1/secrets';

test.describe(OPERATION, () => {
  test('crea el secreto', async ({ sharedTenant, platformApi }) => {
    // given
    const { auth, tenant } = sharedTenant;
    const label = `E2E secret ${Date.now()}`;

    // when
    const response = await platformApi.call('/api/v1/secrets', {
      method: 'POST',
      auth,
      tenant,
      data: {
        data: {
          type: 'secrets',
          attributes: {
            purpose: 'generic.credential',
            label,
            value: 's3cret-value',
          },
        },
      },
    });

    // then
    await expectStatus(response, 201, OPERATION);
    const payload = await readJson<{
      data: { type: string; id: string; attributes: { purpose: string; label: string; has_value: boolean; version: number } };
    }>(response, OPERATION);
    expect(payload.data.type, `${OPERATION}: data.type`).toBe('secrets');
    expect(payload.data.id, `${OPERATION}: id`).toBeTruthy();
    expect(payload.data.attributes.purpose, `${OPERATION}: purpose`).toBe('generic.credential');
    expect(payload.data.attributes.label, `${OPERATION}: label`).toBe(label);
    expect(payload.data.attributes.has_value, `${OPERATION}: has_value`).toBe(true);
    expect(payload.data.attributes.version, `${OPERATION}: version`).toBe(1);
  });

  test('400 si el purpose es desconocido', async ({ sharedTenant, platformApi }) => {
    // given
    const { auth, tenant } = sharedTenant;

    // when
    const response = await platformApi.call('/api/v1/secrets', {
      method: 'POST',
      auth,
      tenant,
      data: {
        data: {
          type: 'secrets',
          attributes: { purpose: 'root.access', label: 'backdoor', value: 'x' },
        },
      },
    });

    // then
    await expectStatus(response, 400, OPERATION);
  });

  test('400 si el valor está vacío', async ({ sharedTenant, platformApi }) => {
    // given
    const { auth, tenant } = sharedTenant;

    // when
    const response = await platformApi.call('/api/v1/secrets', {
      method: 'POST',
      auth,
      tenant,
      data: {
        data: {
          type: 'secrets',
          attributes: { purpose: 'generic.credential', label: 'empty', value: '   ' },
        },
      },
    });

    // then
    await expectStatus(response, 400, OPERATION);
  });

  test('401 sin token', async ({ sharedTenant, platformApi }) => {
    // given
    const { tenant } = sharedTenant;

    // when
    const response = await platformApi.call('/api/v1/secrets', {
      method: 'POST',
      tenant,
      data: {
        data: {
          type: 'secrets',
          attributes: { purpose: 'generic.credential', label: 'unauth', value: 'x' },
        },
      },
    });

    // then
    await expectStatus(response, 401, OPERATION);
  });
});
