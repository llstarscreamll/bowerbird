# Error Management & Propagation Specification

## Core Requirements

1. **Standardized JSON:API Responses:**
   - All errors returned by the backend must strictly adhere to the JSON:API error format.
   - A centralized HTTP error handler (`api.Wrap`) must catch all errors returned by domain controllers and map them to standard responses, preventing leaky or improperly formatted errors (like plain text fallbacks).
   - Traceability must be maintained by injecting and returning a unique correlation ID (`sentry-trace`) in the JSON:API `id` field.

2. **Environment-Aware Debugging:**
   - **Production:** Internal server errors (500s) must obscure stack traces and underlying infrastructure messages to protect system internals.
   - **Development/Local:** The API must inject a `meta._debug` node containing the raw Go error and a clean stack trace.
   - The frontend global interceptor should listen for `meta._debug` and output formatted, collapsed logs to the browser console to aid developer and AI agent debugging.

3. **Backend Error Hierarchy (Go):**
   - Controllers must not handle HTTP responses directly for errors. They must return standard Go `error` types.
   - The platform layer provides `apperrors` to construct and wrap domain-specific errors (e.g., `CodeValidation`, `CodeConflict`) without coupling the domain to HTTP statuses.
   - The mapping layer (`api.MapError`) evaluates `errors.As` / `errors.Is` to determine the proper HTTP status code and standard message for the JSON:API response.

4. **Frontend UI Feedback (Angular):**
   - The frontend implements a dual-feedback mechanism to communicate errors to the user, strictly split by context:
   - **Toast (`ToastService`):** Global, transient alerts. Must be used exclusively for:
     - 5xx (Server Errors).
     - 0 (Network Errors).
     - Global success notifications ("Saved successfully").
     - These are automatically managed by the HTTP interceptor without component intervention.
   - **Alert (`<app-alert>`):** Contextual, static, and persistent callouts. Must be used for:
     - 4xx (Client Errors / Validation).
     - Domain logic rejections (e.g., "Email already registered").
     - Components must handle these manually by reading the enriched error object thrown by the interceptor.

5. **Internationalization (i18n) of Errors:**
   - The backend must return semantic error codes (e.g., `ERR_USER_NOT_FOUND`), not hardcoded user-facing text.
   - The frontend's `ErrorTranslationService` intercepts these codes and maps them to localized, user-friendly strings in Spanish.

## Use Cases

- [ERR-001] **Network Drop:** User submits form without internet -> Angular Interceptor catches status 0 -> Triggers Toast with "Error de conexión".
- [ERR-002] **Backend Panic / Unhandled Exception:** DB drops connection -> Controller returns raw error -> `api.Wrap` catches it -> Returns 500 JSON:API -> In Dev: shows stacktrace in `meta._debug` -> Interceptor logs trace to console and shows generic Toast to user.
- [ERR-003] **Domain Validation Failure:** User enters duplicated slug -> Controller returns `appErrors.Wrap(err, CodeConflict, "slug exists")` -> `api.Wrap` maps to 409 JSON:API -> Interceptor translates code to "El identificador ya existe" -> Throws to component -> Component renders `<app-alert>` above the form.
