# Project State

## Decisions

- Use JSON:API with `meta._debug` containing raw error/stacktrace for local/dev observability.
- Standardize backend error propagation using `apperrors.Wrap` and a central HTTP adapter (`api.Wrap`).
- Implement a dual UI feedback pattern in frontend: `AlertComponent` for contextual 4xx errors, `ToastService` for global 5xx/network errors.
- For PROD-SYNC-089, do not implement tenant-specific DLQs; implement controlled-failure resilience in workers (payload limits, timeout, panic recovery, and continue processing).

## Memory

- When doing UI feedback, rely on interceptor translation vs inline translations.
