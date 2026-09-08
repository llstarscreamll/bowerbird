import { expect } from '@playwright/test';
import { test } from '../../support/api.fixture';
import { expectStatus, readJson } from '../../support/http-assertions';

const OPERATION = 'GET /health';

test.describe(OPERATION, () => {
  test('responde ok', async ({ platformApi }) => {
    // given

    // when
    const response = await platformApi.call('/health');

    // then
    await expectStatus(response, 200, OPERATION);
    const payload = await readJson<{ status: string }>(response, OPERATION);
    expect(payload.status, `${OPERATION}: status`).toBe('ok');
  });
});

test.describe('GET /api/health', () => {
  test('responde ok', async ({ platformApi }) => {
    // given
    const operation = 'GET /api/health';

    // when
    const response = await platformApi.call('/api/health');

    // then
    await expectStatus(response, 200, operation);
    const payload = await readJson<{ status: string }>(response, operation);
    expect(payload.status, `${operation}: status`).toBe('ok');
  });
});
