export const GLOBAL_ROUTES = new Set(['', 'login', 'lobby', 'workspaces', 'onboarding', 'profile']);

const AUTH_COOKIE_ENDPOINT_PATHS = ['/refresh', '/logout'];

export function requiresCookieAuth(url: string): boolean {
  return AUTH_COOKIE_ENDPOINT_PATHS.some((endpoint) => url.includes(endpoint));
}
