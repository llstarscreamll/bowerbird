# Error Management Tasks

## 1. [x] Backend: Domain Error Construct

- **What**: Create an application-agnostic error construct that holds a domain code and message.
- **Where**: `apps/backend/internal/platform/apperrors/errors.go`
- **Depends on**: None
- **Reuses**: Go `errors` package pattern
- **Done when**: `New`, `Wrap`, and `Code` mapping are available.
- **Tests**: `apps/backend/internal/platform/apperrors/errors_test.go`
- **Gate**: `cd apps/backend && go test ./internal/platform/apperrors/...`

## 2. [x] Backend: JSON:API Error Mapper & Handler Adapter

- **What**: Create JSON:API error representations, a central mapping logic, and an `http.HandlerFunc` adapter.
- **Where**: `apps/backend/internal/platform/http/api/errors.go` and `apps/backend/internal/platform/http/api/handler.go`
- **Depends on**: Task 1
- **Reuses**: None
- **Done when**: We have a `Wrap` function that takes a `func(w, r) error`, maps returned errors to JSON:API, injects `sentry-trace` ID, and handles `_debug` when `isDev` is true.
- **Tests**: `apps/backend/internal/platform/http/api/errors_test.go`
- **Gate**: `cd apps/backend && go test ./internal/platform/http/api/...`

## 3. [x] Backend: Refactor Controllers

- **What**: Update existing HTTP handlers to return `error` and use the new `api.Wrap` adapter. Update `cmd/api/main.go` to pass the `cfg.AppEnv` environment flag correctly.
- **Where**: Handlers in `health`, `identity`, `inbox`, `organization`, and `cmd/api/main.go`.
- **Depends on**: Task 2
- **Reuses**: New `api.Wrap`
- **Done when**: All endpoints successfully map unhandled errors to JSON:API automatically.
- **Tests**: Existing handler tests
- **Gate**: `cd apps/backend && go test ./...`

## 4. [x] Frontend: Toast Component & Service

- **What**: Create a global Toast service and standalone Tailwind component for transient notifications (e.g. 500 errors, network errors).
- **Where**: `apps/pwa/src/app/core/presentation/components/toast/toast.component.ts` & `apps/pwa/src/app/core/services/toast.service.ts`
- **Depends on**: None
- **Reuses**: `AlertComponent` styles loosely for consistency.
- **Done when**: Service exposes `showError`, `showSuccess`, etc. Component renders them floating.
- **Tests**: Not strict UI test, but verify compilation.
- **Gate**: `cd apps/pwa && pnpm lint`

## 5. [x] Frontend: Global Error Interceptor & Translation

- **What**: Intercept HTTP 4xx/5xx responses, parse JSON:API errors, log `_debug` using `console.group`, and dispatch toasts or component alerts using translated Spanish text.
- **Where**: `apps/pwa/src/app/core/interceptors/error.interceptor.ts` and `apps/pwa/src/app/core/services/error-translation.service.ts`
- **Depends on**: Task 4
- **Reuses**: None
- **Done when**: Interceptor is registered in `app.config.ts`, logs `_debug` properly, triggers toast for 500s.
- **Tests**: `apps/pwa/src/app/core/interceptors/error.interceptor.spec.ts`
- **Gate**: `cd apps/pwa && pnpm test`
