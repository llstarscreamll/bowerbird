# Tasks — PROD-SYNC-089

## Overview

- Total tasks: 10
- Parallelizable: 3 (`[P]`)
- Final gate: lint + test + build for backend and PWA

## Tasks

### T1 — Canonical sync error catalog

- Type: Backend
- Depends on: none
- What:
  - Define canonical codes (`ERR_SYNC_REAUTH_REQUIRED`, `ERR_SYNC_RATE_LIMITED`, etc.).
  - Map to HTTP status and `links.about`.
  - Define allowed `meta` schema.
- Where:
  - `apps/backend/internal/platform/apperrors/*`
  - `apps/backend/internal/platform/http/api/*`
- Done when:
  - Versioned catalog exists and is used by the central mapper.
  - No hardcoded codes outside the catalog.
- Tests: unit — code → status/title/help mapping.

### T2 — JSON:API serialization with actionable meta and masking

- Type: Backend
- Depends on: T1
- What:
  - Extend `api.Wrap` / `MapError` for `links.about` and sync meta.
  - Whitelist `meta`.
  - Redact secrets in `detail` and `meta._debug`.
- Where:
  - `apps/backend/internal/platform/http/api/errors.go`
  - `apps/backend/internal/platform/http/api/errors_test.go`
- Done when:
  - Sync errors are valid JSON:API with expected fields.
  - No sensitive data leaks in error responses.
- Tests: unit + integration for JSON:API payload.

### T3 — Provider error → canonical translator

- Type: Backend
- Depends on: T1
- What:
  - Translate revoked/expired OAuth and rate limiting per provider.
  - Extract `retry_after_seconds` when present.
- Where:
  - `apps/backend/internal/connections/application/*`
  - `apps/backend/internal/inbox/application/*`
- Done when: external errors never surface raw; always go through the canonical mapper.
- Tests: unit per provider for main scenarios.

### T4 — Worker resilience guardrails (no per-tenant DLQ)

- Type: Backend
- Depends on: T3
- What:
  - Payload size limits (MIME/HTML/text).
  - Per-message timeout with context cancel.
  - Panic recovery per work unit.
  - Classify malicious/oversized payload as non-retriable; continue queue.
- Where:
  - `apps/backend/internal/inbox/application/*`
  - `apps/backend/internal/inbox/infrastructure/*`
- Done when: one bad message fails controlled and does not block later processing.
- Tests: integration — bad message then valid message in queue.

### T5 — Structural validation and server-side mail sanitization

- Type: Backend
- Depends on: T4
- What:
  - Validate `from` / `to` / `date` (RFC).
  - Clean operational content before persist.
- Where:
  - `apps/backend/internal/inbox/domain/*`
  - `apps/backend/internal/inbox/application/*`
- Done when: invalid/malicious data does not reach the persisted operational model.
- Tests: unit for validators and sanitizers.

### T6 [P] — Typed sync error contract on frontend

- Type: Frontend
- Depends on: T2
- What: Extend JSON:API error types for `requires_reauth`, `provider`, `retry_after_seconds`.
- Where: `apps/pwa/src/app/core/*`
- Done when: interceptor/store consume typed meta without unsafe casts.
- Tests: unit for parsing/typing.

### T7 [P] — Actionable UX: contextual alert + reauth CTA + countdown

- Type: Frontend
- Depends on: T6
- What:
  - Alert for 4xx sync with reconnect CTA.
  - Rate-limit countdown with retry disabled until expiry.
- Where:
  - `apps/pwa/src/app/features/inbox/*`
  - `apps/pwa/src/app/core/interceptors/error.interceptor.ts`
- Done when: reauth/rate-limit scenarios render correctly with no infinite spinner.
- Tests: unit/component for state and countdown.

### T8 [P] — Frontend HTML visual security layer

- Type: Frontend
- Depends on: none
- What:
  - Integrate DOMPurify.
  - Block external images by default with opt-in.
  - Secure links (`target`, `rel`).
  - Isolated render with `iframe sandbox`.
- Where: `apps/pwa/src/app/features/inbox/*`
- Done when: enriched HTML runs no scripts and loads no external images without consent.
- Tests: unit for sanitization and anchor/img mutation.

### T9 — Observability and support events

- Type: Backend + Frontend
- Depends on: T2, T4, T7
- What: Emit structured logs/events for sync errors and UX decisions.
- Where:
  - `apps/backend/internal/*`
  - `apps/pwa/src/app/*`
- Done when: a sync error is traceable end-to-end by `correlation_id`.
- Tests: assert minimum log fields where utility tests apply.

### T10 — Final validation (feature gate)

- Type: Verification
- Depends on: T1..T9
- What:
  - Run lint/test/build for impacted modules.
  - Manual Stage/UAT smoke against acceptance criteria.
- Commands:
  - `pnpm --filter @bowerbird/backend lint`
  - `pnpm --filter @bowerbird/backend test`
  - `pnpm --filter @bowerbird/backend build`
  - `pnpm --filter @bowerbird/pwa lint`
  - `pnpm --filter @bowerbird/pwa test`
  - `pnpm --filter @bowerbird/pwa build`
- Done when: all tests pass and acceptance criteria are evidenced.

## Recommended order

1. T1 → T2 → T3 → T4 → T5
2. In parallel: T6, T7, T8
3. T9
4. T10

## Implementation notes

- Keep `api.Wrap(handler, isDev)` and `appErrors.Wrap(...)`.
- Do not use `http.Error()` in handlers.
- No per-tenant DLQ; resilience via worker guardrails.
