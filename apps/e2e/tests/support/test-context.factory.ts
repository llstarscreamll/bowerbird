import type { LocalUserCredentials } from './user.factory';
import type { AuthSession, PlatformApiClient, TenantContext } from './platform-api.client';

export type AuthenticatedTenantContext = {
  auth: AuthSession;
  tenant: TenantContext;
  user: LocalUserCredentials;
};

export async function bootstrapAuthenticatedTenantContext(newUser: LocalUserCredentials, platformApi: PlatformApiClient): Promise<AuthenticatedTenantContext> {
  await platformApi.registerLocalOrFail(newUser);
  const auth = await platformApi.loginLocalOrFail(newUser);

  const randomId = Date.now();
  const tenant = await platformApi.createTenantOrFail(auth, {
    name: `E2E Inbox ${randomId}`,
    slug: `e2e-inbox-${randomId}`,
  });

  return { auth, tenant, user: newUser };
}
