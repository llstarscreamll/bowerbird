# Project state

## Decisions

- JSON:API with `meta._debug` (raw error/stack) for local/dev observability.
- Backend errors via `appErrors.Wrap` and central `api.Wrap`.
- Dual UI feedback: `hlm-alert` for contextual 4xx/validation; `ToastService` + Sonner for global 5xx/network.
- PROD-SYNC-089: no per-tenant DLQs; worker controlled-failure (payload limits, timeout, panic recovery, continue).
- Inbox moves toward a standard mail client. `InboxMessageReceived` stays for invoicing, emitted only on first insert. Gmail OAuth: `gmail.modify` + `gmail.send`. Microsoft: `Mail.ReadWrite` + `Mail.Send`. Sync is async via SQS (`InboxSyncAccount`) and incremental via Gmail History.
- Tenant product access: catalog + entitlements in `internal/entitlements` (not `internal/platform`). JWT is identity-only; operators and feature grants resolve at runtime. Default pack: invoicing + mail without send.
- Auth hardening: refresh tokens are JWT+jti persisted in control-plane `refresh_tokens` (rotate on refresh, revoke on logout/delete). Identity OAuth uses random HttpOnly state cookie. Google requires `verified_email`. Tenant DB access requires membership when user claims are present. CORS reflects an exact Origin allowlist with credentials (never `*`).

## Memory

- For UI feedback, prefer interceptor translation over inline translations.
