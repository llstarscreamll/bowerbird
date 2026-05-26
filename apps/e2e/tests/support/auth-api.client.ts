import type { APIRequestContext, APIResponse } from '@playwright/test';
import type { LocalUserCredentials } from './user.factory';

export class AuthApiClient {
  constructor(
    private readonly request: APIRequestContext,
    private readonly apiBaseUrl: string,
  ) {}

  async registerLocal(credentials: LocalUserCredentials): Promise<APIResponse> {
    return this.request.post(`${this.apiBaseUrl}/api/v1/auth/register-local`, {
      data: {
        email: credentials.email,
        password: credentials.password,
      },
    });
  }

  async registerLocalOrFail(credentials: LocalUserCredentials): Promise<void> {
    const response = await this.registerLocal(credentials);

    if (response.ok()) {
      return;
    }

    const body = await response.text();
    throw new Error(
      `Failed to create local account. status=${response.status()} baseUrl=${this.apiBaseUrl} body=${body}`,
    );
  }
}
