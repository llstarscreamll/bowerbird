import { expect, type APIResponse } from '@playwright/test';

export type JsonApiErrorObject = {
  id?: string;
  status?: string;
  code?: string;
  title?: string;
  detail?: string;
  meta?: Record<string, unknown>;
};

export type JsonApiErrorDocument = {
  errors: JsonApiErrorObject[];
};

export async function expectJsonApiError(response: APIResponse): Promise<JsonApiErrorDocument> {
  const contentType = response.headers()['content-type'] || '';
  expect(contentType).toContain('application/vnd.api+json');

  const payload = (await response.json()) as JsonApiErrorDocument;
  expect(Array.isArray(payload.errors)).toBeTruthy();
  expect(payload.errors.length).toBeGreaterThan(0);

  const firstError = payload.errors[0];
  expect(firstError.status).toBeTruthy();
  expect(firstError.code).toBeTruthy();
  expect(firstError.title).toBeTruthy();
  expect(firstError.detail).toBeTruthy();
  expect(firstError.meta).toBeTruthy();
  expect(firstError.meta?.timestamp).toBeTruthy();

  return payload;
}
