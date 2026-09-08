import { expect, type APIResponse } from '@playwright/test';
import { readJson } from './http-assertions';

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

export async function expectJsonApiError(response: APIResponse, operation: string): Promise<JsonApiErrorDocument> {
  const contentType = response.headers()['content-type'] || '';
  if (!contentType.includes('application/vnd.api+json')) {
    const body = (await response.text()).slice(0, 800);
    expect(contentType, `${operation}: expected JSON:API content-type, got "${contentType}". body=${body}`).toContain('application/vnd.api+json');
  }

  const payload = await readJson<JsonApiErrorDocument>(response, operation);
  expect(payload.errors, `${operation}: JSON:API document must include errors[]`).toBeTruthy();
  expect(Array.isArray(payload.errors), `${operation}: errors must be an array`).toBeTruthy();
  expect(payload.errors.length, `${operation}: errors[] must not be empty`).toBeGreaterThan(0);

  const firstError = payload.errors[0];
  expect(firstError.status, `${operation}: errors[0].status is missing`).toBeTruthy();
  expect(firstError.code, `${operation}: errors[0].code is missing`).toBeTruthy();
  expect(firstError.title, `${operation}: errors[0].title is missing`).toBeTruthy();
  expect(firstError.detail, `${operation}: errors[0].detail is missing`).toBeTruthy();
  expect(firstError.meta, `${operation}: errors[0].meta is missing`).toBeTruthy();
  expect(firstError.meta?.timestamp, `${operation}: errors[0].meta.timestamp is missing`).toBeTruthy();

  return payload;
}
