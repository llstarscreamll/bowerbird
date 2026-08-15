# Project State

## Decisions

- Use JSON:API with `meta._debug` containing raw error/stacktrace for local/dev observability.
- Standardize backend error propagation using `appErrors.Wrap` and a central HTTP adapter (`api.Wrap`).
- Implement a dual UI feedback pattern in frontend: `hlm-alert` for contextual 4xx/validation errors, `ToastService` + Sonner for global 5xx/network errors.
- For PROD-SYNC-089, do not implement tenant-specific DLQs; implement controlled-failure resilience in workers (payload limits, timeout, panic recovery, and continue processing).
- Inbox evolves from invoice ingestion to a standard mail client. `InboxMessageReceived` stays for invoicing but is emitted only on first insert. Gmail OAuth uses `gmail.modify` + `gmail.send`. Microsoft connections use `Mail.ReadWrite` + `Mail.Send`. Sync is async via SQS (`InboxSyncAccount`) and incremental via Gmail History.
- Tenant product access is catalog + entitlements in `internal/entitlements`, not in `internal/platform`. JWT stays identity-only; platform operators and feature grants are resolved at runtime. Default pack is invoicing + mail without send.

## Memory

- When doing UI feedback, rely on interceptor translation vs inline translations.
