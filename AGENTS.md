# AGENTS

## Repo essentials

- Use Mise for tool installation and version management (`.mise.toml`). Run `mise install` after clone.
- Node.js runtime is pinned to `24.x` (`package.json` engines + `.nvmrc`).
- Go runtime is pinned to `1.25.x` (`.mise.toml` + `apps/backend/go.mod`).
- Package manager is `pnpm` (`packageManager: pnpm@10.16.1`). Prefer `pnpm` commands; do not switch to npm.
- Monorepo layout is fixed by `pnpm-workspace.yaml`: `apps/*` and `packages/*`.
- Task orchestration is through Turborepo (`turbo.json`). Root scripts are the source of truth.
- CodeGraph is configured for codebase structural exploration. Run `codegraph init -i` if index is missing or out of date.

## Multi-Tenancy & Identity Architecture (CRITICAL)

This system uses a **Database-per-tenant** architecture with a central **Control Plane**.

- **Control Plane DB:** Holds global identity (`users`, `user_identities`), the `tenants` catalog, and `tenant_memberships` (roles: OWNER, ADMIN, MEMBER). **No personal data** (names, avatars) is stored here.
- **Tenant DBs:** Each organization gets its own physical PostgreSQL database (e.g. `tenant_acme`). Contains domain data and user _profiles_ (local `users` table with `first_name`, `last_name`, `avatar_url`).
- **Migrations split:** Migrations MUST be placed in either `apps/backend/migrations/controlplane` or `apps/backend/migrations/tenant`. Never mix them.
- **Tenant Resolution:** The Angular frontend (`apps/pwa`) runs on a single domain (e.g. `app.bowerbird.com` / `app.bowerbird.dev`) and uses path-based routing (`/:tenant/dashboard`). It extracts the tenant slug from the URL path and sends it as the `X-Tenant-ID` header. The Go backend reads this header to route DB connections dynamically via `Registry`.
- **Authentication Strategy:** Mixed session approach. Backend returns short-lived Access Tokens in the JSON body (stored in Angular memory/SignalStore to prevent XSS) and a long-lived Refresh Token in an `HttpOnly`, `Secure` cookie. Automatic refresh is handled by the `authInterceptor` in Angular.

## High-value commands

- Install deps: `pnpm install`
- Full dev stack: `pnpm run dev` (starts Docker infra, API dev server, Angular dev server)
- **Database Migrations**:
  - `pnpm run migrate:all` (Migrates Control Plane, then iterates and migrates all active Tenant databases)
  - `pnpm run migrate:controlplane` (Migrates only the central DB)
  - `pnpm run migrate:tenants` (Migrates only the tenant DBs)
- Format: `pnpm run format` (Prettier on TS/Markdown, etc.)
- Full checks: `pnpm run lint && pnpm run test && pnpm run build`
- Deploy (root): `pnpm run deploy` (runs build first, then `@bowerbird/infra` deploy)

## Targeted package commands

- API only: `pnpm --filter @bowerbird/backend dev|lint|test|build`
- Web only: `pnpm --filter @bowerbird/pwa dev|lint|test|build`
- Infra only: `pnpm --filter @bowerbird/infra lint|build|deploy|synth`

## Runtime + setup gotchas

- API environment separation: `.env` handles process/infra flags, `secrets.json` handles AWS SSM secrets (DB URLs, Queues, etc.).
- API dev uses Air live reload via `apps/backend/.air.toml`. Air must be installed on host. The `apps/backend/package.json` dev script uses `exec air` to ensure graceful shutdown on `SIGINT` (prevents zombie port 8080 processes).
- Local dependencies: Postgres (`5432`), Redis (`6379`), LocalStack (`4566`) from `docker-compose.yml`.
- Local AWS resources (S3/SQS/EventBridge/SSM) are auto-created by `apps/backend/scripts/init-localstack.sh` reading from `secrets.json`.

## Package boundaries and entrypoints

- `apps/backend`
  - HTTP entrypoint: `cmd/api/main.go`. Custom CLI entrypoint: `cmd/migrate/main.go`. Lambda entrypoints: `cmd/lambda/*`.
  - Asymmetric architecture:
    - `internal/platform/`: Flat structure for technical adapters (AWS, DB `Registry` for multi-tenant pools, Config, Middlewares).
    - `internal/<domain>/`: Strict Clean Architecture/DDD (domain, application, infrastructure, presentation).
  - Strict Rules: `platform` never imports domains. Domains never import each other directly. `main.go` is the exclusive place for dependency injection.
- `apps/pwa`
  - Angular 21 Zoneless standalone app (`apps/pwa/angular.json`, `apps/pwa/src/app/app.config.ts`).
  - Uses `@ngrx/signals` (SignalStore) for global state management. Avoid NgRx/Redux.
  - Tests use Vitest via `@angular/build:unit-test`.
  - Dev proxy uses local DNS (`Caddyfile` + `/etc/hosts`) mapping `app.bowerbird.dev` to `4200` and `api.bowerbird.dev` to `8080`.
- `packages/infra`
  - CDK app entrypoint: `packages/infra/bin/index.ts`
  - Main stack: `packages/infra/bin/bowerbird-stack.ts` (Handles SSL certs and CloudFront CORS).

## Deploy-specific constraints

- `packages/infra/.env` is required (`AWS_ACCOUNT_ID`, domain vars, etc.).
- Infra stack packages frontend from local build output: `apps/pwa/dist/pwa/browser`; run web/root build before deploy.
- CloudFront/S3 deploy strategy intentionally keeps old hashed assets (`prune: false`). Do not change this casually.
