# Authentication

Mixed session: short-lived JWT access token + HttpOnly refresh cookie.

## Access token (JWT)

- Short TTL (about 15 minutes).
- Returned in JSON (`login-local`, `refresh`).
- Stored in memory only (SignalStore) — never `localStorage` / `sessionStorage`.
- Sent as `Authorization: Bearer <token>`.

## Refresh token

- Longer TTL (about 7 days).
- Set via `Set-Cookie`: `HttpOnly`, `Secure`, `SameSite=Strict`.
- Browser attaches it automatically to `/api/v1/auth/refresh`.

## OAuth (Google / Microsoft)

1. After OAuth exchange, backend sets the refresh cookie and 302s to `/lobby` with no tokens in the URL.
2. `authGuard` sees no in-memory access token and calls refresh.
3. Backend returns a new access token; lobby loads.

`auth.interceptor.ts`: on 401, refresh once and retry.
