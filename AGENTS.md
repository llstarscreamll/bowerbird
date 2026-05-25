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
- Infra targeted: `pnpm --filter @bowerbird/infra lint|test|build|synth|deploy`.

## Backend (`apps/backend`)

- API entrypoint is `cmd/api/main.go`; local `dev` uses Air (`.air.toml`) and sources `apps/backend/.env` if present.
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
- Tenant header is derived from first non-global path segment in `core/interceptors/tenant.interceptor.ts`.
- Auth refresh behavior is in `core/interceptors/auth.interceptor.ts` (401 -> refresh -> retry).
- Feature convention: keep business/data orchestration in `*/application/*store.ts`; keep `presentation` components thin.
- Shared inbox/provider primitives now live in `src/app/core/domain/inbox-types.ts` (avoid cross-feature domain imports for these types).
- Shared visual primitives are in `src/styles.css` (`.card`, `.input-field`, `.btn-primary`, `.btn-secondary`).

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
