# Spec — error management and propagation

## Core requirements

### Standardized JSON:API responses

- All backend errors use JSON:API error format.
- Central handler `api.Wrap` maps controller errors to standard responses (no plain-text leaks).
- Correlation ID (`sentry-trace`) goes in the JSON:API `id` field.

### Environment-aware debugging

- **Production:** 500s hide stack traces and infrastructure messages.
- **Dev/local:** `meta._debug` includes raw Go error and a clean stack trace.
- Frontend interceptor logs `meta._debug` as formatted, collapsed console output for developers/agents.

### Backend error hierarchy (Go)

- Controllers return `error`; they do not write HTTP error responses directly.
- `apperrors` wraps domain errors (e.g. `CodeValidation`, `CodeConflict`) without HTTP coupling.
- `api.MapError` uses `errors.As` / `errors.Is` for status and standard JSON:API messaging.

### Frontend UI feedback (Angular)

Dual feedback by context:

| Channel                | Use for                                                                           |
| ---------------------- | --------------------------------------------------------------------------------- |
| Toast (`ToastService`) | 5xx, network (0), global success — interceptor-driven                             |
| Alert (`<app-alert>`)  | 4xx / validation / domain rejections — component reads enriched interceptor error |

### Error i18n

- Backend returns semantic codes (e.g. `ERR_USER_NOT_FOUND`), not user-facing copy.
- Frontend `ErrorTranslationService` maps codes to localized Spanish strings.

## Use cases

- [ERR-001] **Network drop:** form submit offline → interceptor status 0 → toast “Error de conexión”.
- [ERR-002] **Unhandled backend failure:** DB down → controller returns error → `api.Wrap` → 500 JSON:API; in dev `meta._debug` + console log + generic toast.
- [ERR-003] **Domain validation:** duplicate slug → `appErrors.Wrap(..., CodeConflict, ...)` → 409 JSON:API → translated code → component `<app-alert>` above the form.
