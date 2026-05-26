# AGENTS

## Toolchain and workspace

- Run `mise install` first. Versions are pinned: Node `24`, Go `1.25`, pnpm `10.16.1` (`.mise.toml`, `.nvmrc`, root `package.json`, `apps/backend/go.mod`).
- Use `pnpm` only. Workspace roots are `apps/*` and `packages/*` (`pnpm-workspace.yaml`), orchestrated by Turbo (`turbo.json`).

## Commands that matter

- Root dev flow: `pnpm run dev` (always runs `pnpm run infra:up` first, then `turbo run dev`).
- Root verification flow: `pnpm run lint && pnpm run test && pnpm run build`.
- Root deploy: `pnpm run deploy` (builds first, deploys only `@bowerbird/infra`).
- Backend targeted: `pnpm --filter @bowerbird/backend dev|lint|test|build|migrate:all`.
- PWA targeted: `pnpm --filter @bowerbird/pwa dev|lint|test|build`.
- E2E targeted: `pnpm --filter @bowerbird/e2e lint|test:e2e|test:e2e:ui`.
- Infra targeted: `pnpm --filter @bowerbird/infra lint|test|build|synth|deploy`.

## Backend (`apps/backend`)

- API entrypoint is `cmd/api/main.go`; local `dev` uses Air (`.air.toml`) and sources `apps/backend/.env` if present.
- Error Handling & JSON:API: **Never** use `http.Error()`. Handlers must return `error` and be registered using `api.Wrap(handlerFunc, isDev)`.
- Domain Errors: Wrap or create errors using `apperrors.Wrap(err, apperrors.CodeX, "msg")`. `api.Wrap` automatically converts these to JSON:API payloads and injects `meta._debug` stack traces when `isDev` is true.
- Migrations CLI is `cmd/migrate/main.go`; keep migration sets split between `migrations/controlplane` and `migrations/tenant`.
- Runtime config is env + SSM merge (`internal/platform/config/config.go`).
  - Local default SSM parameter: `/bowerbird/local/secrets`.
  - `inbox_credentials_encryption_key` is required from SSM (no env fallback).
- LocalStack bootstrap script (`apps/backend/scripts/init-localstack.sh`) creates SQS/EventBridge/S3 and writes SSM secrets from `apps/backend/secrets.json`.
  - If `apps/backend/secrets.json` changes, re-run inside container:
    `docker exec bowerbird-localstack /etc/localstack/init/ready.d/init-localstack.sh`.

## PWA (`apps/pwa`)

- Angular standalone + zoneless app. Wiring is in `src/app/app.config.ts`, routes in `src/app/app.routes.ts`.
- Serve command is fixed to `ng serve --host 0.0.0.0 --port 4200`; `angular.json` only allows host `app.bowerbird.dev`.
- Tenant routing: Tenant pages are children of the `/:tenantId` route and wrapped by `TenantLayoutComponent`.
- Tenant header is derived from the `tenantId` param via `core/interceptors/tenant.interceptor.ts`.
- Error Handling & UI Feedback: `error.interceptor.ts` globally handles JSON:API responses and logs `meta._debug` to the console.
  - **Toast (`ToastService`)**: Use for global, transient messages like 5xx server errors, network drops, or success notifications ("Saved successfully"). The interceptor handles 5xx/network toasts automatically.
  - **Alert (`AlertComponent`)**: Use inline within forms or pages for contextual, actionable 4xx validation errors (e.g., "Email already in use"). Components should handle these manually.
- Auth refresh behavior is in `core/interceptors/auth.interceptor.ts` (401 -> refresh -> retry).
- Feature convention: keep business/data orchestration in `*/application/*store.ts`; keep `presentation` components thin.
- Shared inbox/provider primitives now live in `src/app/core/domain/inbox-types.ts` (avoid cross-feature domain imports for these types).
- Shared visual primitives are in `src/styles.css` (`.card`, `.input-field`, `.btn-primary`, `.btn-secondary`).

## E2E Testing (`apps/e2e`)

- Uses Playwright. Always run `pnpm run test:e2e:install` to ensure the local browser is present before running tests.
- Execution requires the local backend to be running (`pnpm run dev`) with `api.bowerbird.dev` accessible (Caddy routing).
- To test the full auth flow, the backend must be in `local` or `development` mode so the `/api/v1/auth/register-local` endpoint is enabled.
- UI doesn't have a signup form yet, so `test.fixture.ts` relies on the API `registerLocalOrFail` directly for setup.
- Commands from root: `pnpm run test:e2e` (headless), `pnpm run test:e2e:ui` (interactive).

## Local infra and deploy constraints

- `docker-compose.yml` runs Postgres `5432`, Redis `6379`, LocalStack `4566`, Caddy `80/443`.
- `Caddyfile` maps `app.bowerbird.dev -> :4200` and `api.bowerbird.dev -> :8080`; use these domains locally for cookie/routing behavior.
- Infra CDK entrypoint is `packages/infra/bin/index.ts` and requires `packages/infra/.env`:
  - `ENV` and `AWS_ACCOUNT_ID` must be set.
  - `AWS_REGION` must be `us-east-1` (enforced).
- Web deploy consumes `apps/pwa/dist/pwa/browser`; build PWA before infra deploy/synth checks that depend on assets.
- In `bowerbird-stack.ts`, S3 deployments use `prune: false` for assets/entrypoints; do not change casually.

## Hooks, formatting, and docs

- Pre-commit runs `pnpm lint-staged` then `pnpm run lint`, then optional `codegraph sync` (`.husky/pre-commit`).
- `lint-staged` applies Prettier to staged web/docs files and `gofmt -w` to staged Go files.
- Root `pnpm run format` only formats `*.{ts,tsx,md,mdx}`.
- Product/domain specs are under `.specs/features/*` and often in Spanish for DIAN/CUFE context; keep domain terms untranslated.
