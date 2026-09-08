import { expect, type APIResponse } from '@playwright/test';

const preview = async (response: APIResponse): Promise<string> => {
  const body = await response.text();
  return body.length > 800 ? `${body.slice(0, 800)}…` : body;
};

export async function expectStatus(response: APIResponse, expected: number, operation: string): Promise<void> {
  const actual = response.status();
  if (actual === expected) {
    return;
  }

  expect(actual, `${operation} expected HTTP ${expected}, got ${actual}. body=${await preview(response)}`).toBe(expected);
}

export async function expectClientError(response: APIResponse, operation: string): Promise<void> {
  const actual = response.status();
  if (actual >= 400 && actual < 500) {
    return;
  }

  expect(actual, `${operation} expected 4xx (denied/invalid), got ${actual}. body=${await preview(response)}`).toBeGreaterThanOrEqual(400);
  expect(actual, `${operation} expected 4xx, got ${actual}`).toBeLessThan(500);
}

export async function readJson<T>(response: APIResponse, operation: string): Promise<T> {
  const body = await response.text();
  try {
    return JSON.parse(body) as T;
  } catch {
    throw new Error(`${operation} expected JSON (HTTP ${response.status()}). body=${body.slice(0, 800)}`);
  }
}
