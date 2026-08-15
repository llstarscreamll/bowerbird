export const GLOBAL_ROUTES = new Set(['', 'login', 'lobby', 'platform', 'workspaces', 'onboarding', 'profile']);

const AUTH_COOKIE_ENDPOINT_PATHS = ['/api/v1/auth/login-local', '/api/v1/auth/register-local', '/api/v1/auth/refresh', '/api/v1/auth/logout'];

export function requiresCookieAuth(url: string): boolean {
  return AUTH_COOKIE_ENDPOINT_PATHS.some((endpoint) => url.includes(endpoint));
}

export function shouldSkipAuthRefresh(url: string): boolean {
  return url.includes('/api/v1/auth/login-local') || url.includes('/api/v1/auth/register-local') || url.includes('/api/v1/auth/refresh') || url.includes('/api/v1/auth/logout');
}
