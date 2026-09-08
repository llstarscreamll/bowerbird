import { expect } from '@playwright/test';
import { test } from '../../support/api.fixture';
import { readJson } from '../../support/http-assertions';
import { newUlid } from '../../support/ulid';

test.describe('cross-tenant X-Tenant-ID', () => {
  test('token de A + tenant de B no autoriza', async ({ sharedTenant, foreignTenant, platformApi }) => {
    // given
    const stolen = { auth: sharedTenant.auth, tenant: foreignTenant.tenant };
    const itemId = newUlid();
    await platformApi.call('/api/v1/catalog/items', {
      method: 'POST',
      auth: foreignTenant.auth,
      tenant: foreignTenant.tenant,
      data: {
        data: {
          type: 'catalog_items',
          id: itemId,
          attributes: { name: 'Victim item', kind: 'goods', internal_sku: `SKU-VICTIM-${Date.now()}` },
        },
      },
    });

    // when
    const catalog = await platformApi.call(`/api/v1/catalog/items/${itemId}`, stolen);
    const connections = await platformApi.call('/api/v1/connections', stolen);
    const inbox = await platformApi.call('/api/v1/inbox/messages', stolen);
    const entitlements = await platformApi.call('/api/v1/entitlements', stolen);
    const rbac = await platformApi.call('/api/v1/rbac/me/permissions', stolen);
    const injected = await platformApi.call('/api/v1/connections', {
      auth: sharedTenant.auth,
      tenantIdHeader: `' OR 1=1 --`,
    });

    // then — membership mismatch must be 403, never 200 and never 500
    expect.soft(catalog.status(), 'GET catalog item with stolen tenant').toBe(403);
    if (catalog.status() === 403) {
      const body = await readJson<{ errors: Array<{ code?: string }> }>(catalog, 'GET catalog item with stolen tenant');
      expect.soft(body.errors[0].code, 'stolen tenant must be ERR_FORBIDDEN').toBe('ERR_FORBIDDEN');
    }
    expect.soft(connections.status(), 'GET connections with stolen tenant').toBe(403);
    expect.soft(inbox.status(), 'GET inbox with stolen tenant').toBe(403);
    expect.soft(entitlements.status(), 'GET entitlements with stolen tenant').toBe(403);
    if (entitlements.status() === 200) {
      const body = await readJson<{ features: string[] }>(entitlements, 'GET /api/v1/entitlements');
      expect.soft(body.features, 'entitlements must not evaluate a tenant you do not belong to').toEqual([]);
    }
    expect.soft(rbac.status(), 'GET rbac with stolen tenant').toBe(403);
    expect.soft(injected.status(), 'injected X-Tenant-ID').toBeGreaterThanOrEqual(400);
    expect.soft(injected.status(), 'injected X-Tenant-ID must not 5xx').toBeLessThan(500);
  });
});
