# AGENTS

## Toolchain and workspace

- Use `mise install` after clone. Tool versions are pinned: Node `24`, Go `1.25`, pnpm `10.16.1` (`.mise.toml`, `.nvmrc`, `go.mod`, root `package.json`).
- Use `pnpm` only. This monorepo is defined by `pnpm-workspace.yaml` (`apps/*`, `packages/*`) and orchestrated by Turbo (`turbo.json`).

## High-value commands (root)

- `pnpm run dev` -> runs `infra:up` first (Docker) and then `turbo run dev`.
- `pnpm run lint`, `pnpm run test`, `pnpm run build` (workspace-wide via Turbo).
- `pnpm run migrate:controlplane | migrate:tenants | migrate:all` (delegates to `@bowerbird/backend`).
- `pnpm run deploy` -> always builds first, then deploys only `@bowerbird/infra`.
- Typical full verification before handoff: `pnpm run lint && pnpm run test && pnpm run build`.

## Package-scoped commands

- Backend: `pnpm --filter @bowerbird/backend dev|lint|test|build|migrate:all`.
- PWA: `pnpm --filter @bowerbird/pwa dev|lint|test|build`.
- Infra: `pnpm --filter @bowerbird/infra lint|test|build|synth|deploy`.

## Backend specifics (`apps/backend`)

- Local dev entrypoint is `cmd/api/main.go`; migrations CLI is `cmd/migrate/main.go`; Lambda entrypoints are under `cmd/lambda/*`.
- `pnpm --filter @bowerbird/backend dev` sources `apps/backend/.env` if present and runs `air -c .air.toml`.
- Runtime config is hybrid: env vars for process flags, then secrets loaded from SSM (`internal/platform/config/config.go`). Local default SSM key is `/bowerbird/local/secrets`.
- LocalStack init script (`apps/backend/scripts/init-localstack.sh`) seeds SQS, EventBridge, S3, and that SSM parameter from `apps/backend/secrets.json`.
- If you change `apps/backend/secrets.json`, re-run init in the container: `docker exec bowerbird-localstack /etc/localstack/init/ready.d/init-localstack.sh`.
- Migration folders are split and must stay split: `apps/backend/migrations/controlplane` and `apps/backend/migrations/tenant`.

## Frontend specifics (`apps/pwa`)

- Angular standalone app (zoneless) with app wiring in `src/app/app.config.ts` and routes in `src/app/app.routes.ts`.
- Dev server is `ng serve --host 0.0.0.0 --port 4200`; allowed host is `app.bowerbird.dev` (`angular.json`).
- Tenant routing is path-based in the interceptor: first non-global URL segment becomes `X-Tenant-ID` (`tenant.interceptor.ts`).
- Auth model is access token in SignalStore + refresh via cookie; `authInterceptor` auto-refreshes on `401` and retries requests.
- Shared UI primitives live in `src/styles.css` (`.card`, `.input-field`, `.btn-primary`, `.btn-secondary`); keep style updates aligned there.

## Local infra and DNS

- `docker-compose.yml` starts Postgres (`5432`), Redis (`6379`), LocalStack (`4566`), and Caddy (`80/443`).
- `Caddyfile` maps `app.bowerbird.dev -> :4200` and `api.bowerbird.dev -> :8080`.
- For realistic cookie/routing behavior, use those domains locally (with `/etc/hosts` entries), not plain localhost.

## Infra deploy constraints (`packages/infra`)

- CDK entrypoint is `packages/infra/bin/index.ts`; it requires `packages/infra/.env`.
- `ENV` and `AWS_ACCOUNT_ID` are mandatory; `AWS_REGION` must be `us-east-1` (enforced in code).
- Web deploy reads static files from `apps/pwa/dist/pwa/browser`; build web before deploy.
- `BucketDeployment` uses `prune: false` for both assets and entrypoints in `bowerbird-stack.ts`; do not flip casually.

## Git hooks and formatting

- Pre-commit hook runs: `pnpm lint-staged` -> `pnpm run lint` -> optional `codegraph sync` (`.husky/pre-commit`).
- `lint-staged` formats staged files with Prettier and runs `gofmt` for `*.go` (`.lintstagedrc.json`).
- Run `npm run format` (or `pnpm run format`) to format `.md`, `.ts`, and `.json` files manually.

## Documentation and Specifications

- **Product definitions**: `docs/product/` holds features and dictionary (Ubiquitous Language).
- **Technical docs**: `docs/technical/` holds ADRs and architecture guidelines.
- **Specifications for AI**: `.specs/features/` holds detailed requirements (`spec.md`, `design.md`). These are often written in Spanish to preserve domain fidelity for local accounting/tax concepts (e.g., DIAN, CUFE). Honor the language and do not translate localized domain terms to English.
