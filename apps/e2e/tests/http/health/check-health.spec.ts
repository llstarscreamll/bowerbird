import { expect } from '@playwright/test';
import { test } from '../../support/api.fixture';
import { expectStatus, readJson } from '../../support/http-assertions';

const OPERATION = 'GET /api/health';

test.describe(OPERATION, () => {
  test('responde ok', async ({ platformApi }) => {
    // given

    // when
    const response = await platformApi.call('/api/health');

    // then
    await expectStatus(response, 200, OPERATION);
    const payload = await readJson<{ status: string }>(response, OPERATION);
    expect(payload.status, `${OPERATION}: status`).toBe('ok');
  });
});
