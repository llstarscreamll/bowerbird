import { expect } from '@playwright/test';
import { test } from '../../support/api.fixture';
import { expectStatus, readJson } from '../../support/http-assertions';

type PartyAttrs = {
  name: string;
  tax_id: string;
  scheme_id: string;
  emails: Array<{ id: string; value: string; kind: string; source: string }>;
  phones: Array<{ id: string; value: string; source: string }>;
  addresses: Array<{
    id: string;
    line: string;
    city: string;
    department: string;
    postal_zone: string;
    country_code: string;
    kind: string;
    source: string;
  }>;
};

type PartyDoc = { data: { id: string; attributes: PartyAttrs } };

test.describe('Party profile channels', () => {
  test('create persiste scheme_id y colecciones vacías', async ({ sharedTenant, platformApi }) => {
    const { auth, tenant } = sharedTenant;
    const stamp = `${Date.now()}`;
    const created = await platformApi.call('/api/v1/parties', {
      method: 'POST',
      auth,
      tenant,
      data: {
        data: {
          attributes: {
            name: `Profile ${stamp}`,
            tax_id: `906${stamp}`,
            scheme_id: '31',
            roles: ['supplier'],
          },
        },
      },
    });
    await expectStatus(created, 201, 'POST /api/v1/parties');
    const payload = await readJson<PartyDoc>(created, 'POST /api/v1/parties');
    expect(payload.data.attributes.scheme_id, 'scheme_id').toBe('31');
    expect(payload.data.attributes.emails, 'emails').toEqual([]);
    expect(payload.data.attributes.phones, 'phones').toEqual([]);
    expect(payload.data.attributes.addresses, 'addresses').toEqual([]);
  });

  test('CRUD de email, teléfono y dirección con source manual', async ({ sharedTenant, platformApi }) => {
    const { auth, tenant } = sharedTenant;
    const stamp = `${Date.now()}`;
    const created = await platformApi.call('/api/v1/parties', {
      method: 'POST',
      auth,
      tenant,
      data: {
        data: {
          attributes: {
            name: `Profile ${stamp}`,
            tax_id: `907${stamp}`,
            scheme_id: '31',
            roles: ['supplier'],
          },
        },
      },
    });
    await expectStatus(created, 201, 'POST /api/v1/parties');
    const createdBody = await readJson<PartyDoc>(created, 'POST /api/v1/parties');
    const id = createdBody.data.id;

    const emailRes = await platformApi.call(`/api/v1/parties/${id}/emails`, {
      method: 'POST',
      auth,
      tenant,
      data: { data: { attributes: { value: `ops-${stamp}@ishop.example` } } },
    });
    await expectStatus(emailRes, 201, 'POST /api/v1/parties/{id}/emails');
    const afterEmail = await readJson<PartyDoc>(emailRes, 'POST /api/v1/parties/{id}/emails');
    expect(afterEmail.data.attributes.emails).toHaveLength(1);
    expect(afterEmail.data.attributes.emails[0].value).toBe(`ops-${stamp}@ishop.example`);
    expect(afterEmail.data.attributes.emails[0].source).toBe('manual');
    expect(afterEmail.data.attributes.emails[0].kind).toBe('general');
    const emailId = afterEmail.data.attributes.emails[0].id;

    const dup = await platformApi.call(`/api/v1/parties/${id}/emails`, {
      method: 'POST',
      auth,
      tenant,
      data: { data: { attributes: { value: `OPS-${stamp}@ishop.example` } } },
    });
    await expectStatus(dup, 409, 'POST duplicate email');

    const phoneRes = await platformApi.call(`/api/v1/parties/${id}/phones`, {
      method: 'POST',
      auth,
      tenant,
      data: { data: { attributes: { value: '(1) 3289133' } } },
    });
    await expectStatus(phoneRes, 201, 'POST /api/v1/parties/{id}/phones');
    const afterPhone = await readJson<PartyDoc>(phoneRes, 'POST /api/v1/parties/{id}/phones');
    expect(afterPhone.data.attributes.phones).toHaveLength(1);
    expect(afterPhone.data.attributes.phones[0].value).toBe('(1) 3289133');
    expect(afterPhone.data.attributes.phones[0].source).toBe('manual');

    const addrRes = await platformApi.call(`/api/v1/parties/${id}/addresses`, {
      method: 'POST',
      auth,
      tenant,
      data: {
        data: {
          attributes: {
            line: 'AV UNIVERSITARIA 50 21',
            city: 'TUNJA',
            department: 'BOYACÁ',
            postal_zone: '150001',
            country_code: 'CO',
            kind: 'physical',
          },
        },
      },
    });
    await expectStatus(addrRes, 201, 'POST /api/v1/parties/{id}/addresses');
    const afterAddr = await readJson<PartyDoc>(addrRes, 'POST /api/v1/parties/{id}/addresses');
    expect(afterAddr.data.attributes.addresses).toHaveLength(1);
    expect(afterAddr.data.attributes.addresses[0].city).toBe('TUNJA');
    expect(afterAddr.data.attributes.addresses[0].source).toBe('manual');
    expect(afterAddr.data.attributes.addresses[0].kind).toBe('physical');

    const getRes = await platformApi.call(`/api/v1/parties/${id}`, { auth, tenant });
    await expectStatus(getRes, 200, 'GET /api/v1/parties/{id}');
    const got = await readJson<PartyDoc>(getRes, 'GET /api/v1/parties/{id}');
    expect(got.data.attributes.scheme_id).toBe('31');
    expect(got.data.attributes.emails[0].source).toBe('manual');
    expect(got.data.attributes.phones[0].source).toBe('manual');
    expect(got.data.attributes.addresses[0].source).toBe('manual');

    const del = await platformApi.call(`/api/v1/parties/${id}/emails/${emailId}`, {
      method: 'DELETE',
      auth,
      tenant,
    });
    await expectStatus(del, 200, 'DELETE /api/v1/parties/{id}/emails/{id}');
    const afterDel = await readJson<PartyDoc>(del, 'DELETE /api/v1/parties/{id}/emails/{id}');
    expect(afterDel.data.attributes.emails).toEqual([]);
    expect(afterDel.data.attributes.phones).toHaveLength(1);
  });
});
