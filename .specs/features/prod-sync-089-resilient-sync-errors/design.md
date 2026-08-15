# Design — PROD-SYNC-089

## Architecture scope

End-to-end resilient, safe error contract for mail sync:

- Backend Go: provider failure → consistent JSON:API → actionable `meta` for UI
- Frontend Angular: contextual alerts with dynamic actions (`reauth`, `retry_after_seconds`)
- Security: mail content sanitization (BE + FE); PII masking in errors
- Workers: tolerate bad messages without per-tenant DLQs

## DDD and modularity

### Bounded contexts

| Context       | Role                                                                   |
| ------------- | ---------------------------------------------------------------------- |
| `platform`    | `apperrors` taxonomy + safe payload; `http/api` JSON:API serialization |
| `connections` | OAuth/provider errors → semantic business codes                        |
| `inbox`       | Sync pipeline, parse guards, worker resilience                         |
| `apps/pwa`    | Interceptor + store/component for actionable UX                        |

### Anti-corruption layer

Raw SDK/provider errors (Google, Microsoft, Yahoo) never reach the frontend. Translate to canonical codes:

- `ERR_SYNC_REAUTH_REQUIRED`
- `ERR_SYNC_RATE_LIMITED`
- `ERR_SYNC_PROVIDER_TEMPORARY`
- `ERR_SYNC_PAYLOAD_REJECTED`
- `ERR_SYNC_INTERNAL`

## Backend design

### Canonical error model

Enriched error type in `platform/apperrors` (or dedicated sync errors package):

- Base: `code`, `message`, `cause`
- Safe context: `provider`, `account_email_masked`, `requires_reauth`, `retry_after_seconds`, `help_path`
- Redaction helpers for secrets/PII

Rule: `message`/`detail` never contain tokens or raw provider payloads.

### JSON:API adapter

Extend `api.MapError` / `api.Wrap` to:

- Build `errors[0].code` / `title` / `detail` / `status`
- Emit `links.about` from a central error-docs catalog
- Emit `meta` only via whitelist
- Keep correlation `id` (`sentry-trace` or equivalent)
- Keep `meta._debug` only in local/dev, already redacted

### Sync error catalog

Deterministic mapping table/file:

- Input: provider error type + context (HTTP status, subcode, headers, retry-after)
- Output: canonical code + HTTP status + UX flags (`requires_reauth`, `retry_after_seconds`)

Goal: add providers without changing core frontend.

### Worker resilience (no per-tenant DLQ)

Controlled per-message failure:

1. Size guards — max bytes for MIME/HTML/text; early reject before deep parse
2. Time/memory — `context.WithTimeout` per message; cooperative cancel in parsers and I/O
3. Robustness — `recover()` per work unit; classify malicious/invalid payload as non-retriable
4. Continuity — structured incident log, mark message failed, continue to next queue message

Keep current queue topology; no per-tenant DLQs.

### Ingestion security

- OAuth tokens encrypted at rest (`connections` repo layer)
- Raw MIME in S3 (audit); sanitized operational data in DB
- RFC validation for email/date fields
- Server-side sanitization so untrusted active HTML is not persisted for UI

## Frontend Angular design

### Typed error contract

Shared JSON:API error type in `core` with sync `meta`:

- `requires_reauth?: boolean`
- `provider?: 'GMAIL' | 'OUTLOOK' | 'YAHOO' | 'HOTMAIL' | string`
- `retry_after_seconds?: number`
- `account_email?: string`

### UX behavior

- 5xx/network → interceptor toast (unchanged)
- 4xx sync business → persistent contextual alert in component/store
- `requires_reauth` → CTA + provider login redirect
- `retry_after_seconds` → visual countdown + disable retry

### Email render security

- Strict DOMPurify sanitization with Angular security hooks
- Enforce link `target` + `rel`
- External images blocked by default; “show images” opt-in
- Isolated render via `iframe sandbox` for enriched content

## Observability

Structured events: `sync_error_classified`, `sync_reauth_required`, `sync_rate_limited`, `sync_payload_rejected`.

Minimum fields: `tenant_id`, `provider`, `account_id`, `error_code`, `correlation_id`.

## Test strategy

**Backend**

- Unit: provider error → canonical; redaction of `detail` / `meta._debug`
- Integration: handler returns strict JSON:API with `links.about` and expected `meta`
- Worker resilience: oversized/malicious message fails; next message succeeds

**Frontend**

- Unit: `meta` parse + branching; countdown disables until 0; sanitization strips scripts/handlers
- Component/e2e: contextual alert visible; correct CTA per provider

## Risks and mitigations

| Risk                                | Mitigation                                 |
| ----------------------------------- | ------------------------------------------ |
| Over-exposure in debug errors       | Central redaction + payload snapshot tests |
| False positives from payload limits | Configurable limits + tuning metrics       |
| BE/FE contract drift                | Shared typed contract + contract tests     |
