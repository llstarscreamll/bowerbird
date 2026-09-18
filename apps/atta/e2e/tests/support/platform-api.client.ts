import type { APIRequestContext, APIResponse } from '@playwright/test';
import type { LocalUserCredentials } from './user.factory';

export type AuthSession = {
  accessToken: string;
};

export type TenantContext = {
  tenantId: string;
  tenantSlug: string;
};

export type OrganizationSummary = {
  id: string;
  name: string;
  slug: string;
  status: string;
  created_at: string;
  members_count?: number;
  current_user_role?: string;
};

type LoginLocalResponse = {
  access_token: string;
  expires_in: number;
};

type ConnectionListResponse = {
  data: Array<{
    id: string;
    provider: string;
    provider_account_email: string;
    status: string;
    sharing_policy: string;
  }>;
};

export class PlatformApiClient {
  constructor(
    private readonly request: APIRequestContext,
    private readonly apiBaseUrl: string,
  ) {}

  async registerLocalOrFail(credentials: LocalUserCredentials): Promise<void> {
    const response = await this.request.post(`${this.apiBaseUrl}/api/v1/auth/register-local`, {
      data: {
        email: credentials.email,
        password: credentials.password,
      },
    });

    if (response.ok()) {
      return;
    }

    const body = await response.text();
    throw new Error(`POST /api/v1/auth/register-local failed: HTTP ${response.status()} at ${this.apiBaseUrl}. body=${body}`);
  }

  loginLocal(credentials: LocalUserCredentials): Promise<APIResponse> {
    return this.request.post(`${this.apiBaseUrl}/api/v1/auth/login-local`, {
      data: {
        email: credentials.email,
        password: credentials.password,
      },
    });
  }

  async loginLocalOrFail(credentials: LocalUserCredentials): Promise<AuthSession> {
    const response = await this.loginLocal(credentials);

    if (!response.ok()) {
      const body = await response.text();
      throw new Error(`POST /api/v1/auth/login-local failed: HTTP ${response.status()} at ${this.apiBaseUrl}. body=${body}`);
    }

    const payload = (await response.json()) as LoginLocalResponse;
    return {
      accessToken: payload.access_token,
    };
  }

  async createTenantOrFail(auth: AuthSession, input: { name: string; slug: string }): Promise<TenantContext> {
    const response = await this.createTenant(auth, input);

    if (!response.ok()) {
      const body = await response.text();
      throw new Error(`POST /api/v1/tenants failed: HTTP ${response.status()} at ${this.apiBaseUrl}. body=${body}`);
    }

    const payload = (await response.json()) as OrganizationSummary;
    return {
      tenantId: payload.id,
      tenantSlug: payload.slug,
    };
  }

  createTenant(auth: AuthSession, input: { name: string; slug: string }): Promise<APIResponse> {
    return this.request.post(`${this.apiBaseUrl}/api/v1/tenants`, {
      headers: {
        Authorization: `Bearer ${auth.accessToken}`,
      },
      data: {
        name: input.name,
        slug: input.slug,
      },
    });
  }

  getTenant(auth: AuthSession, tenantId: string): Promise<APIResponse> {
    return this.request.get(`${this.apiBaseUrl}/api/v1/tenants/${encodeURIComponent(tenantId)}`, {
      headers: {
        Authorization: `Bearer ${auth.accessToken}`,
      },
    });
  }

  async listConnectionsJson(auth: AuthSession, tenant: TenantContext, traceId?: string): Promise<ConnectionListResponse> {
    const response = await this.listConnections(auth, tenant, traceId);
    if (!response.ok()) {
      const body = await response.text();
      throw new Error(`GET /api/v1/connections failed: HTTP ${response.status()} at ${this.apiBaseUrl}. body=${body}`);
    }

    return (await response.json()) as ConnectionListResponse;
  }

  listConnections(auth: AuthSession, tenant: TenantContext, traceId?: string): Promise<APIResponse> {
    return this.request.get(`${this.apiBaseUrl}/api/v1/connections`, {
      headers: this.authHeaders(auth, tenant, traceId),
    });
  }

  listInboxMessages(auth: AuthSession, tenant: TenantContext, traceId?: string): Promise<APIResponse> {
    return this.request.get(`${this.apiBaseUrl}/api/v1/inbox/messages`, {
      headers: this.authHeaders(auth, tenant, traceId),
    });
  }

  listInboxSyncStatus(auth: AuthSession, tenant: TenantContext, traceId?: string): Promise<APIResponse> {
    return this.request.get(`${this.apiBaseUrl}/api/v1/inbox/sync-status`, {
      headers: this.authHeaders(auth, tenant, traceId),
    });
  }

  triggerInboxSync(auth: AuthSession, tenant: TenantContext, accountId?: string, traceId?: string): Promise<APIResponse> {
    return this.request.post(`${this.apiBaseUrl}/api/v1/inbox/sync`, {
      headers: this.authHeaders(auth, tenant, traceId),
      data: accountId ? { account_id: accountId } : {},
    });
  }

  getInboxMessage(auth: AuthSession, tenant: TenantContext, messageId: string, traceId?: string): Promise<APIResponse> {
    return this.request.get(`${this.apiBaseUrl}/api/v1/inbox/messages/${encodeURIComponent(messageId)}`, {
      headers: this.authHeaders(auth, tenant, traceId),
    });
  }

  sendInboxMessage(auth: AuthSession, tenant: TenantContext, payload: Record<string, unknown>, traceId?: string): Promise<APIResponse> {
    return this.request.post(`${this.apiBaseUrl}/api/v1/inbox/messages`, {
      headers: this.authHeaders(auth, tenant, traceId),
      data: payload,
    });
  }

  call(
    path: string,
    init: {
      method?: 'GET' | 'POST' | 'PATCH' | 'PUT' | 'DELETE';
      auth?: AuthSession;
      token?: string;
      tenant?: TenantContext;
      tenantIdHeader?: string;
      data?: unknown;
      rawBody?: string;
      contentType?: string;
      traceId?: string;
    } = {},
  ): Promise<APIResponse> {
    const headers: Record<string, string> = {};
    if (init.token) {
      headers.Authorization = `Bearer ${init.token}`;
    } else if (init.auth) {
      headers.Authorization = `Bearer ${init.auth.accessToken}`;
    }
    if (init.tenantIdHeader !== undefined) {
      headers['X-Tenant-ID'] = init.tenantIdHeader;
    } else if (init.tenant) {
      headers['X-Tenant-ID'] = init.tenant.tenantSlug;
    }
    if (init.contentType) {
      headers['Content-Type'] = init.contentType;
    }
    if (init.traceId) {
      headers['sentry-trace'] = init.traceId;
    }

    return this.request.fetch(`${this.apiBaseUrl}${path}`, {
      method: init.method ?? 'GET',
      headers,
      ...(init.rawBody !== undefined ? { data: init.rawBody } : init.data !== undefined ? { data: init.data } : {}),
    });
  }

  private authHeaders(auth: AuthSession, tenant: TenantContext, traceId?: string): Record<string, string> {
    return {
      Authorization: `Bearer ${auth.accessToken}`,
      'X-Tenant-ID': tenant.tenantSlug,
      ...(traceId ? { 'sentry-trace': traceId } : {}),
    };
  }
}
