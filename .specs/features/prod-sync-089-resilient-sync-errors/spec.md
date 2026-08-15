# Spec — resilient UX for sync failures

## Status

- Status: Approved for design and implementation
- Date: 2026-05-27
- Epic: PROD-SYNC-089

## Goal

Standardize sync errors for external mailboxes (Gmail, Outlook/Hotmail, Yahoo) so the UI shows actionable states—no infinite loaders or generic messages—under a consistent, safe JSON:API contract.

## Business context

- Silent sync failures drive churn and support tickets.
- OAuth revocation and rate limiting are handled inconsistently today.
- Product needs a reusable error-state catalog for modular UX.

## Functional requirements

### RF-ERR (error contract and actionable UX)

- [PROD-SYNC-089-RF-001] Every 4xx/5xx sync endpoint error returns JSON:API `errors[]`.
- [PROD-SYNC-089-RF-002] Each sync error includes `links.about` with a self-describing help URL per error type.
- [PROD-SYNC-089-RF-003] `detail` names the provider and affected account (e.g. Outlook account `personal@outlook.com` needs attention).
- [PROD-SYNC-089-RF-004] When OAuth reconnect is required: `meta.requires_reauth=true` and `meta.provider=<PROVIDER>` for an immediate UI CTA.
- [PROD-SYNC-089-RF-005] When rate-limited or temporarily blocked: `meta.retry_after_seconds` for countdown and temporary retry disable.
- [PROD-SYNC-089-RF-006] No sync failure leaves indeterminate UI (infinite spinner or blank screen).

### RF-SEC-BE (backend / ingestion security)

- [PROD-SYNC-089-RF-007] Access and refresh tokens persist encrypted at rest (AES-GCM-256 or equivalent) in infrastructure.
- [PROD-SYNC-089-RF-008] Raw MIME/EML may live in S3 for audit; operational UI data persists sanitized.
- [PROD-SYNC-089-RF-009] Structural fields (from/to/date) validate against RFC formats; invalid or malicious content is rejected or cleaned.
- [PROD-SYNC-089-RF-010] Cap text/HTML size before parse to mitigate memory exhaustion from extreme payloads.
- [PROD-SYNC-089-RF-011] Sync worker fails controlled on malicious/heavy payloads (e.g. textual zip bomb): log, ack/nack per policy, continue other messages.

### RF-SEC-FE (Angular display security)

- [PROD-SYNC-089-RF-012] Sanitize rich email HTML before render (no scripts, event handlers, or disallowed iframes).
- [PROD-SYNC-089-RF-013] Block external images by default; require explicit user opt-in.
- [PROD-SYNC-089-RF-014] Mail links use `target="_blank"` and `rel="noopener noreferrer"`.
- [PROD-SYNC-089-RF-015] Render enriched HTML in a sandboxed `iframe` with no script execution.

### RF-PRIV (privacy / error opacity)

- [PROD-SYNC-089-RF-016] `detail` and `meta` never expose secrets (tokens, passwords, crypto signatures).
- [PROD-SYNC-089-RF-017] In `meta._debug` (dev/local), mask PII/secrets before serializing.

## Non-functional requirements

- [PROD-SYNC-089-RNF-001] Modular design by bounded context (`connections`, `inbox`, `platform`, PWA feature stores/components).
- [PROD-SYNC-089-RNF-002] Business errors use stable semantic codes (not provider-coupled).
- [PROD-SYNC-089-RNF-003] New providers (e.g. iCloud) extend without changing core alert UI components.
- [PROD-SYNC-089-RNF-004] Telemetry traceable by tenant, provider, account_id, correlation_id.

## Architecture constraints

- [PROD-SYNC-089-ARC-001] No per-tenant DLQs (operational cost).
- [PROD-SYNC-089-ARC-002] Resilience via logical isolation in the worker: per-message resource limits, timeout/cancel via context, panic handling, and controlled discard of bad messages so the queue is not degraded globally.

## Use cases

- [PROD-SYNC-089-UC-001] Revoked Outlook token → JSON:API with `requires_reauth=true` → UI alert with reconnect CTA.
- [PROD-SYNC-089-UC-002] Yahoo/Microsoft rate limit → `retry_after_seconds=120` → UI countdown and blocked retry.
- [PROD-SYNC-089-UC-003] Malicious HTML payload → backend cleans/limits; frontend sanitizes and sandboxes without script execution.
- [PROD-SYNC-089-UC-004] Oversized / textual zip-bomb message → worker fails that message controlled and keeps processing the queue.

## Acceptance criteria

1. No sync endpoint returns errors outside JSON:API.
2. Stage/UAT: no infinite loading on provider failures.
3. Reauth and rate-limit countdown work from `meta` contract fields.
4. Security tests cover sanitization, external image block, and safe links.
5. Resilience tests show one bad message does not block normal queue consumption.

## Out of scope

- Per-tenant dedicated DLQ.
- Full observability redesign beyond minimum sync-traceability fields.
