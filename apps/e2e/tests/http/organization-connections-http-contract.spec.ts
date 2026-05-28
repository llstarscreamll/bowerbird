import { expect } from '@playwright/test';
import { test } from '../fixtures/test.fixture';
import { expectJsonApiError } from '../support/jsonapi-assertions';
import { bootstrapAuthenticatedTenantContext } from '../support/test-context.factory';

test.describe('HTTP Contract: organizations and connections', () => {
  test('POST/GET /api/v1/organizations mantiene contrato de creacion y consulta', async ({ newUser, platformApi }) => {
    await platformApi.registerLocalOrFail(newUser);
    const auth = await platformApi.loginLocalOrFail(newUser);

    const randomId = Date.now();
    const createResponse = await platformApi.createOrganization(auth, {
      name: `E2E Org ${randomId}`,
      slug: `e2e-org-${randomId}`,
    });

    expect(createResponse.status()).toBe(201);
    const created = (await createResponse.json()) as {
      id: string;
      name: string;
      slug: string;
      status: string;
      created_at: string;
    };

    expect(created.id).toBeTruthy();
    expect(created.name).toBe(`E2E Org ${randomId}`);
    expect(created.slug).toBe(`e2e-org-${randomId}`);
    expect(created.status).toBeTruthy();
    expect(created.created_at).toBeTruthy();

    const getResponse = await platformApi.getOrganization(auth, created.id);
    expect(getResponse.status()).toBe(200);

    const fetched = (await getResponse.json()) as {
      id: string;
      name: string;
      slug: string;
    };

    expect(fetched.id).toBe(created.id);
    expect(fetched.name).toBe(created.name);
    expect(fetched.slug).toBe(created.slug);
  });

  test('POST /api/v1/organizations con slug repetido retorna error JSON:API de conflicto', async ({ newUser, platformApi }) => {
    await platformApi.registerLocalOrFail(newUser);
    const auth = await platformApi.loginLocalOrFail(newUser);

    const randomId = Date.now();
    const slug = `e2e-org-conflict-${randomId}`;
    await platformApi.createOrganizationOrFail(auth, {
      name: `E2E Conflict A ${randomId}`,
      slug,
    });

    const conflictResponse = await platformApi.createOrganization(auth, {
      name: `E2E Conflict B ${randomId}`,
      slug,
    });

    expect(conflictResponse.status()).toBe(409);
    const payload = await expectJsonApiError(conflictResponse);
    const firstError = payload.errors[0];

    expect(firstError.code).toBe('ERR_CONFLICT');
    expect(firstError.status).toBe('409');
    expect(firstError.detail?.toLowerCase()).toContain('slug');
    expect(firstError.id).toBeTruthy();
  });

  test('GET /api/v1/connections retorna contrato estable en tenant nuevo', async ({ newUser, platformApi }) => {
    const context = await bootstrapAuthenticatedTenantContext(newUser, platformApi);
    const traceId = `e2e-connections-list-${Date.now()}`;

    const response = await platformApi.listConnections(context.auth, context.tenant, traceId);
    expect(response.status()).toBe(200);

    const payload = (await response.json()) as {
      data: unknown[];
    };

    expect(Array.isArray(payload.data)).toBeTruthy();
    expect(payload.data).toHaveLength(0);
  });
});
