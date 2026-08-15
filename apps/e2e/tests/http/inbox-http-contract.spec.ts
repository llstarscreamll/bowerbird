import { expect } from '@playwright/test';
import { test } from '../fixtures/test.fixture';
import { expectJsonApiError } from '../support/jsonapi-assertions';
import { bootstrapAuthenticatedTenantContext } from '../support/test-context.factory';

test.describe('HTTP Contract: inbox sync endpoints', () => {
  test('lista de mensajes y estado de sync responde 200 con estructuras vacias en tenant nuevo', async ({ newUser, platformApi }) => {
    const context = await bootstrapAuthenticatedTenantContext(newUser, platformApi);

    await test.step('GET /api/v1/inbox/messages retorna lista vacia', async () => {
      const response = await platformApi.listInboxMessages(context.auth, context.tenant);

      expect(response.status()).toBe(200);
      const payload = (await response.json()) as { data?: unknown[]; total?: number; limit?: number; offset?: number };
      expect(Array.isArray(payload.data)).toBeTruthy();
      expect(payload.data).toHaveLength(0);
      expect(payload.total).toBe(0);
    });

    await test.step('GET /api/v1/inbox/sync-status retorna lista vacia', async () => {
      const response = await platformApi.listInboxSyncStatus(context.auth, context.tenant);

      expect(response.status()).toBe(200);
      const payload = (await response.json()) as unknown[];
      expect(Array.isArray(payload)).toBeTruthy();
      expect(payload).toHaveLength(0);
    });
  });

  test('POST /api/v1/inbox/sync retorna 202 cuando no hay cuentas activas', async ({ newUser, platformApi }) => {
    const context = await bootstrapAuthenticatedTenantContext(newUser, platformApi);

    const response = await platformApi.triggerInboxSync(context.auth, context.tenant);
    expect(response.status()).toBe(202);

    const payload = (await response.json()) as { message?: string };
    expect(payload.message).toBe('Sync triggered');
  });

  test('POST /api/v1/inbox/sync con account_id desconocido sigue encolando sync del tenant', async ({ newUser, platformApi }) => {
    const context = await bootstrapAuthenticatedTenantContext(newUser, platformApi);

    const response = await platformApi.triggerInboxSync(context.auth, context.tenant, 'missing-account-id');
    expect(response.status()).toBe(202);

    const payload = (await response.json()) as { message?: string };
    expect(payload.message).toBe('Sync triggered');
  });

  test('POST /api/v1/inbox/messages retorna 403 sin mail.send', async ({ newUser, platformApi }) => {
    const context = await bootstrapAuthenticatedTenantContext(newUser, platformApi);
    const traceId = `e2e-send-forbidden-${Date.now()}`;

    const response = await platformApi.sendInboxMessage(
      context.auth,
      context.tenant,
      {
        account_id: 'missing-account',
        to: ['someone@example.com'],
        subject: 'Hello',
        body_text: 'Hi',
      },
      traceId,
    );
    expect(response.status()).toBe(403);

    const payload = await expectJsonApiError(response);
    const firstError = payload.errors[0];
    expect(firstError.code).toBe('ERR_FORBIDDEN');
    expect(firstError.meta?.feature_key).toBe('mail.send');
  });

  test('GET /api/v1/inbox/messages/{id} inexistente retorna JSON:API not found', async ({ newUser, platformApi }) => {
    const context = await bootstrapAuthenticatedTenantContext(newUser, platformApi);
    const traceId = `e2e-message-not-found-${Date.now()}`;

    const response = await platformApi.getInboxMessage(context.auth, context.tenant, 'missing-message-id', traceId);
    expect(response.status()).toBe(404);

    const payload = await expectJsonApiError(response);
    const firstError = payload.errors[0];

    expect(firstError.id).toBe(traceId);
    expect(firstError.status).toBe('404');
    expect(firstError.code).toBe('ERR_NOT_FOUND');
    expect(firstError.detail).toContain('message not found');
  });
});
